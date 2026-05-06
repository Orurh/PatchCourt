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
	report.Worse = append(report.Worse, worseContractChanges(result.ContractChanges, result.ContractImpacts)...)

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
		switch change.Kind {
		case findingdiff.FindingChangeKindAdded:
			if change.After == nil {
				continue
			}

			finding := change.After
			items = append(items, ReviewImpactItem{
				Kind:       "finding_added",
				Severity:   string(finding.Severity),
				Title:      fmt.Sprintf("Added %s finding", model.HumanFindingKind(finding.Kind)),
				Detail:     finding.Title,
				ID:         change.ID,
				Risk:       finding.Risk,
				Suggestion: finding.Suggestion,
				Evidence:   finding.Evidence,
			})

		case findingdiff.FindingChangeKindChanged:
			if !findingChangeGotWorse(change) || change.After == nil {
				continue
			}

			finding := change.After
			items = append(items, ReviewImpactItem{
				Kind:       "finding_worsened",
				Severity:   string(finding.Severity),
				Title:      fmt.Sprintf("Existing %s finding worsened", model.HumanFindingKind(finding.Kind)),
				Detail:     changedFindingDetail(change),
				ID:         change.ID,
				Risk:       finding.Risk,
				Suggestion: finding.Suggestion,
				Evidence:   changedFindingEvidence(change, finding.Evidence),
			})
		}
	}

	return items
}

func betterFindingChanges(changes []findingdiff.FindingChange) []ReviewImpactItem {
	items := make([]ReviewImpactItem, 0)

	for _, change := range changes {
		switch change.Kind {
		case findingdiff.FindingChangeKindRemoved:
			if change.Before == nil {
				continue
			}

			finding := change.Before
			items = append(items, ReviewImpactItem{
				Kind:       "finding_removed",
				Severity:   string(finding.Severity),
				Title:      fmt.Sprintf("Removed %s finding", model.HumanFindingKind(finding.Kind)),
				Detail:     finding.Title,
				ID:         change.ID,
				Risk:       finding.Risk,
				Suggestion: finding.Suggestion,
				Evidence:   finding.Evidence,
			})

		case findingdiff.FindingChangeKindChanged:
			if !findingChangeGotBetter(change) || change.Before == nil {
				continue
			}

			finding := change.Before
			items = append(items, ReviewImpactItem{
				Kind:       "finding_weakened",
				Severity:   string(finding.Severity),
				Title:      fmt.Sprintf("Existing %s finding weakened", model.HumanFindingKind(finding.Kind)),
				Detail:     changedFindingDetail(change),
				ID:         change.ID,
				Risk:       finding.Risk,
				Suggestion: finding.Suggestion,
				Evidence:   change.RemovedEvidence,
			})
		}
	}

	return items
}

func findingChangeGotWorse(change findingdiff.FindingChange) bool {
	if change.Before == nil || change.After == nil {
		return false
	}

	if model.SeverityRank(change.After.Severity) > model.SeverityRank(change.Before.Severity) {
		return true
	}

	if len(change.AddedEvidence) > len(change.RemovedEvidence) {
		return true
	}

	return confidenceRank(change.After.Confidence) > confidenceRank(change.Before.Confidence)
}

func findingChangeGotBetter(change findingdiff.FindingChange) bool {
	if change.Before == nil || change.After == nil {
		return false
	}

	if model.SeverityRank(change.After.Severity) < model.SeverityRank(change.Before.Severity) {
		return true
	}

	if len(change.RemovedEvidence) > len(change.AddedEvidence) {
		return true
	}

	return confidenceRank(change.After.Confidence) < confidenceRank(change.Before.Confidence)
}

func changedFindingDetail(change findingdiff.FindingChange) string {
	parts := make([]string, 0, 4)

	if change.Before != nil && change.After != nil && change.Before.Severity != change.After.Severity {
		parts = append(parts, fmt.Sprintf("severity %s -> %s", change.Before.Severity, change.After.Severity))
	}

	if change.Before != nil && change.After != nil && change.Before.Confidence != change.After.Confidence {
		parts = append(parts, fmt.Sprintf("confidence %s -> %s", change.Before.Confidence, change.After.Confidence))
	}

	if len(change.AddedEvidence) > 0 {
		parts = append(parts, fmt.Sprintf("+%d evidence", len(change.AddedEvidence)))
	}

	if len(change.RemovedEvidence) > 0 {
		parts = append(parts, fmt.Sprintf("-%d evidence", len(change.RemovedEvidence)))
	}

	if len(parts) == 0 && change.After != nil {
		return change.After.Title
	}

	return fmt.Sprintf("%s: %s", changeTitle(change), joinDetails(parts))
}

func changedFindingEvidence(change findingdiff.FindingChange, fallback []model.Evidence) []model.Evidence {
	if len(change.AddedEvidence) > 0 {
		return change.AddedEvidence
	}

	return fallback
}

func changeTitle(change findingdiff.FindingChange) string {
	if change.After != nil && change.After.Title != "" {
		return change.After.Title
	}

	if change.Before != nil && change.Before.Title != "" {
		return change.Before.Title
	}

	return "finding changed"
}

func joinDetails(parts []string) string {
	if len(parts) == 0 {
		return ""
	}

	result := parts[0]
	for i := 1; i < len(parts); i++ {
		result += ", " + parts[i]
	}

	return result
}

func confidenceRank(confidence model.Confidence) int {
	switch confidence {
	case model.ConfidenceHigh:
		return 3
	case model.ConfidenceMedium:
		return 2
	case model.ConfidenceLow:
		return 1
	default:
		return 0
	}
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

func worseContractChanges(changes []contracts.SymbolChange, impacts []reportmodel.ContractImpact) []ReviewImpactItem {
	items := make([]ReviewImpactItem, 0)
	impactIndex := contractImpactIndex(impacts)

	for _, change := range changes {
		impact := impactIndex[contractImpactKey(string(change.Kind), change.SymbolKey)]

		switch change.Kind {
		case contracts.ChangeKindRemoved:
			items = append(items, ReviewImpactItem{
				Kind:   "contract_removed",
				Title:  "Removed public contract symbol",
				Detail: change.SymbolKey,
			})

		case contracts.ChangeKindAdded:
			if impact.Impact == "breaking" {
				items = append(items, ReviewImpactItem{
					Kind:   "contract_required_added",
					Title:  "Added required public contract symbol",
					Detail: contractImpactDetail(change.SymbolKey, impact),
				})
			}

		case contracts.ChangeKindSignatureChanged:
			if isWorseContractImpact(impact) {
				items = append(items, ReviewImpactItem{
					Kind:   "contract_signature_changed",
					Title:  "Changed public contract signature with impacted callers",
					Detail: contractImpactDetail(change.SymbolKey, impact),
				})
			}

		case contracts.ChangeKindModifiersChanged:
			if isWorseContractImpact(impact) {
				items = append(items, ReviewImpactItem{
					Kind:   "contract_modifiers_changed",
					Title:  "Changed public contract modifiers with impacted callers",
					Detail: contractImpactDetail(change.SymbolKey, impact),
				})
			}
		}
	}

	return items
}

func contractImpactIndex(impacts []reportmodel.ContractImpact) map[string]reportmodel.ContractImpact {
	index := make(map[string]reportmodel.ContractImpact, len(impacts))

	for _, impact := range impacts {
		index[contractImpactKey(impact.ChangeKind, impact.SymbolKey)] = impact
	}

	return index
}

func contractImpactKey(kind string, symbolKey string) string {
	return kind + "|" + symbolKey
}

func isWorseContractImpact(impact reportmodel.ContractImpact) bool {
	if impact.SymbolKey == "" {
		return false
	}

	if impact.DeliveryImpacted {
		return true
	}

	return len(impact.ImpactedFiles) > 0 && !impact.TestsChanged
}

func contractImpactDetail(symbolKey string, impact reportmodel.ContractImpact) string {
	if impact.SymbolKey == "" {
		return symbolKey
	}

	detail := symbolKey

	if impact.DeliveryImpacted {
		detail += " (delivery/API impacted)"
	}

	if !impact.TestsChanged {
		detail += " (no test-like files changed)"
	}

	return detail
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
	case "contract_removed", "contract_required_added", "contract_signature_changed":
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
			Kind:       "unchanged_finding",
			Severity:   string(after.Severity),
			Title:      fmt.Sprintf("Existing %s finding", model.HumanFindingKind(after.Kind)),
			Detail:     after.Title,
			ID:         after.ID,
			Risk:       after.Risk,
			Suggestion: after.Suggestion,
			Evidence:   append([]model.Evidence(nil), after.Evidence...),
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
