package app

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/orurh/patchcourt/internal/analysis/contracts"
	"github.com/orurh/patchcourt/internal/analysis/engine"
	"github.com/orurh/patchcourt/internal/model"
)

type ReviewFormat string

const (
	ReviewFormatText ReviewFormat = "text"
	ReviewFormatJSON ReviewFormat = "json"
)

type ReviewRequest struct {
	BeforePath string
	AfterPath  string

	BeforeRoot string
	AfterRoot  string
	ConfigPath string
}

type ReviewResult struct {
	ContractChanges []contracts.SymbolChange `json:"contract_changes"`
}

func (a *App) RunReview(ctx context.Context, req ReviewRequest) (*ReviewResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("review canceled before start: %w", err)
	}

	beforeProject, afterProject, err := a.loadReviewProjects(ctx, req)
	if err != nil {
		return nil, err
	}

	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("review canceled after loading project models: %w", err)
	}

	return &ReviewResult{
		ContractChanges: contracts.DiffSymbols(beforeProject.Symbols, afterProject.Symbols),
	}, nil
}

func (a *App) loadReviewProjects(ctx context.Context, req ReviewRequest) (*model.ProjectModel, *model.ProjectModel, error) {
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

	beforeProject, err := readProjectModelJSON(req.BeforePath)
	if err != nil {
		return nil, nil, fmt.Errorf("read before project model: %w", err)
	}

	afterProject, err := readProjectModelJSON(req.AfterPath)
	if err != nil {
		return nil, nil, fmt.Errorf("read after project model: %w", err)
	}

	return beforeProject, afterProject, nil
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

func readProjectModelJSON(path string) (*model.ProjectModel, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}

	var project model.ProjectModel
	if err := json.Unmarshal(data, &project); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}

	return &project, nil
}
