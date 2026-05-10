package review

import (
	"strings"

	contracts "github.com/orurh/patchcourt/internal/diff/contract"
	"github.com/orurh/patchcourt/internal/model"
	"github.com/orurh/patchcourt/internal/reportmodel"
)

func worseContractChanges(changes []contracts.SymbolChange, impacts []reportmodel.ContractImpact) []ReviewImpactItem {
	return nil
}

func needsReviewContractChanges(changes []contracts.SymbolChange, impacts []reportmodel.ContractImpact) []ReviewImpactItem {
	items := make([]ReviewImpactItem, 0)
	impactIndex := contractImpactIndex(impacts)
	removedParents := removedContractParentNames(changes)
	groupedRemovedMethods := make(map[string]int)

	for _, change := range changes {
		impact := impactIndex[contractImpactKey(string(change.Kind), change.SymbolKey)]

		switch change.Kind {
		case contracts.ChangeKindRemoved:
			parentName, methodName := contractSymbolParts(change.SymbolKey)
			if methodName != "" {
				if _, ok := removedParents[parentName]; ok {
					if isWorseContractImpact(impact) {
						groupedRemovedMethods[parentName]++
					}
					continue
				}
			}

			items = append(items, contractNeedsReviewItem(change, impact))

		case contracts.ChangeKindAdded:
			if impact.Impact == contractImpactBreaking {
				items = append(items, ReviewImpactItem{
					Kind:       "contract_required_added",
					Title:      "Required public contract symbol added; verify implementation impact",
					Detail:     contractImpactDetail(change.SymbolKey, impact),
					Evidence:   contractChangeEvidence(change),
					Suggestion: "Review implementers and generated/public API consumers before treating this as a hard failure.",
				})
			}

		case contracts.ChangeKindSignatureChanged, contracts.ChangeKindModifiersChanged:
			items = append(items, contractNeedsReviewItem(change, impact))
		}
	}

	for parentName, methodCount := range groupedRemovedMethods {
		items = append(items, ReviewImpactItem{
			Kind:       "contract_boundary_delivery_impact",
			Title:      "Contract boundary changed with delivery/API impact",
			Detail:     parentName + " removed; " + intString(methodCount) + " removed methods have delivery/API impact",
			Suggestion: "Review whether delivery/API behavior was intentionally migrated to replacement contracts.",
		})
	}

	return items
}

func contractNeedsReviewItem(change contracts.SymbolChange, impact reportmodel.ContractImpact) ReviewImpactItem {
	if isWorseContractImpact(impact) {
		return ReviewImpactItem{
			Kind:       "contract_delivery_impact",
			Title:      "Public contract changed with delivery/API impact",
			Detail:     contractImpactDetail(change.SymbolKey, impact),
			Evidence:   contractChangeEvidence(change),
			Suggestion: "Review whether delivery/API behavior was intentionally migrated, compatibility was expected to break, or replacement contracts cover the old behavior.",
		}
	}

	return ReviewImpactItem{
		Kind:       "contract_boundary_changed",
		Title:      "Contract boundary changed; verify cleanup vs breaking API",
		Detail:     change.SymbolKey,
		Evidence:   contractChangeEvidence(change),
		Suggestion: "Confirm whether this is intentional abstraction cleanup, replacement contract introduction, or a breaking public API removal.",
	}
}

func removedContractParentNames(changes []contracts.SymbolChange) map[string]struct{} {
	parents := make(map[string]struct{})

	for _, change := range changes {
		if change.Kind != contracts.ChangeKindRemoved {
			continue
		}

		parentName := removedContractParentName(change.SymbolKey)
		if parentName == "" {
			continue
		}

		parents[parentName] = struct{}{}
	}

	return parents
}

func removedContractParentName(symbolKey string) string {
	parts := strings.Split(symbolKey, "::")
	if len(parts) != 2 {
		return ""
	}

	switch parts[0] {
	case "class", "struct":
		return parts[1]
	default:
		return ""
	}
}

func contractChangeEvidence(change contracts.SymbolChange) []model.Evidence {
	evidence := make([]model.Evidence, 0, 2)

	if change.Before != nil {
		evidence = append(evidence, model.Evidence{
			File:      change.Before.File,
			LineStart: change.Before.Line,
			Snippet:   change.Before.Signature,
			Message:   "before contract symbol",
		})
	}

	if change.After != nil {
		evidence = append(evidence, model.Evidence{
			File:      change.After.File,
			LineStart: change.After.Line,
			Snippet:   change.After.Signature,
			Message:   "after contract symbol",
		})
	}

	return evidence
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
