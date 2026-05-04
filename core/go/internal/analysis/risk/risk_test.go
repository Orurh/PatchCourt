package risk

import (
	"testing"

	"github.com/orurh/patchcourt/internal/analysis/contracts"
	"github.com/orurh/patchcourt/internal/analysis/findingdiff"
	"github.com/orurh/patchcourt/internal/diff/dep"
	"github.com/orurh/patchcourt/internal/model"
	"github.com/stretchr/testify/require"
)

func TestCalculate_ScoresAddedPolicyViolationFinding(t *testing.T) {
	score := Calculate(Input{
		FindingChanges: []findingdiff.FindingChange{
			{
				Kind: findingdiff.FindingChangeKindAdded,
				ID:   "architecture.api.cameras",
				After: &model.Finding{
					ID:       "architecture.api.cameras",
					Kind:     model.FindingKindPolicyViolation,
					Severity: model.SeverityHigh,
				},
			},
		},
	})

	require.Equal(t, 7, score.Points)
	require.Equal(t, LevelMedium, score.Level)
	require.Len(t, score.Reasons, 1)
	require.Equal(t, "added high policy violation: architecture.api.cameras", score.Reasons[0].Message)
}

func TestCalculate_ScoresContractAndDependencyChanges(t *testing.T) {
	score := Calculate(Input{
		ContractChanges: []contracts.SymbolChange{
			{
				Kind:      contracts.ChangeKindSignatureChanged,
				SymbolKey: "method::ICameraAdapter::RunPreflight",
			},
			{
				Kind:      contracts.ChangeKindModifiersChanged,
				SymbolKey: "method::ICameraAdapter::RunPreflight",
			},
		},
		DependencyChanges: []depdiff.DependencyChange{
			{
				Kind: depdiff.DependencyChangeKindAdded,
				Key:  "include|src/server/api_router.cc|src/cameras/sony.h",
				After: &model.DependencyEdge{
					FromLayer: "api",
					ToLayer:   "cameras",
					Resolved:  true,
				},
			},
		},
		LayerEdgeChanges: []depdiff.LayerEdgeChange{
			{
				Kind:      depdiff.DependencyChangeKindAdded,
				FromLayer: "api",
				ToLayer:   "cameras",
			},
		},
	})

	require.Equal(t, 7, score.Points)
	require.Equal(t, LevelMedium, score.Level)
	require.Len(t, score.Reasons, 4)
}

func TestCalculate_CriticalLevel(t *testing.T) {
	score := Calculate(Input{
		FindingChanges: []findingdiff.FindingChange{
			addedFinding("one", model.SeverityCritical),
			addedFinding("two", model.SeverityHigh),
		},
	})

	require.Equal(t, LevelCritical, score.Level)
}

func addedFinding(id string, severity model.Severity) findingdiff.FindingChange {
	return findingdiff.FindingChange{
		Kind: findingdiff.FindingChangeKindAdded,
		ID:   id,
		After: &model.Finding{
			ID:       id,
			Severity: severity,
		},
	}
}

func TestCalculate_ScoresIncreasedLayerEdgeCount(t *testing.T) {
	score := Calculate(Input{
		LayerEdgeChanges: []depdiff.LayerEdgeChange{
			{
				Kind:        depdiff.DependencyChangeKindChanged,
				FromLayer:   "domain",
				ToLayer:     "application",
				BeforeCount: 2,
				AfterCount:  3,
			},
		},
	})

	require.Equal(t, 1, score.Points)
	require.Equal(t, LevelLow, score.Level)
	require.Len(t, score.Reasons, 1)
	require.Equal(t, "layer edge count increased: domain -> application (2 -> 3)", score.Reasons[0].Message)
}

func TestCalculate_DoesNotScoreAddedPublicContractSymbolByDefault(t *testing.T) {
	score := Calculate(Input{
		ContractChanges: []contracts.SymbolChange{
			{
				Kind:      contracts.ChangeKindAdded,
				SymbolKey: "class::ICameraAdapter",
			},
		},
	})

	require.Equal(t, 0, score.Points)
	require.Equal(t, LevelLow, score.Level)
	require.Empty(t, score.Reasons)
}

func TestCalculate_DoesNotScoreNoisyAddedDependencies(t *testing.T) {
	score := Calculate(Input{
		DependencyChanges: []depdiff.DependencyChange{
			{
				Kind: depdiff.DependencyChangeKindAdded,
				Key:  "include|src/domain/a.h|string",
				After: &model.DependencyEdge{
					FromLayer: "domain",
					Resolved:  true,
				},
			},
			{
				Kind: depdiff.DependencyChangeKindAdded,
				Key:  "include|src/domain/a.h|src/domain/b.h",
				After: &model.DependencyEdge{
					FromLayer: "domain",
					ToLayer:   "domain",
					Resolved:  true,
				},
			},
			{
				Kind: depdiff.DependencyChangeKindAdded,
				Key:  "include|src/domain/a.h|vendor.h",
				After: &model.DependencyEdge{
					FromLayer: "domain",
					External:  true,
				},
			},
		},
	})

	require.Equal(t, 0, score.Points)
	require.Equal(t, LevelLow, score.Level)
	require.Empty(t, score.Reasons)
}
