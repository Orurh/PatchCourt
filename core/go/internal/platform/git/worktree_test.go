package git

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFindRootAndReviewWorktrees(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed")
	}

	ctx := context.Background()
	repo := t.TempDir()

	runTestGit(t, repo, "init")
	runTestGit(t, repo, "config", "user.email", "patchcourt@example.test")
	runTestGit(t, repo, "config", "user.name", "PatchCourt Test")

	require.NoError(t, os.WriteFile(filepath.Join(repo, "file.txt"), []byte("base\n"), 0o644))
	runTestGit(t, repo, "add", "file.txt")
	runTestGit(t, repo, "commit", "-m", "base")
	baseSHA := stringsTrim(runTestGitOutput(t, repo, "rev-parse", "HEAD"))

	require.NoError(t, os.WriteFile(filepath.Join(repo, "file.txt"), []byte("head\n"), 0o644))
	runTestGit(t, repo, "commit", "-am", "head")
	headSHA := stringsTrim(runTestGitOutput(t, repo, "rev-parse", "HEAD"))

	root, err := FindRoot(ctx, filepath.Join(repo, "subdir"))
	require.Error(t, err)

	require.NoError(t, os.Mkdir(filepath.Join(repo, "subdir"), 0o755))
	root, err = FindRoot(ctx, filepath.Join(repo, "subdir"))
	require.NoError(t, err)
	requireSamePath(t, repo, root)

	client := NewClient(root)

	resolvedBase, err := client.ResolveRef(ctx, baseSHA)
	require.NoError(t, err)
	require.Equal(t, baseSHA, resolvedBase)

	worktrees, err := client.CreateReviewWorktrees(ctx, baseSHA, headSHA)
	require.NoError(t, err)
	require.DirExists(t, worktrees.Before)
	require.DirExists(t, worktrees.After)

	beforeData, err := os.ReadFile(filepath.Join(worktrees.Before, "file.txt"))
	require.NoError(t, err)
	require.Equal(t, "base\n", string(beforeData))

	afterData, err := os.ReadFile(filepath.Join(worktrees.After, "file.txt"))
	require.NoError(t, err)
	require.Equal(t, "head\n", string(afterData))

	require.NoError(t, worktrees.Cleanup(ctx))
	require.NoDirExists(t, worktrees.TempDir)
}

func runTestGit(t *testing.T, dir string, args ...string) {
	t.Helper()

	_ = runTestGitOutput(t, dir, args...)
}

func runTestGitOutput(t *testing.T, dir string, args ...string) string {
	t.Helper()

	cmd := exec.Command("git", args...)
	cmd.Dir = dir

	out, err := cmd.CombinedOutput()
	require.NoErrorf(t, err, "git %v failed: %s", args, string(out))

	return string(out)
}

func stringsTrim(value string) string {
	for len(value) > 0 && (value[len(value)-1] == '\n' || value[len(value)-1] == '\r' || value[len(value)-1] == ' ' || value[len(value)-1] == '\t') {
		value = value[:len(value)-1]
	}

	return value
}

func requireSamePath(t *testing.T, expected string, actual string) {
	t.Helper()

	expectedClean, err := filepath.EvalSymlinks(expected)
	require.NoError(t, err)

	actualClean, err := filepath.EvalSymlinks(actual)
	require.NoError(t, err)

	require.Equal(t, expectedClean, actualClean)
}
