package app

import (
	"context"
	"fmt"
	"github.com/orurh/patchcourt/internal/reportmodel"

	"github.com/orurh/patchcourt/internal/analysis/contracts"
	"github.com/orurh/patchcourt/internal/analysis/depdiff"
	"github.com/orurh/patchcourt/internal/analysis/findingdiff"
	"github.com/orurh/patchcourt/internal/analysis/risk"
	projectdiff "github.com/orurh/patchcourt/internal/diff/project"
	"github.com/orurh/patchcourt/internal/model"
	"github.com/orurh/patchcourt/internal/state"
)

type ReviewSummary = reportmodel.ReviewSummary
type ReviewResult = reportmodel.ReviewResult

type ReviewFormat string

const (
	ReviewFormatText     ReviewFormat = "text"
	ReviewFormatJSON     ReviewFormat = "json"
	ReviewFormatMarkdown ReviewFormat = "markdown"
)

type ReviewRequest struct {
	BeforePath string
	AfterPath  string

	BeforeRoot string
	AfterRoot  string
	ConfigPath string

	SinceLastRoot string
	UpdateState   bool

	GitRoot  string
	BaseRef  string
	HeadRef  string
	Worktree bool
}

func (a *App) RunReview(ctx context.Context, req ReviewRequest) (*ReviewResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("review canceled before start: %w", err)
	}

	beforeProject, afterProject, gitChangedFiles, err := a.loadReviewProjects(ctx, req)
	if err != nil {
		return nil, err
	}

	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("review canceled after loading project models: %w", err)
	}

	if req.UpdateState {
		root := req.SinceLastRoot
		if root == "" {
			root = req.AfterRoot
		}

		if root == "" {
			return nil, fmt.Errorf("--update-state requires --since-last or --after-root")
		}

		if _, err := state.SaveState(state.SaveStateOptions{
			Root:       root,
			ConfigPath: req.ConfigPath,
			Project:    afterProject,
		}); err != nil {
			return nil, fmt.Errorf("update state: %w", err)
		}
	}

	changeSet := projectdiff.Compare(beforeProject, afterProject)
	changeSet.ChangedFiles = projectdiff.MergeChangedFiles(changeSet.ChangedFiles, gitChangedFiles)
	policyIndex := buildPolicyViolationEdgeIndex(afterProject)
	reviewRisk := risk.Calculate(risk.Input{
		ContractChanges:   changeSet.ContractChanges,
		DependencyChanges: policyRelevantRiskDependencyChanges(changeSet.DependencyChanges, policyIndex),
		LayerEdgeChanges:  policyRelevantRiskLayerEdgeChanges(changeSet.LayerEdgeChanges, policyIndex),
		FindingChanges:    changeSet.FindingChanges,
	})

	result := &ReviewResult{
		SchemaVersion:     reportmodel.ReviewResultSchemaVersion,
		Summary:           buildReviewSummary(changeSet.ContractChanges, changeSet.DependencyChanges, changeSet.LayerEdgeChanges, changeSet.FindingChanges),
		Risk:              reviewRisk,
		ChangedFiles:      changeSet.ChangedFiles,
		ContractChanges:   changeSet.ContractChanges,
		DependencyChanges: changeSet.DependencyChanges,
		LayerEdgeChanges:  changeSet.LayerEdgeChanges,
		FindingChanges:    changeSet.FindingChanges,
	}

	result.Impact = BuildReviewImpactReport(result, beforeProject, afterProject)

	return result, nil
}

func buildReviewSummary(
	contractChanges []contracts.SymbolChange,
	dependencyChanges []depdiff.DependencyChange,
	layerEdgeChanges []depdiff.LayerEdgeChange,
	findingChanges []findingdiff.FindingChange,
) ReviewSummary {
	summary := ReviewSummary{
		ContractChanges:   len(contractChanges),
		DependencyChanges: len(dependencyChanges),
		LayerEdgeChanges:  len(layerEdgeChanges),
		FindingChanges:    len(findingChanges),
	}

	for _, change := range findingChanges {
		switch change.Kind {
		case findingdiff.FindingChangeKindAdded:
			summary.AddedFindings++

			if change.After != nil {
				if isHighOrCritical(change.After.Severity) {
					summary.AddedHighFindings++
				}

				if change.After.Kind == model.FindingKindPolicyViolation {
					summary.AddedPolicyFindings++
				}
			}

		case findingdiff.FindingChangeKindRemoved:
			summary.RemovedFindings++
		}
	}

	return summary
}

func isHighOrCritical(severity model.Severity) bool {
	return severity == model.SeverityHigh || severity == model.SeverityCritical
}
