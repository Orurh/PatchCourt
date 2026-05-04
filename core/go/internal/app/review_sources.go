package app

import (
	"context"
	"fmt"

	"github.com/orurh/patchcourt/internal/changes"
	"github.com/orurh/patchcourt/internal/model"
)

func (a *App) loadReviewProjects(ctx context.Context, req ReviewRequest) (*model.ProjectModel, *model.ProjectModel, []string, error) {
	if req.Worktree {
		if req.HeadRef != "" {
			return nil, nil, nil, fmt.Errorf("--worktree cannot be combined with --head")
		}

		gitPair, err := changes.NewGitBaseToWorktreeSourcePair(ctx, changes.GitBaseToWorktreeSourcePairOptions{
			Root:       req.GitRoot,
			BaseRef:    req.BaseRef,
			ConfigPath: req.ConfigPath,
			Analyzer:   a.analysis,
		})
		if err != nil {
			return nil, nil, nil, err
		}
		defer func() {
			_ = gitPair.Cleanup(context.Background())
		}()

		beforeProject, afterProject, err := changes.LoadPair(ctx, gitPair.Pair)
		if err != nil {
			return nil, nil, nil, err
		}

		return beforeProject, afterProject, gitPair.ChangedFiles, nil
	}

	if req.BaseRef != "" || req.HeadRef != "" {
		gitPair, err := changes.NewGitReviewSourcePair(ctx, changes.GitReviewSourcePairOptions{
			Root:       req.GitRoot,
			BaseRef:    req.BaseRef,
			HeadRef:    req.HeadRef,
			ConfigPath: req.ConfigPath,
			Analyzer:   a.analysis,
		})
		if err != nil {
			return nil, nil, nil, err
		}
		defer func() {
			_ = gitPair.Cleanup(context.Background())
		}()

		beforeProject, afterProject, err := changes.LoadPair(ctx, gitPair.Pair)
		if err != nil {
			return nil, nil, nil, err
		}

		return beforeProject, afterProject, gitPair.ChangedFiles, nil
	}

	pair, err := a.reviewSourcePair(req)
	if err != nil {
		return nil, nil, nil, err
	}

	beforeProject, afterProject, err := changes.LoadPair(ctx, pair)
	if err != nil {
		if req.SinceLastRoot != "" {
			return nil, nil, nil, fmt.Errorf("%w. Run `patchcourt check %s --save-state` first", err, req.SinceLastRoot)
		}

		return nil, nil, nil, err
	}

	return beforeProject, afterProject, nil, nil
}

func (a *App) reviewSourcePair(req ReviewRequest) (changes.SourcePair, error) {
	if req.SinceLastRoot != "" {
		return changes.SourcePair{
			Before: changes.StateSource{
				Root: req.SinceLastRoot,
			},
			After: changes.RootSource{
				Root:       req.SinceLastRoot,
				ConfigPath: req.ConfigPath,
				Operation:  "review-since-last",
				Analyzer:   a.analysis,
			},
		}, nil
	}

	if req.BeforePath != "" || req.AfterPath != "" {
		if req.BeforePath == "" {
			return changes.SourcePair{}, fmt.Errorf("before project model path is required")
		}

		if req.AfterPath == "" {
			return changes.SourcePair{}, fmt.Errorf("after project model path is required")
		}

		return changes.SourcePair{
			Before: changes.SnapshotSource{
				Path: req.BeforePath,
			},
			After: changes.SnapshotSource{
				Path: req.AfterPath,
			},
		}, nil
	}

	if req.BeforeRoot == "" {
		return changes.SourcePair{}, fmt.Errorf("before root is required")
	}

	if req.AfterRoot == "" {
		return changes.SourcePair{}, fmt.Errorf("after root is required")
	}

	return changes.SourcePair{
		Before: changes.RootSource{
			Root:       req.BeforeRoot,
			ConfigPath: req.ConfigPath,
			Operation:  "review-before",
			Analyzer:   a.analysis,
		},
		After: changes.RootSource{
			Root:       req.AfterRoot,
			ConfigPath: req.ConfigPath,
			Operation:  "review-after",
			Analyzer:   a.analysis,
		},
	}, nil
}
