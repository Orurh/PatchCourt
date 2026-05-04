package git

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestClient_ChangedFiles(t *testing.T) {
	root := initGitDiffTestRepo(t)

	writeGitDiffTestFile(t, root, "a.txt", "one\n")
	runGitDiffTest(t, root, "add", ".")
	runGitDiffTest(t, root, "commit", "-m", "base")

	base := stringsTrim(runGitDiffTestOutput(t, root, "rev-parse", "HEAD"))

	writeGitDiffTestFile(t, root, "a.txt", "two\n")
	writeGitDiffTestFile(t, root, "b.txt", "new\n")
	runGitDiffTest(t, root, "add", ".")
	runGitDiffTest(t, root, "commit", "-m", "head")

	head := stringsTrim(runGitDiffTestOutput(t, root, "rev-parse", "HEAD"))

	files, err := NewClient(root).ChangedFiles(context.Background(), base, head)
	require.NoError(t, err)
	require.Equal(t, []string{"a.txt", "b.txt"}, files)
}

func TestClient_ChangedFilesToWorktree(t *testing.T) {
	root := initGitDiffTestRepo(t)

	writeGitDiffTestFile(t, root, "a.txt", "one\n")
	runGitDiffTest(t, root, "add", ".")
	runGitDiffTest(t, root, "commit", "-m", "base")

	base := stringsTrim(runGitDiffTestOutput(t, root, "rev-parse", "HEAD"))

	writeGitDiffTestFile(t, root, "a.txt", "two\n")
	writeGitDiffTestFile(t, root, "nested/b.txt", "new\n")

	files, err := NewClient(root).ChangedFilesToWorktree(context.Background(), base)
	require.NoError(t, err)
	require.Equal(t, []string{"a.txt", "nested/b.txt"}, files)
}

func initGitDiffTestRepo(t *testing.T) string {
	t.Helper()

	root := t.TempDir()

	runGitDiffTest(t, root, "init")
	runGitDiffTest(t, root, "config", "user.email", "test@example.com")
	runGitDiffTest(t, root, "config", "user.name", "Test User")

	return root
}

func writeGitDiffTestFile(t *testing.T, root string, relPath string, content string) {
	t.Helper()

	path := filepath.Join(root, filepath.FromSlash(relPath))
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
}

func runGitDiffTest(t *testing.T, dir string, args ...string) {
	t.Helper()
	_ = runGitDiffTestOutput(t, dir, args...)
}

func runGitDiffTestOutput(t *testing.T, dir string, args ...string) string {
	t.Helper()

	out, err := runGit(context.Background(), dir, args...)
	require.NoErrorf(t, err, "git %v failed: %s", args, out)

	return out
}

func TestClient_ChangedFilesToWorktree_ExcludesIgnoredUntrackedFiles(t *testing.T) {
	root := initGitDiffTestRepo(t)

	writeGitDiffTestFile(t, root, ".gitignore", "ignored.txt\n")
	writeGitDiffTestFile(t, root, "a.txt", "one\n")
	runGitDiffTest(t, root, "add", ".")
	runGitDiffTest(t, root, "commit", "-m", "base")

	base := stringsTrim(runGitDiffTestOutput(t, root, "rev-parse", "HEAD"))

	writeGitDiffTestFile(t, root, "ignored.txt", "ignored\n")
	writeGitDiffTestFile(t, root, "visible.txt", "visible\n")

	files, err := NewClient(root).ChangedFilesToWorktree(context.Background(), base)
	require.NoError(t, err)
	require.Equal(t, []string{"visible.txt"}, files)
}
