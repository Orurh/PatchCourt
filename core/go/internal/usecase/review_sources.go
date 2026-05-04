package usecase

import (
	"context"
	"fmt"

	"github.com/orurh/patchcourt/internal/model"
	"github.com/orurh/patchcourt/internal/source"
	"github.com/orurh/patchcourt/internal/usecase/ports"
)

type ReviewProjectLoader struct {
	Analysis ports.AnalysisService
}

func NewReviewProjectLoader(analysis ports.AnalysisService) ReviewProjectLoader {
	return ReviewProjectLoader{
		Analysis: analysis,
	}
}

func (l ReviewProjectLoader) LoadProjects(ctx context.Context, req ReviewRequest) (*model.ProjectModel, *model.ProjectModel, []string, error) {
	if req.Worktree {
		if req.HeadRef != "" {
			return nil, nil, nil, fmt.Errorf("--worktree cannot be combined with --head")
		}

		gitPair, err := source.NewGitBaseToWorktreeSourcePair(ctx, source.GitBaseToWorktreeSourcePairOptions{
			Root:       req.GitRoot,
			BaseRef:    req.BaseRef,
			ConfigPath: req.ConfigPath,
			Analyzer:   l.Analysis,
		})
		if err != nil {
			return nil, nil, nil, err
		}
		defer func() {
			_ = gitPair.Cleanup(context.Background())
		}()

		beforeProject, afterProject, err := source.LoadPair(ctx, gitPair.Pair)
		if err != nil {
			return nil, nil, nil, err
		}

		return beforeProject, afterProject, gitPair.ChangedFiles, nil
	}

	if req.BaseRef != "" || req.HeadRef != "" {
		gitPair, err := source.NewGitReviewSourcePair(ctx, source.GitReviewSourcePairOptions{
			Root:       req.GitRoot,
			BaseRef:    req.BaseRef,
			HeadRef:    req.HeadRef,
			ConfigPath: req.ConfigPath,
			Analyzer:   l.Analysis,
		})
		if err != nil {
			return nil, nil, nil, err
		}
		defer func() {
			_ = gitPair.Cleanup(context.Background())
		}()

		beforeProject, afterProject, err := source.LoadPair(ctx, gitPair.Pair)
		if err != nil {
			return nil, nil, nil, err
		}

		return beforeProject, afterProject, gitPair.ChangedFiles, nil
	}

	pair, err := l.SourcePair(req)
	if err != nil {
		return nil, nil, nil, err
	}

	beforeProject, afterProject, err := source.LoadPair(ctx, pair)
	if err != nil {
		if req.SinceLastRoot != "" {
			return nil, nil, nil, fmt.Errorf("%w. Run `patchcourt check %s --save-state` first", err, req.SinceLastRoot)
		}

		return nil, nil, nil, err
	}

	return beforeProject, afterProject, nil, nil
}

func (l ReviewProjectLoader) SourcePair(req ReviewRequest) (source.SourcePair, error) {
	if req.SinceLastRoot != "" {
		return source.SourcePair{
			Before: source.StateSource{
				Root: req.SinceLastRoot,
			},
			After: source.RootSource{
				Root:       req.SinceLastRoot,
				ConfigPath: req.ConfigPath,
				Operation:  "review-since-last",
				Analyzer:   l.Analysis,
			},
		}, nil
	}

	if req.BeforePath != "" || req.AfterPath != "" {
		if req.BeforePath == "" {
			return source.SourcePair{}, fmt.Errorf("before project model path is required")
		}

		if req.AfterPath == "" {
			return source.SourcePair{}, fmt.Errorf("after project model path is required")
		}

		return source.SourcePair{
			Before: source.SnapshotSource{
				Path: req.BeforePath,
			},
			After: source.SnapshotSource{
				Path: req.AfterPath,
			},
		}, nil
	}

	if req.BeforeRoot == "" {
		return source.SourcePair{}, fmt.Errorf("before root is required")
	}

	if req.AfterRoot == "" {
		return source.SourcePair{}, fmt.Errorf("after root is required")
	}

	return source.SourcePair{
		Before: source.RootSource{
			Root:       req.BeforeRoot,
			ConfigPath: req.ConfigPath,
			Operation:  "review-before",
			Analyzer:   l.Analysis,
		},
		After: source.RootSource{
			Root:       req.AfterRoot,
			ConfigPath: req.ConfigPath,
			Operation:  "review-after",
			Analyzer:   l.Analysis,
		},
	}, nil
}
