package usecase

import (
	"context"

	"github.com/orurh/patchcourt/internal/engine"
)

type AnalysisService interface {
	Analyze(ctx context.Context, req engine.AnalyzeRequest) (*engine.AnalyzeResult, error)
}
