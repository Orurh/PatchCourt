package app

import (
	"context"
	"fmt"

	"github.com/orurh/patchcourt/internal/analysis/contracts"
	"github.com/orurh/patchcourt/internal/analysis/depdiff"
	"github.com/orurh/patchcourt/internal/analysis/findingdiff"
	"github.com/orurh/patchcourt/internal/analysis/risk"
	"github.com/orurh/patchcourt/internal/changes"
	"github.com/orurh/patchcourt/internal/model"
)

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

type ReviewSummary struct {
	ContractChanges     int `json:"contract_changes"`
	DependencyChanges   int `json:"dependency_changes"`
	LayerEdgeChanges    int `json:"layer_edge_changes"`
	FindingChanges      int `json:"finding_changes"`
	AddedFindings       int `json:"added_findings"`
	RemovedFindings     int `json:"removed_findings"`
	AddedHighFindings   int `json:"added_high_findings"`
	AddedPolicyFindings int `json:"added_policy_findings"`
}

type ReviewResult struct {
	Summary           ReviewSummary               `json:"summary"`
	Risk              risk.Score                  `json:"risk"`
	Impact            ReviewImpactReport          `json:"impact"`
	ContractChanges   []contracts.SymbolChange    `json:"contract_changes"`
	DependencyChanges []depdiff.DependencyChange  `json:"dependency_changes"`
	LayerEdgeChanges  []depdiff.LayerEdgeChange   `json:"layer_edge_changes"`
	FindingChanges    []findingdiff.FindingChange `json:"finding_changes"`
}

func (a *App) RunReview(ctx context.Context, req ReviewRequest) (*ReviewResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("review canceled before start: %w", err)
	}

	beforeProject, afterProject, err := a.loadReviewProjects(ctx, req)
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

		if _, err := changes.SaveState(changes.SaveStateOptions{
			Root:       root,
			ConfigPath: req.ConfigPath,
			Project:    afterProject,
		}); err != nil {
			return nil, fmt.Errorf("update state: %w", err)
		}
	}

	changeSet := changes.Compare(beforeProject, afterProject)

	result := &ReviewResult{
		Summary:           buildReviewSummary(changeSet.ContractChanges, changeSet.DependencyChanges, changeSet.LayerEdgeChanges, changeSet.FindingChanges),
		Risk:              changeSet.Risk,
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
