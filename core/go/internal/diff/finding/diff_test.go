package findingdiff

import (
	"testing"

	"github.com/orurh/patchcourt/internal/model"
	"github.com/stretchr/testify/require"
)

func TestDiffFindings_DetectsAddedFinding(t *testing.T) {
	after := []model.Finding{
		finding("architecture.api.cameras", model.SeverityHigh),
	}

	changes := DiffFindings(nil, after)

	require.Len(t, changes, 1)
	require.Equal(t, FindingChangeKindAdded, changes[0].Kind)
	require.Equal(t, "architecture.api.cameras", changes[0].ID)
	require.NotNil(t, changes[0].After)
	require.Nil(t, changes[0].Before)
	require.Equal(t, 1, changes[0].AfterEvidenceCount)
	require.Len(t, changes[0].AddedEvidence, 1)
}

func TestDiffFindings_DetectsRemovedFinding(t *testing.T) {
	before := []model.Finding{
		finding("architecture.api.cameras", model.SeverityHigh),
	}

	changes := DiffFindings(before, nil)

	require.Len(t, changes, 1)
	require.Equal(t, FindingChangeKindRemoved, changes[0].Kind)
	require.Equal(t, "architecture.api.cameras", changes[0].ID)
	require.NotNil(t, changes[0].Before)
	require.Nil(t, changes[0].After)
	require.Equal(t, 1, changes[0].BeforeEvidenceCount)
	require.Len(t, changes[0].RemovedEvidence, 1)
}

func TestDiffFindings_IgnoresUnchangedFinding(t *testing.T) {
	before := []model.Finding{
		finding("architecture.api.cameras", model.SeverityHigh),
	}
	after := []model.Finding{
		finding("architecture.api.cameras", model.SeverityHigh),
	}

	changes := DiffFindings(before, after)

	require.Empty(t, changes)
}

func TestDiffFindings_DetectsChangedFindingSeverity(t *testing.T) {
	before := []model.Finding{
		finding("cpp.async.this_capture", model.SeverityMedium),
	}
	after := []model.Finding{
		finding("cpp.async.this_capture", model.SeverityHigh),
	}

	changes := DiffFindings(before, after)

	require.Len(t, changes, 1)
	require.Equal(t, FindingChangeKindChanged, changes[0].Kind)
	require.Equal(t, "cpp.async.this_capture", changes[0].ID)
	require.True(t, changes[0].SeverityChanged)
	require.False(t, changes[0].ConfidenceChanged)
	require.NotNil(t, changes[0].Before)
	require.NotNil(t, changes[0].After)
	require.Equal(t, model.SeverityMedium, changes[0].Before.Severity)
	require.Equal(t, model.SeverityHigh, changes[0].After.Severity)
}

func TestDiffFindings_DetectsChangedFindingConfidence(t *testing.T) {
	beforeFinding := finding("cpp.async.this_capture", model.SeverityHigh)
	beforeFinding.Confidence = model.ConfidenceMedium

	afterFinding := finding("cpp.async.this_capture", model.SeverityHigh)
	afterFinding.Confidence = model.ConfidenceHigh

	changes := DiffFindings([]model.Finding{beforeFinding}, []model.Finding{afterFinding})

	require.Len(t, changes, 1)
	require.Equal(t, FindingChangeKindChanged, changes[0].Kind)
	require.True(t, changes[0].ConfidenceChanged)
	require.False(t, changes[0].SeverityChanged)
}

func TestDiffFindings_DetectsAddedEvidenceForExistingFinding(t *testing.T) {
	before := finding("cpp.async.raw_pointer_capture", model.SeverityHigh)
	before.Evidence = []model.Evidence{
		evidence("src/gopro_manager.cc", 10, "old capture"),
	}

	after := finding("cpp.async.raw_pointer_capture", model.SeverityHigh)
	after.Evidence = []model.Evidence{
		evidence("src/gopro_manager.cc", 10, "old capture"),
		evidence("src/gopro_manager.cc", 42, "new capture"),
	}

	changes := DiffFindings([]model.Finding{before}, []model.Finding{after})

	require.Len(t, changes, 1)
	require.Equal(t, FindingChangeKindChanged, changes[0].Kind)
	require.Equal(t, 1, changes[0].BeforeEvidenceCount)
	require.Equal(t, 2, changes[0].AfterEvidenceCount)
	require.Len(t, changes[0].AddedEvidence, 1)
	require.Empty(t, changes[0].RemovedEvidence)
	require.Equal(t, 42, changes[0].AddedEvidence[0].LineStart)
}

func TestDiffFindings_DetectsRemovedEvidenceForExistingFinding(t *testing.T) {
	before := finding("cpp.async.raw_pointer_capture", model.SeverityHigh)
	before.Evidence = []model.Evidence{
		evidence("src/gopro_manager.cc", 10, "old capture"),
		evidence("src/gopro_manager.cc", 42, "removed capture"),
	}

	after := finding("cpp.async.raw_pointer_capture", model.SeverityHigh)
	after.Evidence = []model.Evidence{
		evidence("src/gopro_manager.cc", 10, "old capture"),
	}

	changes := DiffFindings([]model.Finding{before}, []model.Finding{after})

	require.Len(t, changes, 1)
	require.Equal(t, FindingChangeKindChanged, changes[0].Kind)
	require.Equal(t, 2, changes[0].BeforeEvidenceCount)
	require.Equal(t, 1, changes[0].AfterEvidenceCount)
	require.Empty(t, changes[0].AddedEvidence)
	require.Len(t, changes[0].RemovedEvidence, 1)
	require.Equal(t, 42, changes[0].RemovedEvidence[0].LineStart)
}

func TestFindingKey_FallsBackToKindSeverityAndTitle(t *testing.T) {
	key := FindingKey(model.Finding{
		Kind:     model.FindingKindDiscoveryHint,
		Severity: model.SeverityLow,
		Title:    "Possibly unused C++ includes",
	})

	require.Equal(t, "discovery_hint|low|Possibly unused C++ includes", key)
}

func TestEvidenceKey_UsesStructuredLocationAndMessage(t *testing.T) {
	key := EvidenceKey(model.Evidence{
		File:      "src/gopro_manager.cc",
		LineStart: 10,
		LineEnd:   12,
		Message:   "captures this",
		FromLayer: "manager",
		ToLayer:   "camera",
		FromFile:  "src/gopro_manager.cc",
		ToFile:    "src/gopro_camera.h",
	})

	require.Contains(t, key, "src/gopro_manager.cc")
	require.Contains(t, key, "10")
	require.Contains(t, key, "captures this")
	require.Contains(t, key, "manager|camera")
}

func finding(id string, severity model.Severity) model.Finding {
	return model.Finding{
		ID:         id,
		Kind:       model.FindingKindPolicyViolation,
		Severity:   severity,
		Title:      "Test finding",
		Confidence: model.ConfidenceHigh,
		Evidence: []model.Evidence{
			evidence("src/server/api_router.cc", 7, "includes src/cameras/sony.h"),
		},
	}
}

func evidence(file string, line int, message string) model.Evidence {
	return model.Evidence{
		File:      file,
		LineStart: line,
		Message:   message,
	}
}
