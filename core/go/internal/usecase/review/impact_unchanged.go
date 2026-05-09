package review

import (
	"fmt"

	findingdiff "github.com/orurh/patchcourt/internal/diff/finding"
	"github.com/orurh/patchcourt/internal/model"
)

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
