package report

import (
	"bytes"
	"testing"

	"github.com/orurh/patchcourt/internal/analysis/depdiff"
	"github.com/orurh/patchcourt/internal/model"
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
