package review

import (
	"sort"

	"github.com/orurh/patchcourt/internal/model"
)

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
