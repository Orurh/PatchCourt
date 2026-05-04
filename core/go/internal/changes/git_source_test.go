package changes

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNormalizeProjectChangedFiles_FiltersAndTrimsProjectPrefix(t *testing.T) {
	got := normalizeProjectChangedFiles([]string{
		"core/go/internal/app/review.go",
		"core/go/internal/changes/git_source.go",
		"README.md",
	}, "core/go")

	require.Equal(t, []string{
		"internal/app/review.go",
		"internal/changes/git_source.go",
	}, got)
}
