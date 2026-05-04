package git

import (
	"context"
	"fmt"
	"sort"
	"strings"
)

func (c Client) ChangedFiles(ctx context.Context, baseRef string, headRef string) ([]string, error) {
	if c.Root == "" {
		return nil, fmt.Errorf("git root is required")
	}
	if baseRef == "" {
		return nil, fmt.Errorf("base ref is required")
	}
	if headRef == "" {
		return nil, fmt.Errorf("head ref is required")
	}

	baseSHA, err := c.ResolveRef(ctx, baseRef)
	if err != nil {
		return nil, err
	}

	headSHA, err := c.ResolveRef(ctx, headRef)
	if err != nil {
		return nil, err
	}

	return c.changedFiles(ctx, baseSHA, headSHA)
}

func (c Client) ChangedFilesToWorktree(ctx context.Context, baseRef string) ([]string, error) {
	if c.Root == "" {
		return nil, fmt.Errorf("git root is required")
	}
	if baseRef == "" {
		return nil, fmt.Errorf("base ref is required")
	}

	baseSHA, err := c.ResolveRef(ctx, baseRef)
	if err != nil {
		return nil, err
	}

	diffOut, err := runGit(ctx, c.Root, "diff", "--name-only", "--diff-filter=ACMRTD", baseSHA, "--")
	if err != nil {
		return nil, fmt.Errorf("git diff changed files from %q to worktree: %w", baseRef, err)
	}

	untrackedOut, err := runGit(ctx, c.Root, "ls-files", "--others", "--exclude-standard")
	if err != nil {
		return nil, fmt.Errorf("git list untracked files: %w", err)
	}

	return mergeChangedFiles(parseChangedFiles(diffOut), parseChangedFiles(untrackedOut)), nil
}

func (c Client) changedFiles(ctx context.Context, baseSHA string, headSHA string) ([]string, error) {
	out, err := runGit(ctx, c.Root, "diff", "--name-only", "--diff-filter=ACMRTD", baseSHA, headSHA, "--")
	if err != nil {
		return nil, fmt.Errorf("git diff changed files %s..%s: %w", baseSHA, headSHA, err)
	}

	return parseChangedFiles(out), nil
}

func parseChangedFiles(out string) []string {
	seen := make(map[string]struct{})

	for _, line := range strings.Split(out, "\n") {
		file := strings.TrimSpace(line)
		if file == "" {
			continue
		}

		file = strings.ReplaceAll(file, "\\", "/")
		seen[file] = struct{}{}
	}

	files := make([]string, 0, len(seen))
	for file := range seen {
		files = append(files, file)
	}

	sort.Strings(files)
	return files
}

func mergeChangedFiles(left []string, right []string) []string {
	seen := make(map[string]struct{}, len(left)+len(right))

	for _, file := range left {
		if file == "" {
			continue
		}
		seen[file] = struct{}{}
	}

	for _, file := range right {
		if file == "" {
			continue
		}
		seen[file] = struct{}{}
	}

	files := make([]string, 0, len(seen))
	for file := range seen {
		files = append(files, file)
	}

	sort.Strings(files)
	return files
}
