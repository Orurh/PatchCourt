package app

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/orurh/patchcourt/internal/analysis/contracts"
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
}

type ReviewResult struct {
	ContractChanges []contracts.SymbolChange `json:"contract_changes"`
}

func (a *App) RunReview(ctx context.Context, req ReviewRequest) (*ReviewResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("review canceled before start: %w", err)
	}

	if req.BeforePath == "" {
		return nil, fmt.Errorf("before project model path is required")
	}

	if req.AfterPath == "" {
		return nil, fmt.Errorf("after project model path is required")
	}

	beforeProject, err := readProjectModelJSON(req.BeforePath)
	if err != nil {
		return nil, fmt.Errorf("read before project model: %w", err)
	}

	afterProject, err := readProjectModelJSON(req.AfterPath)
	if err != nil {
		return nil, fmt.Errorf("read after project model: %w", err)
	}

	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("review canceled after loading project models: %w", err)
	}

	return &ReviewResult{
		ContractChanges: contracts.DiffSymbols(beforeProject.Symbols, afterProject.Symbols),
	}, nil
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
