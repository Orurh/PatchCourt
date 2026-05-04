package files

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWriteFileAtomic_CreatesParentAndWritesFile(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "nested", "artifact.txt")

	err := WriteFileAtomic(path, []byte("hello"), 0o644)
	require.NoError(t, err)

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, "hello", string(data))
}

func TestWriteFileAtomic_ReplacesExistingFile(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "artifact.txt")

	require.NoError(t, os.WriteFile(path, []byte("old"), 0o644))
	require.NoError(t, WriteFileAtomic(path, []byte("new"), 0o644))

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, "new", string(data))
}

func TestWriteJSONAtomic_WritesReadableJSON(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "metadata.json")

	err := WriteJSONAtomic(path, map[string]any{
		"schema_version": 1,
		"root":           "/repo",
	})
	require.NoError(t, err)

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Contains(t, string(data), "\n  \"schema_version\": 1")

	var decoded map[string]any
	require.NoError(t, json.Unmarshal(data, &decoded))
	require.Equal(t, "/repo", decoded["root"])
}
