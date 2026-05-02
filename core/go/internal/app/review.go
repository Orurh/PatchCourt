package app

import (
	"context"
	"fmt"

	"github.com/orurh/patchcourt/internal/analysis/contracts"
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
