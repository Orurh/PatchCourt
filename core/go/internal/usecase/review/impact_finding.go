package review

import (
	"fmt"

	findingdiff "github.com/orurh/patchcourt/internal/diff/finding"
	"github.com/orurh/patchcourt/internal/model"
)

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
