package contracts

import (
	"sort"
	"strings"

	"github.com/orurh/patchcourt/internal/model"
)

type ChangeKind string

const (
	ChangeKindAdded            ChangeKind = "added"
	ChangeKindRemoved          ChangeKind = "removed"
	ChangeKindSignatureChanged ChangeKind = "signature_changed"
	ChangeKindModifiersChanged ChangeKind = "modifiers_changed"
)

type SymbolChange struct {
	Kind        ChangeKind         `json:"kind"`
	SymbolKey   string             `json:"symbol_key"`
	Before      *model.SymbolModel `json:"before,omitempty"`
	After       *model.SymbolModel `json:"after,omitempty"`
	AddedMods   []string           `json:"added_modifiers,omitempty"`
	RemovedMods []string           `json:"removed_modifiers,omitempty"`
}

func DiffSymbols(before []model.SymbolModel, after []model.SymbolModel) []SymbolChange {
	beforeIndex := indexExportedSymbols(before)
	afterIndex := indexExportedSymbols(after)

	changes := make([]SymbolChange, 0)

	keys := mergedSortedKeys(beforeIndex, afterIndex)
	for _, key := range keys {
		beforeSymbol, hadBefore := beforeIndex[key]
		afterSymbol, hasAfter := afterIndex[key]

		switch {
		case !hadBefore && hasAfter:
			afterCopy := afterSymbol
			changes = append(changes, SymbolChange{
				Kind:      ChangeKindAdded,
				SymbolKey: key,
				After:     &afterCopy,
			})

		case hadBefore && !hasAfter:
			beforeCopy := beforeSymbol
			changes = append(changes, SymbolChange{
				Kind:      ChangeKindRemoved,
				SymbolKey: key,
				Before:    &beforeCopy,
			})

		case hadBefore && hasAfter:
			if normalizeSignature(beforeSymbol.Signature) != normalizeSignature(afterSymbol.Signature) {
				beforeCopy := beforeSymbol
				afterCopy := afterSymbol
				changes = append(changes, SymbolChange{
					Kind:      ChangeKindSignatureChanged,
					SymbolKey: key,
					Before:    &beforeCopy,
					After:     &afterCopy,
				})
			}

			addedMods, removedMods := diffStringSet(beforeSymbol.Modifiers, afterSymbol.Modifiers)
			if len(addedMods) > 0 || len(removedMods) > 0 {
				beforeCopy := beforeSymbol
				afterCopy := afterSymbol
				changes = append(changes, SymbolChange{
					Kind:        ChangeKindModifiersChanged,
					SymbolKey:   key,
					Before:      &beforeCopy,
					After:       &afterCopy,
					AddedMods:   addedMods,
					RemovedMods: removedMods,
				})
			}
		}
	}

	return changes
}

func indexExportedSymbols(symbols []model.SymbolModel) map[string]model.SymbolModel {
	index := make(map[string]model.SymbolModel)

	for _, symbol := range symbols {
		if !symbol.Exported {
			continue
		}

		key := SymbolKey(symbol)
		if key == "" {
			continue
		}

		index[key] = symbol
	}

	return index
}

func SymbolKey(symbol model.SymbolModel) string {
	parts := []string{
		string(symbol.Kind),
		symbol.Parent,
		symbol.Name,
	}

	return strings.Join(parts, "::")
}

func mergedSortedKeys(left map[string]model.SymbolModel, right map[string]model.SymbolModel) []string {
	seen := make(map[string]struct{}, len(left)+len(right))

	for key := range left {
		seen[key] = struct{}{}
	}

	for key := range right {
		seen[key] = struct{}{}
	}

	keys := make([]string, 0, len(seen))
	for key := range seen {
		keys = append(keys, key)
	}

	sort.Strings(keys)
	return keys
}

func normalizeSignature(signature string) string {
	return strings.Join(strings.Fields(signature), " ")
}

func diffStringSet(before []string, after []string) (added []string, removed []string) {
	beforeSet := make(map[string]struct{}, len(before))
	afterSet := make(map[string]struct{}, len(after))

	for _, value := range before {
		beforeSet[value] = struct{}{}
	}

	for _, value := range after {
		afterSet[value] = struct{}{}
	}

	for value := range afterSet {
		if _, ok := beforeSet[value]; !ok {
			added = append(added, value)
		}
	}

	for value := range beforeSet {
		if _, ok := afterSet[value]; !ok {
			removed = append(removed, value)
		}
	}

	sort.Strings(added)
	sort.Strings(removed)

	return added, removed
}
