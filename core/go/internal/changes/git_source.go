package changes

import (
	"context"
	"fmt"

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
	Pair    SourcePair
	cleanup func(context.Context) error
}

func NewGitReviewSourcePair(ctx context.Context, opts GitReviewSourcePairOptions) (*GitReviewSourcePair, error) {
	if opts.BaseRef == "" {
		return nil, fmt.Errorf("base ref is required")
	}

	if opts.HeadRef == "" {
		return nil, fmt.Errorf("head ref is required")
	}

	gitRoot, err := platformgit.FindRoot(ctx, opts.Root)
	if err != nil {
		return nil, err
	}

	client := platformgit.NewClient(gitRoot)

	worktrees, err := client.CreateReviewWorktrees(ctx, opts.BaseRef, opts.HeadRef)
	if err != nil {
		return nil, err
	}

	return &GitReviewSourcePair{
		Pair: SourcePair{
			Before: RootSource{
				Root:       worktrees.Before,
				ConfigPath: opts.ConfigPath,
				Operation:  "review-base",
				Analyzer:   opts.Analyzer,
			},
			After: RootSource{
				Root:       worktrees.After,
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

	gitRoot, err := platformgit.FindRoot(ctx, opts.Root)
	if err != nil {
		return nil, err
	}

	client := platformgit.NewClient(gitRoot)

	baseWorktree, err := client.CreateDetachedWorktree(ctx, opts.BaseRef, "before")
	if err != nil {
		return nil, err
	}

	return &GitReviewSourcePair{
		Pair: SourcePair{
			Before: RootSource{
				Root:       baseWorktree.Path,
				ConfigPath: opts.ConfigPath,
				Operation:  "review-base",
				Analyzer:   opts.Analyzer,
			},
			After: RootSource{
				Root:       gitRoot,
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
