package source

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNormalizeProjectChangedFiles_FiltersAndTrimsProjectPrefix(t *testing.T) {
	got := normalizeProjectChangedFiles([]string{
		"core/go/internal/usecase/review.go",
		"core/go/internal/source/git.go",
		"README.md",
	}, "core/go")

	require.Equal(t, []string{
		"internal/source/git.go",
		"internal/usecase/review.go",
	}, got)
}
