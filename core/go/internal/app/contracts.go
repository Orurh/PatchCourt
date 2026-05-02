package app

import (
	"context"

	"github.com/orurh/patchcourt/internal/analysis/engine"
)

type AnalysisService interface {
	Analyze(ctx context.Context, req engine.AnalyzeRequest) (*engine.AnalyzeResult, error)
}
