package source

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	platformgit "github.com/orurh/patchcourt/internal/platform/git"
)

type GitReviewSourcePairOptions struct {
	Root       string
	BaseRef    string
	HeadRef    string
	ConfigPath string
	Analyzer   Analyzer
}

type GitBaseToWorktreeSourcePairOptions struct {
	Root       string
	BaseRef    string
	ConfigPath string
	Analyzer   Analyzer
}

type GitReviewSourcePair struct {
	Pair         SourcePair
	ChangedFiles []string
	cleanup      func(context.Context) error
}

func NewGitReviewSourcePair(ctx context.Context, opts GitReviewSourcePairOptions) (*GitReviewSourcePair, error) {
	if opts.BaseRef == "" {
		return nil, fmt.Errorf("base ref is required")
	}

	if opts.HeadRef == "" {
		return nil, fmt.Errorf("head ref is required")
	}

	gitRoot, projectRel, _, err := resolveGitProjectRoots(ctx, opts.Root)
	if err != nil {
		return nil, err
	}

	client := platformgit.NewClient(gitRoot)

	worktrees, err := client.CreateReviewWorktrees(ctx, opts.BaseRef, opts.HeadRef)
	if err != nil {
		return nil, err
	}

	changedFiles, err := client.ChangedFiles(ctx, opts.BaseRef, opts.HeadRef)
	if err != nil {
		_ = worktrees.Cleanup(context.Background())
		return nil, err
	}

	return &GitReviewSourcePair{
		ChangedFiles: normalizeProjectChangedFiles(changedFiles, projectRel),
		Pair: SourcePair{
			Before: RootSource{
				Root:       filepath.Join(worktrees.Before, projectRel),
				ConfigPath: opts.ConfigPath,
				Operation:  "review-base",
				Analyzer:   opts.Analyzer,
			},
			After: RootSource{
				Root:       filepath.Join(worktrees.After, projectRel),
				ConfigPath: opts.ConfigPath,
				Operation:  "review-head",
				Analyzer:   opts.Analyzer,
			},
		},
		cleanup: worktrees.Cleanup,
	}, nil
}

func NewGitBaseToWorktreeSourcePair(ctx context.Context, opts GitBaseToWorktreeSourcePairOptions) (*GitReviewSourcePair, error) {
	if opts.BaseRef == "" {
		return nil, fmt.Errorf("base ref is required")
	}

	gitRoot, projectRel, projectRoot, err := resolveGitProjectRoots(ctx, opts.Root)
	if err != nil {
		return nil, err
	}

	client := platformgit.NewClient(gitRoot)

	baseWorktree, err := client.CreateDetachedWorktree(ctx, opts.BaseRef, "before")
	if err != nil {
		return nil, err
	}

	changedFiles, err := client.ChangedFilesToWorktree(ctx, opts.BaseRef)
	if err != nil {
		_ = baseWorktree.Cleanup(context.Background())
		return nil, err
	}

	return &GitReviewSourcePair{
		ChangedFiles: normalizeProjectChangedFiles(changedFiles, projectRel),
		Pair: SourcePair{
			Before: RootSource{
				Root:       filepath.Join(baseWorktree.Path, projectRel),
				ConfigPath: opts.ConfigPath,
				Operation:  "review-base",
				Analyzer:   opts.Analyzer,
			},
			After: RootSource{
				Root:       projectRoot,
				ConfigPath: opts.ConfigPath,
				Operation:  "review-working-tree",
				Analyzer:   opts.Analyzer,
			},
		},
		cleanup: baseWorktree.Cleanup,
	}, nil
}

func (p *GitReviewSourcePair) Cleanup(ctx context.Context) error {
	if p == nil || p.cleanup == nil {
		return nil
	}

	return p.cleanup(ctx)
}

func resolveGitProjectRoots(ctx context.Context, root string) (gitRoot string, projectRel string, projectRoot string, err error) {
	if root == "" {
		root = "."
	}

	projectRoot, err = filepath.Abs(root)
	if err != nil {
		return "", "", "", fmt.Errorf("resolve project root: %w", err)
	}

	gitRoot, err = platformgit.FindRoot(ctx, projectRoot)
	if err != nil {
		return "", "", "", err
	}

	projectRel, err = filepath.Rel(gitRoot, projectRoot)
	if err != nil {
		return "", "", "", fmt.Errorf("resolve project path relative to git root: %w", err)
	}

	if projectRel == "." {
		projectRel = ""
	}

	return gitRoot, projectRel, projectRoot, nil
}

func normalizeProjectChangedFiles(files []string, projectRel string) []string {
	projectRel = filepath.ToSlash(projectRel)
	if projectRel == "." {
		projectRel = ""
	}

	seen := make(map[string]struct{}, len(files))

	for _, file := range files {
		file = filepath.ToSlash(file)
		file = strings.TrimSpace(file)
		if file == "" {
			continue
		}

		if projectRel != "" {
			prefix := projectRel + "/"
			rel, ok := strings.CutPrefix(file, prefix)
			if !ok {
				continue
			}
			file = rel
		}

		if file != "" {
			seen[file] = struct{}{}
		}
	}

	result := make([]string, 0, len(seen))
	for file := range seen {
		result = append(result, file)
	}

	sort.Strings(result)
	return result
}
