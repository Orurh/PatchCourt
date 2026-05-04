package ports

import (
	"context"

	"github.com/orurh/patchcourt/internal/engine"
)

// AnalysisService is the analysis engine port used by use cases.
//
// Use cases depend on this interface instead of a concrete analyzer/engine.
// That keeps the application layer testable and lets CLI/HTTP/background
// adapters reuse the same structured results without knowing analyzer internals.
type AnalysisService interface {
	Analyze(ctx context.Context, req engine.AnalyzeRequest) (*engine.AnalyzeResult, error)
}
