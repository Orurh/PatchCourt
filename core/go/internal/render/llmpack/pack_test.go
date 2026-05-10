package llmpack

import (
	"bytes"
	"testing"

	"github.com/orurh/patchcourt/internal/diff/contract"
	"github.com/orurh/patchcourt/internal/diff/dep"
	findingdiff "github.com/orurh/patchcourt/internal/diff/finding"
	"github.com/orurh/patchcourt/internal/model"
	"github.com/orurh/patchcourt/internal/reportmodel"
	"github.com/orurh/patchcourt/internal/reviewrisk"
	"github.com/stretchr/testify/require"
)

func TestWriteReviewContext_RendersDeterministicContextPack(t *testing.T) {
	var out bytes.Buffer

	WriteReviewContext(&out, ReviewContextInput{
		MaxItems: 3,
		Result: reportmodel.ReviewResult{
			SchemaVersion: reportmodel.ReviewResultSchemaVersion,
			Summary: reportmodel.ReviewSummary{
				ContractChanges:   1,
				DependencyChanges: 1,
				LayerEdgeChanges:  1,
				FindingChanges:    1,
				AddedFindings:     1,
			},
			Risk: risk.Score{
				Level:  risk.LevelMedium,
				Points: 5,
				Reasons: []risk.Reason{
					{Points: 3, Message: "public contract symbol removed: method::ICamera::Status"},
				},
			},
			Impact: reportmodel.ReviewImpactReport{
				Worse: []reportmodel.ReviewImpactItem{
					{
						Kind:   "contract_removed",
						Title:  "Removed public contract symbol",
						Detail: "method::ICamera::Status",
					},
				},
				Better: []reportmodel.ReviewImpactItem{
					{
						Kind:  "finding_removed",
						Title: "Removed policy violation finding",
						ID:    "architecture.domain.cameras",
					},
				},
				NeedsReview: []reportmodel.ReviewImpactItem{
					{
						Kind:       "contract_delivery_impact",
						Title:      "Public contract changed with delivery/API impact",
						Detail:     "method::ICamera::Status",
						Suggestion: "Verify migration and tests.",
					},
				},
			},
			ContractChanges: []contracts.SymbolChange{
				{
					Kind:      contracts.ChangeKindRemoved,
					SymbolKey: "method::ICamera::Status",
					Before: &model.SymbolModel{
						Signature: "virtual Status GetStatus() const = 0;",
					},
				},
			},
			DependencyChanges: []depdiff.DependencyChange{
				{
					Kind: depdiff.DependencyChangeKindAdded,
					Key:  "include|src/api.cc|src/cameras/sony.h",
					After: &model.DependencyEdge{
						FromFile:  "src/api.cc",
						ToFile:    "src/cameras/sony.h",
						FromLayer: "api",
						ToLayer:   "cameras",
					},
				},
				{
					Kind: depdiff.DependencyChangeKindAdded,
					Key:  "import|src/api.cc|testing",
					After: &model.DependencyEdge{
						FromFile: "src/api.cc",
						Target:   "testing",
						External: true,
					},
				},
			},
			LayerEdgeChanges: []depdiff.LayerEdgeChange{
				{
					Kind:       depdiff.DependencyChangeKindAdded,
					FromLayer:  "api",
					ToLayer:    "cameras",
					AfterCount: 1,
				},
			},
		},
	})

	got := out.String()

	require.Contains(t, got, "# PatchCourt Review Context")
	require.Contains(t, got, "Do not invent files, dependencies, symbols, or findings not listed here.")
	require.Contains(t, got, "- Schema: `patchcourt.review.v1`")
	require.Contains(t, got, "- Risk: `medium`, 5 points")
	require.Contains(t, got, "## Changed files")
	require.Contains(t, got, "- `src/api.cc`")
	require.Contains(t, got, "- `src/cameras/sony.h`")
	require.Contains(t, got, "## Touched layers")
	require.Contains(t, got, "- `api`")
	require.Contains(t, got, "- `cameras`")
	require.Contains(t, got, "## Architecture impact")
	require.Contains(t, got, "### Real problems introduced")
	require.Contains(t, got, "### Verified improvements")
	require.Contains(t, got, "### Needs review / AI follow-up")
	require.Contains(t, got, "Removed public contract symbol")
	require.Contains(t, got, "architecture.domain.cameras")
	require.Contains(t, got, "## AI follow-up prompt")
	require.Contains(t, got, "Do not invent files, dependencies, symbols, call sites, tests, or findings not listed in this context pack.")
	require.Contains(t, got, "contract_delivery_impact: Public contract changed with delivery/API impact")
	require.Contains(t, got, "Verify migration and tests.")
	require.Contains(t, got, "## Contract changes")
	require.Contains(t, got, "method::ICamera::Status")
	require.Contains(t, got, "## Dependency changes")
	require.Contains(t, got, "src/api.cc -> src/cameras/sony.h")
	require.NotContains(t, got, "`import|src/api.cc|testing`")
	require.Contains(t, got, "## Review questions")
	require.Contains(t, got, "Public contract changed `method::ICamera::Status`, but no test-like files changed")
	require.Contains(t, got, "Verify callers and add or update tests")
}

func TestWriteReviewContext_ReportsRawDependencyChangesWhenNoneReviewRelevant(t *testing.T) {
	var out bytes.Buffer

	WriteReviewContext(&out, ReviewContextInput{
		MaxItems: 3,
		Result: reportmodel.ReviewResult{
			SchemaVersion: reportmodel.ReviewResultSchemaVersion,
			Summary: reportmodel.ReviewSummary{
				DependencyChanges: 1,
			},
			DependencyChanges: []depdiff.DependencyChange{
				{
					Kind: depdiff.DependencyChangeKindAdded,
					Key:  "import|internal/render/llmpack/pack.go|sort",
					After: &model.DependencyEdge{
						FromFile: "internal/render/llmpack/pack.go",
						Target:   "sort",
						External: true,
					},
				},
			},
		},
	})

	got := out.String()

	require.Contains(t, got, "## Dependency changes")
	require.Contains(t, got, "- none review-relevant; raw dependency changes: 1")
}

func TestWriteReviewContext_UsesReviewResultChangedFiles(t *testing.T) {
	var out bytes.Buffer

	WriteReviewContext(&out, ReviewContextInput{
		MaxItems: 10,
		Result: reportmodel.ReviewResult{
			SchemaVersion: reportmodel.ReviewResultSchemaVersion,
			ChangedFiles: []string{
				"internal/render/llmpack/pack.go",
				"internal/usecase/review.go",
			},
		},
	})

	got := out.String()

	require.Contains(t, got, "## Changed files")
	require.Contains(t, got, "- `internal/usecase/review.go`")
	require.Contains(t, got, "- `internal/render/llmpack/pack.go`")
}

func TestWriteReviewContext_SeparatesRawAndAnalyzedChangedFiles(t *testing.T) {
	var out bytes.Buffer

	WriteReviewContext(&out, ReviewContextInput{
		MaxItems: 10,
		Result: reportmodel.ReviewResult{
			SchemaVersion: reportmodel.ReviewResultSchemaVersion,
			ChangedFiles: []string{
				"frontend/src/app/App.tsx",
				"src/api.cc",
			},
			DependencyChanges: []depdiff.DependencyChange{
				{
					Kind: depdiff.DependencyChangeKindAdded,
					Key:  "include|src/api.cc|src/cameras/sony.h",
					After: &model.DependencyEdge{
						FromFile:  "src/api.cc",
						ToFile:    "src/cameras/sony.h",
						FromLayer: "api",
						ToLayer:   "cameras",
					},
				},
			},
		},
	})

	got := out.String()

	require.Contains(t, got, "## Changed files")
	require.Contains(t, got, "- `frontend/src/app/App.tsx`")
	require.Contains(t, got, "## Analyzed changed files")
	require.Contains(t, got, "- `src/api.cc`")
	require.Contains(t, got, "- `src/cameras/sony.h`")
}

func TestWriteReviewContext_SummaryIncludesRawAndAnalyzedChangedFileCounts(t *testing.T) {
	var out bytes.Buffer

	WriteReviewContext(&out, ReviewContextInput{
		MaxItems: 10,
		Result: reportmodel.ReviewResult{
			SchemaVersion: reportmodel.ReviewResultSchemaVersion,
			ChangedFiles: []string{
				"frontend/src/app/App.tsx",
				"src/api.cc",
			},
			DependencyChanges: []depdiff.DependencyChange{
				{
					Kind: depdiff.DependencyChangeKindAdded,
					Key:  "include|src/api.cc|src/cameras/sony.h",
					After: &model.DependencyEdge{
						FromFile:  "src/api.cc",
						ToFile:    "src/cameras/sony.h",
						FromLayer: "api",
						ToLayer:   "cameras",
					},
				},
			},
		},
	})

	got := out.String()

	require.Contains(t, got, "- Changed files: 2")
	require.Contains(t, got, "- Analyzed changed files: 2")
}

func TestWriteReviewContext_AsksForTestsWhenPublicContractChangedWithoutRelatedTests(t *testing.T) {
	var out bytes.Buffer

	WriteReviewContext(&out, ReviewContextInput{
		MaxItems: 10,
		Result: reportmodel.ReviewResult{
			SchemaVersion: reportmodel.ReviewResultSchemaVersion,
			ChangedFiles: []string{
				"src/domain/interfaces/i_camera_manager_controller.h",
				"frontend/src/app/App.tsx",
			},
			ContractChanges: []contracts.SymbolChange{
				{
					Kind:      contracts.ChangeKindSignatureChanged,
					SymbolKey: "method::ICameraManagerController::GetCameraStatus",
					Before: &model.SymbolModel{
						File:   "src/domain/interfaces/i_camera_manager_controller.h",
						Name:   "GetCameraStatus",
						Parent: "ICameraManagerController",
					},
					After: &model.SymbolModel{
						File:   "src/domain/interfaces/i_camera_manager_controller.h",
						Name:   "GetCameraStatus",
						Parent: "ICameraManagerController",
					},
				},
			},
		},
	})

	got := out.String()

	require.Contains(t, got, "Public contract changed `method::ICameraManagerController::GetCameraStatus`, but no test-like files changed")
	require.Contains(t, got, "Verify callers and add or update tests")
}

func TestWriteReviewContext_RecognizesRelatedChangedTestsForPublicContractChange(t *testing.T) {
	var out bytes.Buffer

	WriteReviewContext(&out, ReviewContextInput{
		MaxItems: 10,
		Result: reportmodel.ReviewResult{
			SchemaVersion: reportmodel.ReviewResultSchemaVersion,
			ChangedFiles: []string{
				"test/unit/mocks/mock_camera_manager_controller.h",
				"test/unit/camera_manager_controller_test.cc",
			},
			ContractChanges: []contracts.SymbolChange{
				{
					Kind:      contracts.ChangeKindRemoved,
					SymbolKey: "method::ICameraManagerController::GetCameraStatus",
					Before: &model.SymbolModel{
						File:   "src/domain/interfaces/i_camera_manager_controller.h",
						Name:   "GetCameraStatus",
						Parent: "ICameraManagerController",
					},
				},
			},
		},
	})

	got := out.String()

	require.Contains(t, got, "test-like files changed in this patch")
	require.NotContains(t, got, "but no test-like files changed")
}

func TestWriteReviewContext_IncludesFindingRiskAndSuggestion(t *testing.T) {
	var out bytes.Buffer

	WriteReviewContext(&out, ReviewContextInput{
		MaxItems: 10,
		Result: reportmodel.ReviewResult{
			FindingChanges: []findingdiff.FindingChange{
				{
					Kind: findingdiff.FindingChangeKindAdded,
					ID:   "architecture.api.cameras",
					After: &model.Finding{
						ID:         "architecture.api.cameras",
						Severity:   model.SeverityHigh,
						Kind:       model.FindingKindPolicyViolation,
						Title:      "Include-level architecture boundary violation",
						Risk:       `Layer "api" includes a header from layer "cameras", which is not allowed by .patchcourt.yaml.`,
						Suggestion: "Keep delivery/API code from depending directly on concrete infrastructure or vendor implementations. Route the call through an application/usecase boundary and depend on a domain/application port instead.",
						Evidence: []model.Evidence{
							{
								File:      "src/api/camera_routes.cc",
								Message:   "includes src/infrastructure/cameras/sony/sony_camera_manager.h",
								FromLayer: "api",
								ToLayer:   "cameras",
							},
						},
					},
				},
			},
		},
	})

	got := out.String()

	require.Contains(t, got, "## Finding changes")
	require.Contains(t, got, "architecture.api.cameras")
	require.Contains(t, got, "after risk:")
	require.Contains(t, got, `Layer "api" includes a header`)
	require.Contains(t, got, "after suggestion:")
	require.Contains(t, got, "Keep delivery/API code")
	require.Contains(t, got, "application/usecase")
	require.Contains(t, got, "domain/application port")
}
