package llmpack

import (
	"bytes"
	"testing"

	"github.com/orurh/patchcourt/internal/analysis/contracts"
	"github.com/orurh/patchcourt/internal/analysis/depdiff"
	"github.com/orurh/patchcourt/internal/analysis/risk"
	"github.com/orurh/patchcourt/internal/model"
	"github.com/orurh/patchcourt/internal/reportmodel"
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
	require.Contains(t, got, "Removed public contract symbol")
	require.Contains(t, got, "architecture.domain.cameras")
	require.Contains(t, got, "## Contract changes")
	require.Contains(t, got, "method::ICamera::Status")
	require.Contains(t, got, "## Dependency changes")
	require.Contains(t, got, "src/api.cc -> src/cameras/sony.h")
	require.NotContains(t, got, "`import|src/api.cc|testing`")
	require.Contains(t, got, "## Review questions")
	require.Contains(t, got, "Verify callers and tests for public contract change")
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
					Key:  "import|internal/output/llmpack/pack.go|sort",
					After: &model.DependencyEdge{
						FromFile: "internal/output/llmpack/pack.go",
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
