package reviewcontract

import (
	"fmt"
	"strings"

	contracts "github.com/orurh/patchcourt/internal/diff/contract"
)

type Impact string

const (
	ImpactBreaking      Impact = "breaking"
	ImpactRisky         Impact = "risky"
	ImpactAdditive      Impact = "additive"
	ImpactInformational Impact = "informational"
)

func ClassifyImpact(change contracts.SymbolChange) Impact {
	switch change.Kind {
	case contracts.ChangeKindRemoved:
		return ImpactBreaking

	case contracts.ChangeKindSignatureChanged:
		return ImpactBreaking

	case contracts.ChangeKindAdded:
		return ImpactAdditive

	case contracts.ChangeKindModifiersChanged:
		if containsString(change.AddedMods, "pure_virtual") {
			return ImpactBreaking
		}

		if containsAnyString(change.RemovedMods, []string{
			"virtual",
			"const",
			"noexcept",
			"override",
			"final",
			"pure_virtual",
		}) {
			return ImpactRisky
		}

		if containsAnyString(change.AddedMods, []string{
			"final",
			"override",
			"noexcept",
		}) {
			return ImpactRisky
		}

		return ImpactInformational

	default:
		return ImpactInformational
	}
}

func Location(change contracts.SymbolChange) string {
	file := ""
	if change.After != nil {
		file = change.After.File
	}
	if file == "" && change.Before != nil {
		file = change.Before.File
	}
	if file == "" {
		return ""
	}

	beforeLine := 0
	if change.Before != nil {
		beforeLine = change.Before.Line
	}

	afterLine := 0
	if change.After != nil {
		afterLine = change.After.Line
	}

	switch {
	case beforeLine > 0 && afterLine > 0 && beforeLine != afterLine:
		return fmt.Sprintf("%s:%d → %d", file, beforeLine, afterLine)
	case afterLine > 0:
		return fmt.Sprintf("%s:%d", file, afterLine)
	case beforeLine > 0:
		return fmt.Sprintf("%s:%d", file, beforeLine)
	default:
		return file
	}
}

func Modifiers(change contracts.SymbolChange) string {
	parts := make([]string, 0, 2)

	if len(change.AddedMods) > 0 {
		parts = append(parts, "added: "+strings.Join(change.AddedMods, ", "))
	}

	if len(change.RemovedMods) > 0 {
		parts = append(parts, "removed: "+strings.Join(change.RemovedMods, ", "))
	}

	return strings.Join(parts, "; ")
}

func containsAnyString(values []string, targets []string) bool {
	for _, target := range targets {
		if containsString(values, target) {
			return true
		}
	}

	return false
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}

	return false
}
