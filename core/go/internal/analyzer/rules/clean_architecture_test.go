package rules

import (
	"testing"

	"github.com/orurh/patchcourt/internal/model"
	"github.com/stretchr/testify/require"
)

func TestArchitectureSuggestionText_CleanArchitectureDomainDependsOutward(t *testing.T) {
	got := architectureSuggestionText("domain", "infrastructure", map[model.DependencyKind]struct{}{
		model.DependencyKindInclude: {},
	})

	require.Contains(t, got, "domain/model/contracts")
	require.Contains(t, got, "depend inward")
}

func TestArchitectureSuggestionText_CleanArchitectureDeliveryDependsOnConcreteImplementation(t *testing.T) {
	got := architectureSuggestionText("api", "cameras", map[model.DependencyKind]struct{}{
		model.DependencyKindInclude: {},
	})

	require.Contains(t, got, "delivery/API")
	require.Contains(t, got, "application/usecase")
	require.Contains(t, got, "port")
}

func TestArchitectureSuggestionText_CleanArchitectureApplicationDependsOnConcreteImplementation(t *testing.T) {
	got := architectureSuggestionText("application", "vendor", map[model.DependencyKind]struct{}{
		model.DependencyKindImport: {},
	})

	require.Contains(t, got, "application/usecase")
	require.Contains(t, got, "port/interface")
	require.Contains(t, got, "infrastructure/adapters implement it")
}

func TestArchitectureSuggestionText_FallsBackToGenericIncludeSuggestion(t *testing.T) {
	got := architectureSuggestionText("feature_a", "feature_b", map[model.DependencyKind]struct{}{
		model.DependencyKindInclude: {},
	})

	require.Contains(t, got, "forward declaration")
}
