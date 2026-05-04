package goanalysis

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseImports(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "main.go")

	require.NoError(t, os.WriteFile(path, []byte(`package main

import (
	"context"
	"fmt"

	"github.com/orurh/patchcourt/internal/usecase"
)

func main() {}
`), 0o644))

	got, err := ParseImports(path)
	require.NoError(t, err)
	require.Equal(t, []string{
		"context",
		"fmt",
		"github.com/orurh/patchcourt/internal/usecase",
	}, got)
}
