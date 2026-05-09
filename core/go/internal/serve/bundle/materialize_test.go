package bundle

import (
	"archive/tar"
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestArchiveEntryTarget_AllowsCleanRelativePath(t *testing.T) {
	outDir := t.TempDir()

	target, err := archiveEntryTarget(outDir, "src/module/file.cc")

	require.NoError(t, err)
	require.Equal(t, filepath.Join(outDir, "src", "module", "file.cc"), target)
}

func TestArchiveEntryTarget_CleansRelativePath(t *testing.T) {
	outDir := t.TempDir()

	target, err := archiveEntryTarget(outDir, "src/../file.cc")

	require.NoError(t, err)
	require.Equal(t, filepath.Join(outDir, "file.cc"), target)
}

func TestArchiveEntryTarget_RejectsUnsafePaths(t *testing.T) {
	outDir := t.TempDir()

	cases := []string{
		".",
		"..",
		"../evil.cc",
		"src/../../evil.cc",
		filepath.Join(string(filepath.Separator), "tmp", "evil.cc"),
	}

	for _, tc := range cases {
		t.Run(tc, func(t *testing.T) {
			_, err := archiveEntryTarget(outDir, tc)
			require.Error(t, err)
		})
	}
}

func TestExtractGitArchiveEntry_WritesRegularFile(t *testing.T) {
	outDir := t.TempDir()
	body := []byte("hello from archive")

	header := &tar.Header{
		Name:     "src/file.cc",
		Typeflag: tar.TypeReg,
		Mode:     0o644,
		Size:     int64(len(body)),
	}

	err := extractGitArchiveEntry(bytes.NewReader(body), header, outDir)

	require.NoError(t, err)

	data, err := os.ReadFile(filepath.Join(outDir, "src", "file.cc"))
	require.NoError(t, err)
	require.Equal(t, body, data)
}

func TestExtractGitArchiveEntry_CreatesDirectory(t *testing.T) {
	outDir := t.TempDir()

	header := &tar.Header{
		Name:     "src/generated",
		Typeflag: tar.TypeDir,
		Mode:     0o755,
	}

	err := extractGitArchiveEntry(bytes.NewReader(nil), header, outDir)

	require.NoError(t, err)

	info, err := os.Stat(filepath.Join(outDir, "src", "generated"))
	require.NoError(t, err)
	require.True(t, info.IsDir())
}

func TestExtractGitArchiveEntry_RejectsUnsafePath(t *testing.T) {
	outDir := t.TempDir()

	header := &tar.Header{
		Name:     "../evil.cc",
		Typeflag: tar.TypeReg,
		Mode:     0o644,
	}

	err := extractGitArchiveEntry(bytes.NewReader([]byte("evil")), header, outDir)

	require.Error(t, err)
}
