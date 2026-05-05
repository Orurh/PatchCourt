package review

import (
	"bytes"
	"testing"

	contractdiff "github.com/orurh/patchcourt/internal/diff/contract"
	depdiff "github.com/orurh/patchcourt/internal/diff/dep"
	findingdiff "github.com/orurh/patchcourt/internal/diff/finding"
	"github.com/orurh/patchcourt/internal/model"
	"github.com/orurh/patchcourt/internal/reportmodel"
	"github.com/stretchr/testify/require"
)

func TestWriteReviewHTML_RendersRiskImpactAndChangedFiles(t *testing.T) {
	var out bytes.Buffer

	err := WriteReviewHTML(&out, reportmodel.ReviewResult{
		Summary: reportmodel.ReviewSummary{
			ContractChanges:   1,
			DependencyChanges: 2,
			LayerEdgeChanges:  3,
			FindingChanges:    4,
			AddedFindings:     1,
			RemovedFindings:   1,
		},
		ChangedFiles: []string{
			"src/api/router.cc",
			"src/cameras/sony.h",
		},
		ContractChanges: []contractdiff.SymbolChange{
			{
				Kind:      contractdiff.ChangeKindSignatureChanged,
				SymbolKey: "method::ICameraAdapter::RunPreflight",
				Before: &model.SymbolModel{
					File:      "src/domain/interfaces/i_camera_adapter.h",
					Line:      12,
					Signature: "RunPreflight() const",
				},
				After: &model.SymbolModel{
					File:      "src/domain/interfaces/i_camera_adapter.h",
					Line:      14,
					Signature: "RunPreflight(int camera_index) const",
				},
			},
		},
		DependencyChanges: []depdiff.DependencyChange{
			{
				Kind: depdiff.DependencyChangeKindAdded,
				Key:  "import|src/api/router.cc|src/cameras/sony.h",
				After: &model.DependencyEdge{
					FromFile:  "src/api/router.cc",
					ToFile:    "src/cameras/sony.h",
					FromLayer: "api",
					ToLayer:   "cameras",
					Usage:     model.DependencyUsageUnknown,
				},
			},
		},
		FindingChanges: []findingdiff.FindingChange{
			{
				Kind: findingdiff.FindingChangeKindAdded,
				ID:   "architecture.api.cameras",
				After: &model.Finding{
					ID:       "architecture.api.cameras",
					Severity: model.SeverityHigh,
					Title:    "Architecture boundary violation",
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
		Impact: reportmodel.ReviewImpactReport{
			Worse: []reportmodel.ReviewImpactItem{
				{
					Kind:   "dependency_added",
					Title:  "Added forbidden dependency",
					Detail: "api -> cameras",
					ID:     "architecture.api.cameras",
				},
			},
			Better: []reportmodel.ReviewImpactItem{
				{
					Kind:  "finding_removed",
					Title: "Removed policy violation",
					ID:    "architecture.cli.platform",
				},
			},
		},
	})
	require.NoError(t, err)

	got := out.String()

	require.Contains(t, got, "<!doctype html>")
	require.Contains(t, got, "PatchCourt")
	require.Contains(t, got, "Review report")
	require.Contains(t, got, "Architecture impact")
	require.Contains(t, got, "Layer impact graph")
	require.Contains(t, got, "graph LR")
	require.Contains(t, got, `api`)
	require.Contains(t, got, `cameras`)
	require.Contains(t, got, "Worse")
	require.Contains(t, got, "Better")
	require.Contains(t, got, "Unchanged debt")
	require.Contains(t, got, "Added forbidden dependency")
	require.Contains(t, got, "architecture.api.cameras")
	require.Contains(t, got, "src/api/router.cc")
	require.Contains(t, got, "Contract changes")
	require.Contains(t, got, "method::ICameraAdapter::RunPreflight")
	require.Contains(t, got, "breaking")
	require.Contains(t, got, "src/domain/interfaces/i_camera_adapter.h:12 → 14")
	require.Contains(t, got, "RunPreflight() const")
	require.Contains(t, got, "RunPreflight(int camera_index) const")
	require.Contains(t, got, "Dependency changes")
	require.Contains(t, got, "Layer edge changes")
	require.Contains(t, got, "Finding changes")
	require.Contains(t, got, "Architecture boundary violation")
	require.Contains(t, got, "Review questions")
	require.Contains(t, got, "Public contract changed `method::ICameraAdapter::RunPreflight`, but no test-like files changed")
}

func TestWriteReviewHTML_EscapesHTML(t *testing.T) {
	var out bytes.Buffer

	err := WriteReviewHTML(&out, reportmodel.ReviewResult{
		ChangedFiles: []string{`src/<script>.cc`},
		ContractChanges: []contractdiff.SymbolChange{
			{
				Kind:      contractdiff.ChangeKindSignatureChanged,
				SymbolKey: "method::<script>",
				After: &model.SymbolModel{
					File:      "src/<contract>.h",
					Signature: "Run(<bad>)",
				},
			},
		},
		DependencyChanges: []depdiff.DependencyChange{
			{
				Kind: depdiff.DependencyChangeKindAdded,
				Key:  "import|src/api/router.cc|src/cameras/sony.h",
				After: &model.DependencyEdge{
					FromFile:  "src/api/router.cc",
					ToFile:    "src/cameras/sony.h",
					FromLayer: "api",
					ToLayer:   "cameras",
					Usage:     model.DependencyUsageUnknown,
				},
			},
		},
		FindingChanges: []findingdiff.FindingChange{
			{
				Kind: findingdiff.FindingChangeKindAdded,
				ID:   "architecture.api.cameras",
				After: &model.Finding{
					ID:       "architecture.api.cameras",
					Severity: model.SeverityHigh,
					Title:    "Architecture boundary violation",
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
		Impact: reportmodel.ReviewImpactReport{
			Worse: []reportmodel.ReviewImpactItem{
				{
					Title:  `<script>alert(1)</script>`,
					Detail: `api -> <cameras>`,
				},
			},
		},
	})
	require.NoError(t, err)

	got := out.String()

	require.Contains(t, got, "src/&lt;script&gt;.cc")
	require.Contains(t, got, "&lt;script&gt;alert(1)&lt;/script&gt;")
	require.Contains(t, got, "api -&gt; &lt;cameras&gt;")
	require.Contains(t, got, "method::&lt;script&gt;")
	require.Contains(t, got, "Run(&lt;bad&gt;)")
	require.NotContains(t, got, "<script>alert(1)</script>")
}
