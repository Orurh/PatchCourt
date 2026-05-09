package risk

import (
	"testing"

	"github.com/orurh/patchcourt/internal/diff/contract"
	"github.com/orurh/patchcourt/internal/diff/dep"
	"github.com/orurh/patchcourt/internal/diff/finding"
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

func TestCalculate_ScoresChangedFindingSeverityIncrease(t *testing.T) {
	score := Calculate(Input{
		FindingChanges: []findingdiff.FindingChange{
			{
				Kind: findingdiff.FindingChangeKindChanged,
				ID:   "cpp.lifetime.this_capture_async",
				Before: &model.Finding{
					ID:       "cpp.lifetime.this_capture_async",
					Severity: model.SeverityMedium,
				},
				After: &model.Finding{
					ID:       "cpp.lifetime.this_capture_async",
					Severity: model.SeverityHigh,
				},
				SeverityChanged: true,
			},
		},
	})

	require.Equal(t, 2, score.Points)
	require.Equal(t, LevelLow, score.Level)
	require.Len(t, score.Reasons, 1)
	require.Equal(t, "finding severity increased: cpp.lifetime.this_capture_async (medium -> high)", score.Reasons[0].Message)
}

func TestCalculate_ScoresChangedFindingAddedEvidence(t *testing.T) {
	score := Calculate(Input{
		FindingChanges: []findingdiff.FindingChange{
			{
				Kind: findingdiff.FindingChangeKindChanged,
				ID:   "cpp.lifetime.raw_pointer_async_capture",
				Before: &model.Finding{
					ID:       "cpp.lifetime.raw_pointer_async_capture",
					Severity: model.SeverityHigh,
				},
				After: &model.Finding{
					ID:       "cpp.lifetime.raw_pointer_async_capture",
					Severity: model.SeverityHigh,
				},
				AddedEvidence: []model.Evidence{
					{File: "src/gopro_manager.cc", LineStart: 10},
					{File: "src/gopro_manager.cc", LineStart: 20},
				},
			},
		},
	})

	require.Equal(t, 2, score.Points)
	require.Equal(t, LevelLow, score.Level)
	require.Len(t, score.Reasons, 1)
	require.Equal(t, "finding evidence increased: cpp.lifetime.raw_pointer_async_capture (+2 net evidence)", score.Reasons[0].Message)
}

func TestCalculate_CapsChangedFindingAddedEvidenceScore(t *testing.T) {
	score := Calculate(Input{
		FindingChanges: []findingdiff.FindingChange{
			{
				Kind: findingdiff.FindingChangeKindChanged,
				ID:   "cpp.lifetime.raw_pointer_async_capture",
				Before: &model.Finding{
					ID:       "cpp.lifetime.raw_pointer_async_capture",
					Severity: model.SeverityHigh,
				},
				After: &model.Finding{
					ID:       "cpp.lifetime.raw_pointer_async_capture",
					Severity: model.SeverityHigh,
				},
				AddedEvidence: []model.Evidence{
					{File: "a.cc", LineStart: 1},
					{File: "a.cc", LineStart: 2},
					{File: "a.cc", LineStart: 3},
					{File: "a.cc", LineStart: 4},
				},
			},
		},
	})

	require.Equal(t, 3, score.Points)
	require.Equal(t, LevelMedium, score.Level)
	require.Len(t, score.Reasons, 1)
	require.Equal(t, "finding evidence increased: cpp.lifetime.raw_pointer_async_capture (+4 net evidence)", score.Reasons[0].Message)
}

func TestCalculate_ScoresChangedFindingConfidenceIncreaseWithoutOtherRiskSignal(t *testing.T) {
	score := Calculate(Input{
		FindingChanges: []findingdiff.FindingChange{
			{
				Kind: findingdiff.FindingChangeKindChanged,
				ID:   "cpp.lifetime.this_capture_async",
				Before: &model.Finding{
					ID:         "cpp.lifetime.this_capture_async",
					Severity:   model.SeverityMedium,
					Confidence: model.ConfidenceMedium,
				},
				After: &model.Finding{
					ID:         "cpp.lifetime.this_capture_async",
					Severity:   model.SeverityMedium,
					Confidence: model.ConfidenceHigh,
				},
				ConfidenceChanged: true,
			},
		},
	})

	require.Equal(t, 1, score.Points)
	require.Equal(t, LevelLow, score.Level)
	require.Len(t, score.Reasons, 1)
	require.Equal(t, "finding confidence increased: cpp.lifetime.this_capture_async (medium -> high)", score.Reasons[0].Message)
}

func TestCalculate_DoesNotScoreChangedFindingEvidenceReduction(t *testing.T) {
	score := Calculate(Input{
		FindingChanges: []findingdiff.FindingChange{
			{
				Kind: findingdiff.FindingChangeKindChanged,
				ID:   "cpp.lifetime.raw_pointer_async_capture",
				Before: &model.Finding{
					ID:       "cpp.lifetime.raw_pointer_async_capture",
					Severity: model.SeverityHigh,
				},
				After: &model.Finding{
					ID:       "cpp.lifetime.raw_pointer_async_capture",
					Severity: model.SeverityHigh,
				},
				RemovedEvidence: []model.Evidence{
					{File: "src/gopro_manager.cc", LineStart: 10},
				},
			},
		},
	})

	require.Equal(t, 0, score.Points)
	require.Equal(t, LevelLow, score.Level)
	require.Empty(t, score.Reasons)
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

func TestCalculate_DoesNotScoreChangedFindingWhenEvidenceNetDecreases(t *testing.T) {
	score := Calculate(Input{
		FindingChanges: []findingdiff.FindingChange{
			{
				Kind: findingdiff.FindingChangeKindChanged,
				ID:   "discovery.shared_candidate.application.constants",
				Before: &model.Finding{
					ID:         "discovery.shared_candidate.application.constants",
					Kind:       model.FindingKindDiscoveryHint,
					Severity:   model.SeverityLow,
					Confidence: model.ConfidenceMedium,
					Evidence: []model.Evidence{
						{File: "a.cc"},
						{File: "b.cc"},
						{File: "c.cc"},
						{File: "d.cc"},
						{File: "e.cc"},
					},
				},
				After: &model.Finding{
					ID:         "discovery.shared_candidate.application.constants",
					Kind:       model.FindingKindDiscoveryHint,
					Severity:   model.SeverityLow,
					Confidence: model.ConfidenceMedium,
					Evidence: []model.Evidence{
						{File: "x.cc"},
					},
				},
				AddedEvidence: []model.Evidence{
					{File: "x.cc"},
				},
				RemovedEvidence: []model.Evidence{
					{File: "a.cc"},
					{File: "b.cc"},
					{File: "c.cc"},
					{File: "d.cc"},
					{File: "e.cc"},
				},
			},
		},
	})

	require.Equal(t, 0, score.Points)
	require.Equal(t, LevelLow, score.Level)
	require.Empty(t, score.Reasons)
}

func TestCalculate_ScoresDiscoveryHintEvidenceGrowthAsDiscoverySignal(t *testing.T) {
	score := Calculate(Input{
		FindingChanges: []findingdiff.FindingChange{
			{
				Kind: findingdiff.FindingChangeKindChanged,
				ID:   "discovery.shared_candidate.application.constants",
				Before: &model.Finding{
					ID:         "discovery.shared_candidate.application.constants",
					Kind:       model.FindingKindDiscoveryHint,
					Severity:   model.SeverityLow,
					Confidence: model.ConfidenceMedium,
				},
				After: &model.Finding{
					ID:         "discovery.shared_candidate.application.constants",
					Kind:       model.FindingKindDiscoveryHint,
					Severity:   model.SeverityLow,
					Confidence: model.ConfidenceMedium,
				},
				AddedEvidence: []model.Evidence{
					{File: "src/cameras/a.cc"},
					{File: "src/cameras/b.cc"},
					{File: "src/cameras/c.cc"},
					{File: "src/cameras/d.cc"},
					{File: "src/cameras/e.cc"},
				},
				RemovedEvidence: []model.Evidence{
					{File: "src/utils/a.cc"},
					{File: "src/utils/b.cc"},
					{File: "src/utils/c.cc"},
				},
			},
		},
	})

	require.Equal(t, 2, score.Points)
	require.Equal(t, LevelLow, score.Level)
	require.Len(t, score.Reasons, 1)
	require.Equal(t, "discovery signal gained evidence: discovery.shared_candidate.application.constants (+2 net evidence)", score.Reasons[0].Message)
	require.Equal(t, 2, score.Reasons[0].Points)
}
