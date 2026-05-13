package review

import (
	"context"
	"fmt"

	"github.com/orurh/patchcourt/internal/diff/contract"
	"github.com/orurh/patchcourt/internal/diff/dep"
	"github.com/orurh/patchcourt/internal/diff/finding"
	projectdiff "github.com/orurh/patchcourt/internal/diff/project"
	"github.com/orurh/patchcourt/internal/model"
	"github.com/orurh/patchcourt/internal/reportmodel"
	"github.com/orurh/patchcourt/internal/reviewrisk"
	"github.com/orurh/patchcourt/internal/state"
	"github.com/orurh/patchcourt/internal/usecase/confighealth"
)

type ReviewSummary = reportmodel.ReviewSummary
type ReviewResult = reportmodel.ReviewResult

type Format string

const (
	FormatText     Format = "text"
	FormatJSON     Format = "json"
	FormatMarkdown Format = "markdown"
)

type Request struct {
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

type Service struct {
	Projects ProjectLoader
}

func NewService(projects ProjectLoader) Service {
	return Service{
		Projects: projects,
	}
}

func (s Service) Run(ctx context.Context, req Request) (*ReviewResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("review canceled before start: %w", err)
	}

	beforeProject, afterProject, gitChangedFiles, err := s.Projects.LoadProjects(ctx, req)
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

	graphNodeCount, graphEdgeCount := reviewLayerGraphSize(afterProject)

	result := &ReviewResult{
		BeforeProject:     beforeProject,
		AfterProject:      afterProject,
		SchemaVersion:     reportmodel.ReviewResultSchemaVersion,
		Summary:           buildReviewSummary(changeSet.ContractChanges, changeSet.DependencyChanges, changeSet.LayerEdgeChanges, changeSet.FindingChanges),
		Risk:              reviewRisk,
		ConfigHealth:      confighealth.Build(afterProject, req.ConfigPath, graphNodeCount, graphEdgeCount),
		ChangedFiles:      changeSet.ChangedFiles,
		ContractChanges:   changeSet.ContractChanges,
		ContractImpacts:   BuildContractImpacts(changeSet.ContractChanges, afterProject, changeSet.ChangedFiles),
		DependencyChanges: changeSet.DependencyChanges,
		LayerEdgeChanges:  changeSet.LayerEdgeChanges,
		FindingChanges:    changeSet.FindingChanges,
	}

	result.Impact = BuildImpactReport(result, beforeProject, afterProject)

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

		case findingdiff.FindingChangeKindChanged:
			if changedFindingGotWorseForSummary(change) && change.After != nil {
				if isHighOrCritical(change.After.Severity) {
					summary.AddedHighFindings++
				}

				if change.After.Kind == model.FindingKindPolicyViolation {
					summary.AddedPolicyFindings++
				}
			}
		}
	}

	return summary
}

func changedFindingGotWorseForSummary(change findingdiff.FindingChange) bool {
	if change.Before == nil || change.After == nil {
		return false
	}

	if model.SeverityRank(change.After.Severity) > model.SeverityRank(change.Before.Severity) {
		return true
	}

	return len(change.AddedEvidence) > len(change.RemovedEvidence)
}

func isHighOrCritical(severity model.Severity) bool {
	return severity == model.SeverityHigh || severity == model.SeverityCritical
}

func reviewLayerGraphSize(project *model.ProjectModel) (int, int) {
	if project == nil {
		return 0, 0
	}

	nodes := make(map[string]struct{})
	edges := make(map[string]struct{})

	for _, dependency := range project.Dependencies {
		if dependency.External || !dependency.Resolved {
			continue
		}
		if dependency.FromLayer == "" || dependency.ToLayer == "" {
			continue
		}
		if dependency.FromLayer == dependency.ToLayer {
			continue
		}

		nodes[dependency.FromLayer] = struct{}{}
		nodes[dependency.ToLayer] = struct{}{}
		edges[dependency.FromLayer+"\x00"+dependency.ToLayer] = struct{}{}
	}

	return len(nodes), len(edges)
}
