package review

import (
	contracts "github.com/orurh/patchcourt/internal/diff/contract"
	"github.com/orurh/patchcourt/internal/reportmodel"
)

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
