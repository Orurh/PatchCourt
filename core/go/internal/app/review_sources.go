package app

import (
	"context"
	"fmt"

	"github.com/orurh/patchcourt/internal/analysis/engine"
	"github.com/orurh/patchcourt/internal/changes"
	"github.com/orurh/patchcourt/internal/model"
)

func (a *App) loadReviewProjects(ctx context.Context, req ReviewRequest) (*model.ProjectModel, *model.ProjectModel, error) {
	if req.SinceLastRoot != "" {
		return a.loadReviewProjectsSinceLast(ctx, req)
	}

	if req.BeforePath != "" || req.AfterPath != "" {
		return loadReviewProjectsFromJSON(req)
	}

	return a.loadReviewProjectsFromRoots(ctx, req)
}

func loadReviewProjectsFromJSON(req ReviewRequest) (*model.ProjectModel, *model.ProjectModel, error) {
	if req.BeforePath == "" {
		return nil, nil, fmt.Errorf("before project model path is required")
	}

	if req.AfterPath == "" {
		return nil, nil, fmt.Errorf("after project model path is required")
	}

	beforeProject, err := changes.ReadProjectModel(req.BeforePath)
	if err != nil {
		return nil, nil, fmt.Errorf("read before project model: %w", err)
	}

	afterProject, err := changes.ReadProjectModel(req.AfterPath)
	if err != nil {
		return nil, nil, fmt.Errorf("read after project model: %w", err)
	}

	return beforeProject, afterProject, nil
}

func (a *App) loadReviewProjectsSinceLast(ctx context.Context, req ReviewRequest) (*model.ProjectModel, *model.ProjectModel, error) {
	state, err := changes.LoadState(changes.LoadStateOptions{
		Root: req.SinceLastRoot,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("load previous state: %w. Run `patchcourt check %s --save-state` first", err, req.SinceLastRoot)
	}

	afterResult, err := a.analysis.Analyze(ctx, engine.AnalyzeRequest{
		Operation:  "review-since-last",
		Root:       req.SinceLastRoot,
		ConfigPath: req.ConfigPath,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("analyze current root: %w", err)
	}

	return state.Project, afterResult.Project, nil
}

func (a *App) loadReviewProjectsFromRoots(ctx context.Context, req ReviewRequest) (*model.ProjectModel, *model.ProjectModel, error) {
	if req.BeforeRoot == "" {
		return nil, nil, fmt.Errorf("before root is required")
	}

	if req.AfterRoot == "" {
		return nil, nil, fmt.Errorf("after root is required")
	}

	beforeResult, err := a.analysis.Analyze(ctx, engine.AnalyzeRequest{
		Operation:  "review-before",
		Root:       req.BeforeRoot,
		ConfigPath: req.ConfigPath,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("analyze before root: %w", err)
	}

	afterResult, err := a.analysis.Analyze(ctx, engine.AnalyzeRequest{
		Operation:  "review-after",
		Root:       req.AfterRoot,
		ConfigPath: req.ConfigPath,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("analyze after root: %w", err)
	}

	return beforeResult.Project, afterResult.Project, nil
}
