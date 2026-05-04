package report

import (
	"bytes"
	"strings"
	"testing"

	"github.com/orurh/patchcourt/internal/analysis/risk"
	"github.com/orurh/patchcourt/internal/diff/dep"
	"github.com/orurh/patchcourt/internal/model"
	"github.com/orurh/patchcourt/internal/reportmodel"
	"github.com/stretchr/testify/require"
)

func TestWriteReviewText_ShowsOnlyReviewRelevantDependencyChanges(t *testing.T) {
	var out bytes.Buffer

	WriteReviewText(&out, ReviewTextResult{
		DependencyChanges: []depdiff.DependencyChange{
			{
				Kind: depdiff.DependencyChangeKindAdded,
				Key:  "include|src/domain/a.h|string",
				After: &model.DependencyEdge{
					FromFile:  "src/domain/a.h",
					Target:    "string",
					FromLayer: "domain",
					Resolved:  true,
				},
			},
			{
				Kind: depdiff.DependencyChangeKindAdded,
				Key:  "include|src/domain/a.h|src/domain/b.h",
				After: &model.DependencyEdge{
					FromFile:  "src/domain/a.h",
					ToFile:    "src/domain/b.h",
					Target:    "src/domain/b.h",
					FromLayer: "domain",
					ToLayer:   "domain",
					Resolved:  true,
				},
			},
			{
				Kind: depdiff.DependencyChangeKindAdded,
				Key:  "include|src/cameras/a.h|src/domain/b.h",
				After: &model.DependencyEdge{
					FromFile:  "src/cameras/a.h",
					ToFile:    "src/domain/b.h",
					Target:    "src/domain/b.h",
					FromLayer: "cameras",
					ToLayer:   "domain",
					Resolved:  true,
				},
			},
		},
	})

	got := out.String()

	require.Contains(t, got, "Dependency changes:")
	require.Contains(t, got, "review-relevant: 1")
	require.Contains(t, got, "raw total:       3")
	require.Contains(t, got, "hidden low-level: 2")
	require.Contains(t, got, "include|src/cameras/a.h|src/domain/b.h")
	require.NotContains(t, got, "include|src/domain/a.h|string")
	require.NotContains(t, got, "include|src/domain/a.h|src/domain/b.h")
}

func TestWriteReviewText_PrintsVerdictBlock(t *testing.T) {
	var out bytes.Buffer

	WriteReviewText(&out, ReviewTextResult{
		Risk: risk.Score{
			Level: risk.LevelMedium,
		},
		Impact: reportmodel.ReviewImpactReport{
			Worse: []reportmodel.ReviewImpactItem{
				{
					Kind:   "contract_removed",
					Title:  "Removed public contract symbol",
					Detail: "method::ICameraManagerController::GetCameraStatus",
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
	})

	got := out.String()

	require.Contains(t, got, "Verdict:")
	require.Contains(t, got, "architecture: mixed")
	require.Contains(t, got, "risk:         medium")
	require.Contains(t, got, "main concerns:")
	require.Contains(t, got, "Removed public contract symbol")
	require.Contains(t, got, "improvements:")
	require.Contains(t, got, "Removed policy violation finding: architecture.domain.cameras")
	require.True(t, strings.Index(got, "Verdict:") < strings.Index(got, "Summary:"))
}

func TestWriteReviewText_VerdictPrefersHighSignalItems(t *testing.T) {
	var out bytes.Buffer

	WriteReviewText(&out, ReviewTextResult{
		Risk: risk.Score{
			Level: risk.LevelLow,
		},
		Impact: reportmodel.ReviewImpactReport{
			Better: []reportmodel.ReviewImpactItem{
				{
					Kind:  "finding_removed",
					Title: "Removed policy violation finding",
					ID:    "architecture.cli.platform",
				},
				{
					Kind:   "layer_edge_removed",
					Title:  "Removed layer dependency",
					Detail: "cli -> platform (1)",
				},
				{
					Kind:   "dependency_removed",
					Title:  "Removed dependency",
					ID:     "import|internal/cli/check.go|internal/platform/files/atomic.go",
					Detail: "internal/cli/check.go -> internal/platform/files/atomic.go (cli -> platform)",
				},
			},
		},
	})

	got := out.String()

	verdictStart := strings.Index(got, "Verdict:")
	require.NotEqual(t, -1, verdictStart)

	summaryStart := strings.Index(got, "Summary:")
	require.NotEqual(t, -1, summaryStart)

	verdict := got[verdictStart:summaryStart]

	require.Contains(t, verdict, "Removed policy violation finding: architecture.cli.platform")
	require.Contains(t, verdict, "Removed layer dependency — cli -> platform (1)")
	require.NotContains(t, verdict, "Removed dependency:")
}
