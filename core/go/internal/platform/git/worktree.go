package git

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type Client struct {
	Root string
}

type ReviewWorktrees struct {
	GitRoot string
	TempDir string
	Before  string
	After   string
}

type DetachedWorktree struct {
	GitRoot string
	TempDir string
	Path    string
}

func FindRoot(ctx context.Context, start string) (string, error) {
	if start == "" {
		start = "."
	}

	out, err := runGit(ctx, start, "rev-parse", "--show-toplevel")
	if err != nil {
		return "", fmt.Errorf("find git root from %s: %w", start, err)
	}

	root := strings.TrimSpace(out)
	if root == "" {
		return "", fmt.Errorf("git root is empty")
	}

	return root, nil
}

func NewClient(root string) Client {
	return Client{Root: root}
}

func (c Client) ResolveRef(ctx context.Context, ref string) (string, error) {
	if ref == "" {
		return "", fmt.Errorf("git ref is required")
	}

	out, err := runGit(ctx, c.Root, "rev-parse", "--verify", ref+"^{commit}")
	if err != nil {
		return "", fmt.Errorf("resolve git ref %q: %w", ref, err)
	}

	sha := strings.TrimSpace(out)
	if sha == "" {
		return "", fmt.Errorf("git ref %q resolved to empty sha", ref)
	}

	return sha, nil
}

func (c Client) CreateDetachedWorktree(ctx context.Context, ref string, name string) (*DetachedWorktree, error) {
	if c.Root == "" {
		return nil, fmt.Errorf("git root is required")
	}

	sha, err := c.ResolveRef(ctx, ref)
	if err != nil {
		return nil, err
	}

	if name == "" {
		name = "worktree"
	}

	tempDir, err := os.MkdirTemp("", "patchcourt-review-*")
	if err != nil {
		return nil, fmt.Errorf("create temporary review dir: %w", err)
	}

	worktree := &DetachedWorktree{
		GitRoot: c.Root,
		TempDir: tempDir,
		Path:    filepath.Join(tempDir, name),
	}

	if err := c.addDetachedWorktree(ctx, worktree.Path, sha); err != nil {
		_ = worktree.Cleanup(context.Background())
		return nil, fmt.Errorf("create detached worktree for %q: %w", ref, err)
	}

	return worktree, nil
}

func (c Client) CreateReviewWorktrees(ctx context.Context, baseRef string, headRef string) (*ReviewWorktrees, error) {
	if c.Root == "" {
		return nil, fmt.Errorf("git root is required")
	}

	baseSHA, err := c.ResolveRef(ctx, baseRef)
	if err != nil {
		return nil, err
	}

	headSHA, err := c.ResolveRef(ctx, headRef)
	if err != nil {
		return nil, err
	}

	tempDir, err := os.MkdirTemp("", "patchcourt-review-*")
	if err != nil {
		return nil, fmt.Errorf("create temporary review dir: %w", err)
	}

	worktrees := &ReviewWorktrees{
		GitRoot: c.Root,
		TempDir: tempDir,
		Before:  filepath.Join(tempDir, "before"),
		After:   filepath.Join(tempDir, "after"),
	}

	if err := c.addDetachedWorktree(ctx, worktrees.Before, baseSHA); err != nil {
		_ = worktrees.Cleanup(context.Background())
		return nil, fmt.Errorf("create base worktree for %q: %w", baseRef, err)
	}

	if err := c.addDetachedWorktree(ctx, worktrees.After, headSHA); err != nil {
		_ = worktrees.Cleanup(context.Background())
		return nil, fmt.Errorf("create head worktree for %q: %w", headRef, err)
	}

	return worktrees, nil
}

func (c Client) addDetachedWorktree(ctx context.Context, path string, sha string) error {
	_, err := runGit(ctx, c.Root, "worktree", "add", "--detach", path, sha)
	if err != nil {
		return fmt.Errorf("git worktree add --detach %s %s: %w", path, sha, err)
	}

	return nil
}

func (w *DetachedWorktree) Cleanup(ctx context.Context) error {
	if w == nil {
		return nil
	}

	var errs []string

	if w.GitRoot != "" && w.Path != "" {
		if _, err := runGit(ctx, w.GitRoot, "worktree", "remove", "--force", w.Path); err != nil {
			errs = append(errs, err.Error())
		}
	}

	if w.TempDir != "" {
		if err := os.RemoveAll(w.TempDir); err != nil {
			errs = append(errs, err.Error())
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("cleanup detached worktree: %s", strings.Join(errs, "; "))
	}

	return nil
}

func (w *ReviewWorktrees) Cleanup(ctx context.Context) error {
	if w == nil {
		return nil
	}

	var errs []string

	if w.GitRoot != "" && w.Before != "" {
		if _, err := runGit(ctx, w.GitRoot, "worktree", "remove", "--force", w.Before); err != nil {
			errs = append(errs, err.Error())
		}
	}

	if w.GitRoot != "" && w.After != "" {
		if _, err := runGit(ctx, w.GitRoot, "worktree", "remove", "--force", w.After); err != nil {
			errs = append(errs, err.Error())
		}
	}

	if w.TempDir != "" {
		if err := os.RemoveAll(w.TempDir); err != nil {
			errs = append(errs, err.Error())
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("cleanup review worktrees: %s", strings.Join(errs, "; "))
	}

	return nil
}

func runGit(ctx context.Context, dir string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	if dir != "" {
		cmd.Dir = dir
	}

	out, err := cmd.CombinedOutput()
	if err != nil {
		message := strings.TrimSpace(string(out))
		if message == "" {
			message = err.Error()
		}
		return "", fmt.Errorf("git %s: %s", strings.Join(args, " "), message)
	}

	return string(out), nil
}
