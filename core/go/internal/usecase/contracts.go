package usecase

import "github.com/orurh/patchcourt/internal/usecase/ports"

// AnalysisService is kept as a compatibility alias for the current usecase facade.
// New split usecase packages should depend on ports.AnalysisService directly.
type AnalysisService = ports.AnalysisService
