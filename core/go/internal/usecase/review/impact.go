package review

import (
	"fmt"
	"github.com/orurh/patchcourt/internal/reportmodel"
	"sort"

	"github.com/orurh/patchcourt/internal/diff/contract"
	"github.com/orurh/patchcourt/internal/diff/dep"
	"github.com/orurh/patchcourt/internal/diff/finding"
	"github.com/orurh/patchcourt/internal/model"
)

type ReviewImpactReport = reportmodel.ReviewImpactReport
type ReviewImpactItem = reportmodel.ReviewImpactItem

func BuildImpactReport(result *ReviewResult, beforeProject *model.ProjectModel, afterProject *model.ProjectModel) ReviewImpactReport {
	if result == nil {
		return ReviewImpactReport{}
	}

	report := ReviewImpactReport{
		Worse:         make([]ReviewImpactItem, 0),
		Better:        make([]ReviewImpactItem, 0),
		UnchangedDebt: make([]ReviewImpactItem, 0),
	}

	policyIndex := buildPolicyViolationEdgeIndex(afterProject)

	report.Worse = append(report.Worse, worseFindingChanges(result.FindingChanges)...)
	report.Worse = append(report.Worse, worseLayerEdgeChanges(result.LayerEdgeChanges, policyIndex)...)
	report.Worse = append(report.Worse, worseDependencyChanges(result.DependencyChanges, policyIndex)...)
	report.Worse = append(report.Worse, worseContractChanges(result.ContractChanges)...)

	report.Better = append(report.Better, betterFindingChanges(result.FindingChanges)...)
	report.Better = append(report.Better, betterLayerEdgeChanges(result.LayerEdgeChanges)...)
	report.Better = append(report.Better, betterDependencyChanges(result.DependencyChanges)...)

	report.UnchangedDebt = unchangedDebtImpact(beforeProject, afterProject, result.FindingChanges)

	sortImpactItems(report.Worse)
	sortImpactItems(report.Better)
	sortImpactItems(report.UnchangedDebt)

	return report
}

func worseFindingChanges(changes []findingdiff.FindingChange) []ReviewImpactItem {
	items := make([]ReviewImpactItem, 0)

	for _, change := range changes {
		if change.Kind != findingdiff.FindingChangeKindAdded || change.After == nil {
			continue
		}

		finding := change.After
		items = append(items, ReviewImpactItem{
			Kind:     "finding_added",
			Severity: string(finding.Severity),
			Title:    fmt.Sprintf("Added %s finding", model.HumanFindingKind(finding.Kind)),
			Detail:   finding.Title,
			ID:       change.ID,
		})
	}

	return items
}

func betterFindingChanges(changes []findingdiff.FindingChange) []ReviewImpactItem {
	items := make([]ReviewImpactItem, 0)

	for _, change := range changes {
		if change.Kind != findingdiff.FindingChangeKindRemoved || change.Before == nil {
			continue
		}

		finding := change.Before
		items = append(items, ReviewImpactItem{
			Kind:     "finding_removed",
			Severity: string(finding.Severity),
			Title:    fmt.Sprintf("Removed %s finding", model.HumanFindingKind(finding.Kind)),
			Detail:   finding.Title,
			ID:       change.ID,
		})
	}

	return items
}

func worseLayerEdgeChanges(changes []depdiff.LayerEdgeChange, policyIndex policyViolationEdgeIndex) []ReviewImpactItem {
	items := make([]ReviewImpactItem, 0)

	for _, change := range changes {
		switch change.Kind {
		case depdiff.DependencyChangeKindAdded:
			if !isPolicyViolationLayerEdge(policyIndex, change.FromLayer, change.ToLayer) {
				continue
			}

			items = append(items, ReviewImpactItem{
				Kind:   "layer_edge_added",
				Title:  "Added forbidden layer dependency",
				Detail: fmt.Sprintf("%s -> %s (%d)", change.FromLayer, change.ToLayer, change.AfterCount),
			})

		case depdiff.DependencyChangeKindChanged:
			if change.AfterCount <= change.BeforeCount {
				continue
			}

			if !isPolicyViolationLayerEdge(policyIndex, change.FromLayer, change.ToLayer) {
				continue
			}

			items = append(items, ReviewImpactItem{
				Kind:   "layer_edge_increased",
				Title:  "Increased forbidden layer dependency",
				Detail: fmt.Sprintf("%s -> %s (%d -> %d)", change.FromLayer, change.ToLayer, change.BeforeCount, change.AfterCount),
			})
		}
	}

	return items
}

func betterLayerEdgeChanges(changes []depdiff.LayerEdgeChange) []ReviewImpactItem {
	items := make([]ReviewImpactItem, 0)

	for _, change := range changes {
		switch change.Kind {
		case depdiff.DependencyChangeKindRemoved:
			items = append(items, ReviewImpactItem{
				Kind:   "layer_edge_removed",
				Title:  "Removed layer dependency",
				Detail: fmt.Sprintf("%s -> %s (%d)", change.FromLayer, change.ToLayer, change.BeforeCount),
			})

		case depdiff.DependencyChangeKindChanged:
			if change.AfterCount < change.BeforeCount {
				items = append(items, ReviewImpactItem{
					Kind:   "layer_edge_decreased",
					Title:  "Reduced layer dependency",
					Detail: fmt.Sprintf("%s -> %s (%d -> %d)", change.FromLayer, change.ToLayer, change.BeforeCount, change.AfterCount),
				})
			}
		}
	}

	return items
}

func worseDependencyChanges(changes []depdiff.DependencyChange, policyIndex policyViolationEdgeIndex) []ReviewImpactItem {
	items := make([]ReviewImpactItem, 0)

	for _, change := range changes {
		if change.Kind != depdiff.DependencyChangeKindAdded || change.After == nil {
			continue
		}

		dep := change.After
		if !isReviewRelevantDependency(*dep) {
			continue
		}

		if !isPolicyViolationDependency(policyIndex, *dep) {
			continue
		}

		items = append(items, ReviewImpactItem{
			Kind:   "dependency_added",
			Title:  "Added forbidden dependency",
			Detail: dependencyImpactDetail(*dep),
			ID:     change.Key,
		})
	}

	return items
}

func betterDependencyChanges(changes []depdiff.DependencyChange) []ReviewImpactItem {
	items := make([]ReviewImpactItem, 0)

	for _, change := range changes {
		if change.Kind != depdiff.DependencyChangeKindRemoved || change.Before == nil {
			continue
		}

		dep := change.Before
		if !isReviewRelevantDependency(*dep) {
			continue
		}

		items = append(items, ReviewImpactItem{
			Kind:   "dependency_removed",
			Title:  "Removed dependency",
			Detail: dependencyImpactDetail(*dep),
			ID:     change.Key,
		})
	}

	return items
}

func worseContractChanges(changes []contracts.SymbolChange) []ReviewImpactItem {
	items := make([]ReviewImpactItem, 0)

	for _, change := range changes {
		switch change.Kind {
		case contracts.ChangeKindRemoved:
			items = append(items, ReviewImpactItem{
				Kind:   "contract_removed",
				Title:  "Removed public contract symbol",
				Detail: change.SymbolKey,
			})
		case contracts.ChangeKindSignatureChanged:
			items = append(items, ReviewImpactItem{
				Kind:   "contract_signature_changed",
				Title:  "Changed public contract signature",
				Detail: change.SymbolKey,
			})
		case contracts.ChangeKindModifiersChanged:
			items = append(items, ReviewImpactItem{
				Kind:   "contract_modifiers_changed",
				Title:  "Changed public contract modifiers",
				Detail: change.SymbolKey,
			})
		}
	}

	return items
}

func isReviewRelevantDependency(dep model.DependencyEdge) bool {
	if dep.External {
		return false
	}

	if dep.FromLayer == "" || dep.ToLayer == "" {
		return false
	}

	if dep.FromLayer == dep.ToLayer {
		return false
	}

	return true
}

func dependencyImpactDetail(dep model.DependencyEdge) string {
	target := dep.ToFile
	if target == "" {
		target = dep.Target
	}

	if dep.FromLayer != "" || dep.ToLayer != "" {
		return fmt.Sprintf("%s -> %s (%s -> %s)", dep.FromFile, target, dep.FromLayer, dep.ToLayer)
	}

	return fmt.Sprintf("%s -> %s", dep.FromFile, target)
}

func sortImpactItems(items []ReviewImpactItem) {
	sort.SliceStable(items, func(i, j int) bool {
		leftRank := impactRank(items[i])
		rightRank := impactRank(items[j])
		if leftRank != rightRank {
			return leftRank > rightRank
		}

		if items[i].Kind != items[j].Kind {
			return items[i].Kind < items[j].Kind
		}

		if items[i].ID != items[j].ID {
			return items[i].ID < items[j].ID
		}

		return items[i].Detail < items[j].Detail
	})
}

func impactRank(item ReviewImpactItem) int {
	switch item.Kind {
	case "finding_added":
		return 100 + severityImpactRank(item.Severity)
	case "contract_removed", "contract_signature_changed":
		return 90
	case "layer_edge_added":
		return 80
	case "dependency_added":
		return 50
	case "finding_removed":
		return 40 + severityImpactRank(item.Severity)
	case "layer_edge_removed":
		return 30
	case "dependency_removed":
		return 20
	default:
		return 0
	}
}

func severityImpactRank(severity string) int {
	switch severity {
	case string(model.SeverityCritical):
		return 4
	case string(model.SeverityHigh):
		return 3
	case string(model.SeverityMedium):
		return 2
	case string(model.SeverityLow):
		return 1
	default:
		return 0
	}
}

const defaultUnchangedDebtLimit = 10

func unchangedDebtImpact(beforeProject *model.ProjectModel, afterProject *model.ProjectModel, changes []findingdiff.FindingChange) []ReviewImpactItem {
	if beforeProject == nil || afterProject == nil {
		return nil
	}

	changed := changedFindingIDs(changes)
	beforeByID := findingsByID(beforeProject.Findings)

	items := make([]ReviewImpactItem, 0)

	for _, after := range afterProject.Findings {
		if _, ok := changed[after.ID]; ok {
			continue
		}

		before, ok := beforeByID[after.ID]
		if !ok {
			continue
		}

		if before.Kind != after.Kind || before.Severity != after.Severity {
			continue
		}

		items = append(items, ReviewImpactItem{
			Kind:     "unchanged_finding",
			Severity: string(after.Severity),
			Title:    fmt.Sprintf("Existing %s finding", model.HumanFindingKind(after.Kind)),
			Detail:   after.Title,
			ID:       after.ID,
		})
	}

	sortImpactItems(items)

	if len(items) > defaultUnchangedDebtLimit {
		return items[:defaultUnchangedDebtLimit]
	}

	return items
}

func changedFindingIDs(changes []findingdiff.FindingChange) map[string]struct{} {
	result := make(map[string]struct{}, len(changes))

	for _, change := range changes {
		if change.ID == "" {
			continue
		}

		result[change.ID] = struct{}{}
	}

	return result
}

func findingsByID(findings []model.Finding) map[string]model.Finding {
	result := make(map[string]model.Finding, len(findings))

	for _, finding := range findings {
		if finding.ID == "" {
			continue
		}

		result[finding.ID] = finding
	}

	return result
}
