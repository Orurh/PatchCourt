package app

import (
	"context"
	"fmt"

	"github.com/orurh/patchcourt/internal/analysis/contracts"
	"github.com/orurh/patchcourt/internal/analysis/depdiff"
	"github.com/orurh/patchcourt/internal/analysis/findingdiff"
	"github.com/orurh/patchcourt/internal/analysis/risk"
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

	contractChanges := contracts.DiffSymbols(beforeProject.Symbols, afterProject.Symbols)
	dependencyChanges := depdiff.DiffDependencies(beforeProject.Dependencies, afterProject.Dependencies)
	layerEdgeChanges := depdiff.DiffLayerEdges(beforeProject.Dependencies, afterProject.Dependencies)
	findingChanges := findingdiff.DiffFindings(beforeProject.Findings, afterProject.Findings)

	reviewRisk := risk.Calculate(risk.Input{
		ContractChanges:   contractChanges,
		DependencyChanges: dependencyChanges,
		LayerEdgeChanges:  layerEdgeChanges,
		FindingChanges:    findingChanges,
	})

	result := &ReviewResult{
		Summary:           buildReviewSummary(contractChanges, dependencyChanges, layerEdgeChanges, findingChanges),
		Risk:              reviewRisk,
		ContractChanges:   contractChanges,
		DependencyChanges: dependencyChanges,
		LayerEdgeChanges:  layerEdgeChanges,
		FindingChanges:    findingChanges,
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
