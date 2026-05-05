package sarif

import (
	"bytes"
	"encoding/json"
	"testing"

	contracts "github.com/orurh/patchcourt/internal/diff/contract"
	findingdiff "github.com/orurh/patchcourt/internal/diff/finding"
	"github.com/orurh/patchcourt/internal/model"
	"github.com/orurh/patchcourt/internal/reportmodel"
	"github.com/stretchr/testify/require"
)

func TestWriteReviewSARIF_RendersValidSARIFLog(t *testing.T) {
	var out bytes.Buffer

	result := reportmodel.ReviewResult{
		FindingChanges: []findingdiff.FindingChange{
			{
				Kind: findingdiff.FindingChangeKindAdded,
				ID:   "architecture.api.cameras",
				After: &model.Finding{
					ID:         "architecture.api.cameras",
					Kind:       model.FindingKindPolicyViolation,
					Severity:   model.SeverityHigh,
					Title:      "Include-level architecture boundary violation",
					Risk:       "API layer depends on camera implementation.",
					Suggestion: "Route the call through an application/usecase boundary.",
					Confidence: model.ConfidenceHigh,
					Evidence: []model.Evidence{
						{
							File:      "src/api/camera_routes.cc",
							LineStart: 12,
							Snippet:   `#include "src/cameras/sony/sony_camera_manager.h"`,
							FromLayer: "api",
							ToLayer:   "cameras",
							FromFile:  "src/api/camera_routes.cc",
							ToFile:    "src/cameras/sony/sony_camera_manager.h",
						},
					},
				},
			},
		},
	}

	require.NoError(t, WriteReviewSARIF(&out, result))

	var log Log
	require.NoError(t, json.Unmarshal(out.Bytes(), &log))

	require.Equal(t, "2.1.0", log.Version)
	require.Len(t, log.Runs, 1)
	require.Equal(t, "PatchCourt", log.Runs[0].Tool.Driver.Name)
	require.Len(t, log.Runs[0].Tool.Driver.Rules, 1)
	require.Len(t, log.Runs[0].Results, 1)

	alert := log.Runs[0].Results[0]
	require.Equal(t, "architecture.api.cameras", alert.RuleID)
	require.Equal(t, "error", alert.Level)
	require.Contains(t, alert.Message.Text, "Include-level architecture boundary violation")
	require.Contains(t, alert.Message.Text, "API layer depends on camera implementation")

	require.Len(t, alert.Locations, 1)
	location := alert.Locations[0].PhysicalLocation
	require.Equal(t, "src/api/camera_routes.cc", location.ArtifactLocation.URI)
	require.NotNil(t, location.Region)
	require.Equal(t, 12, location.Region.StartLine)
	require.NotNil(t, location.Region.Snippet)
	require.Contains(t, location.Region.Snippet.Text, "sony_camera_manager.h")

	require.Equal(t, "architecture.api.cameras", alert.Properties["patchcourt.id"])
	require.Equal(t, "policy_violation", alert.Properties["patchcourt.kind"])
	require.Equal(t, "high", alert.Properties["patchcourt.severity"])
	require.Equal(t, "worse", alert.Properties["patchcourt.impact"])
}

func TestBuildReviewSARIF_MapsSeverityLevels(t *testing.T) {
	tests := []struct {
		name     string
		severity model.Severity
		want     string
	}{
		{name: "critical", severity: model.SeverityCritical, want: "error"},
		{name: "high", severity: model.SeverityHigh, want: "error"},
		{name: "medium", severity: model.SeverityMedium, want: "warning"},
		{name: "low", severity: model.SeverityLow, want: "note"},
		{name: "unknown", severity: "", want: "warning"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := reportmodel.ReviewResult{
				FindingChanges: []findingdiff.FindingChange{
					{
						Kind: findingdiff.FindingChangeKindAdded,
						ID:   "finding." + tt.name,
						After: &model.Finding{
							ID:       "finding." + tt.name,
							Severity: tt.severity,
							Title:    "Test finding",
							Evidence: []model.Evidence{
								{File: "src/file.cc"},
							},
						},
					},
				},
			}

			log := BuildReviewSARIF(result)

			require.Len(t, log.Runs[0].Results, 1)
			require.Equal(t, tt.want, log.Runs[0].Results[0].Level)
		})
	}
}

func TestBuildReviewSARIF_IgnoresRemovedFindingChanges(t *testing.T) {
	result := reportmodel.ReviewResult{
		FindingChanges: []findingdiff.FindingChange{
			{
				Kind: findingdiff.FindingChangeKindRemoved,
				ID:   "architecture.old",
				Before: &model.Finding{
					ID:       "architecture.old",
					Severity: model.SeverityHigh,
					Title:    "Old finding",
					Evidence: []model.Evidence{
						{File: "src/old.cc"},
					},
				},
			},
		},
	}

	log := BuildReviewSARIF(result)

	require.Empty(t, log.Runs[0].Results)
	require.Empty(t, log.Runs[0].Tool.Driver.Rules)
}

func TestBuildReviewSARIF_RendersContractAlerts(t *testing.T) {
	result := reportmodel.ReviewResult{
		ContractChanges: []contracts.SymbolChange{
			{
				Kind:      contracts.ChangeKindSignatureChanged,
				SymbolKey: "method::ICameraAdapter::RunPreflight",
				Before: &model.SymbolModel{
					File:      "src/domain/interfaces/i_camera_adapter.h",
					Line:      10,
					Signature: "virtual bool RunPreflight() const = 0;",
				},
				After: &model.SymbolModel{
					File:      "src/domain/interfaces/i_camera_adapter.h",
					Line:      10,
					Signature: "virtual bool RunPreflight(int cameraIndex) const = 0;",
				},
			},
		},
	}

	log := BuildReviewSARIF(result)

	require.Len(t, log.Runs[0].Results, 1)
	alert := log.Runs[0].Results[0]

	require.Equal(t, "patchcourt.contract.signature-changed", alert.RuleID)
	require.Equal(t, "error", alert.Level)
	require.Contains(t, alert.Message.Text, "Public contract signature changed")
	require.Equal(t, "method::ICameraAdapter::RunPreflight", alert.Properties["patchcourt.symbol_key"])
	require.Len(t, alert.Locations, 1)
	require.Equal(t, "src/domain/interfaces/i_camera_adapter.h", alert.Locations[0].PhysicalLocation.ArtifactLocation.URI)
}

func TestBuildReviewSARIF_RendersWorseImpactWithEvidence(t *testing.T) {
	result := reportmodel.ReviewResult{
		Impact: reportmodel.ReviewImpactReport{
			Worse: []reportmodel.ReviewImpactItem{
				{
					Kind:       "forbidden_dependency",
					Severity:   string(model.SeverityHigh),
					Title:      "New forbidden dependency",
					Detail:     "api -> cameras",
					Risk:       "Delivery layer now knows camera implementation details.",
					Suggestion: "Depend on an application port instead.",
					ID:         "architecture.api.cameras",
					Evidence: []model.Evidence{
						{
							File:      "src/api/camera_routes.cc",
							LineStart: 7,
							FromLayer: "api",
							ToLayer:   "cameras",
						},
					},
				},
			},
		},
	}

	log := BuildReviewSARIF(result)

	require.Len(t, log.Runs[0].Results, 1)
	alert := log.Runs[0].Results[0]

	require.Equal(t, "patchcourt.impact.forbidden-dependency", alert.RuleID)
	require.Equal(t, "error", alert.Level)
	require.Contains(t, alert.Message.Text, "New forbidden dependency")
	require.Equal(t, "forbidden_dependency", alert.Properties["patchcourt.kind"])
	require.Equal(t, "worse", alert.Properties["patchcourt.impact"])
}

func TestBuildReviewSARIF_DoesNotRenderBetterOrUnchangedDebt(t *testing.T) {
	result := reportmodel.ReviewResult{
		Impact: reportmodel.ReviewImpactReport{
			Better: []reportmodel.ReviewImpactItem{
				{
					Kind:  "finding_removed",
					Title: "Removed old finding",
					Evidence: []model.Evidence{
						{File: "src/better.cc"},
					},
				},
			},
			UnchangedDebt: []reportmodel.ReviewImpactItem{
				{
					Kind:  "existing_cycle",
					Title: "Existing cycle",
					Evidence: []model.Evidence{
						{File: "src/debt.cc"},
					},
				},
			},
		},
	}

	log := BuildReviewSARIF(result)

	require.Empty(t, log.Runs[0].Results)
	require.Empty(t, log.Runs[0].Tool.Driver.Rules)
}
