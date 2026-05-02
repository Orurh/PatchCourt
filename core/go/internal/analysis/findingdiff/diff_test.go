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

func TestFindingKey_FallsBackToKindSeverityAndTitle(t *testing.T) {
	key := FindingKey(model.Finding{
		Kind:     model.FindingKindDiscoveryHint,
		Severity: model.SeverityLow,
		Title:    "Possibly unused C++ includes",
	})

	require.Equal(t, "discovery_hint|low|Possibly unused C++ includes", key)
}

func finding(id string, severity model.Severity) model.Finding {
	return model.Finding{
		ID:         id,
		Kind:       model.FindingKindPolicyViolation,
		Severity:   severity,
		Title:      "Test finding",
		Confidence: model.ConfidenceHigh,
	}
}
