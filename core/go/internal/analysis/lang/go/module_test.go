package goanalysis

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestModulePath(t *testing.T) {
	root := t.TempDir()

	require.NoError(t, os.WriteFile(filepath.Join(root, "go.mod"), []byte(`module github.com/orurh/patchcourt

go 1.26
`), 0o644))

	require.Equal(t, "github.com/orurh/patchcourt", ModulePath(root))
}
