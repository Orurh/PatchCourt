package app

import (
	"github.com/orurh/patchcourt/internal/analysis/engine"
	"github.com/orurh/patchcourt/internal/platform/logx"
)

type App struct {
	logger   logx.Logger
	analysis AnalysisService
}

func New(logger logx.Logger) *App {
	return NewWithAnalysis(logger, nil)
}

func NewWithAnalysis(logger logx.Logger, analysis AnalysisService) *App {
	if logger == nil {
		logger = logx.Nop()
	}

	if analysis == nil {
		analysis = engine.New(engine.Options{
			Logger: logger,
		})
	}

	return &App{
		logger:   logger,
		analysis: analysis,
	}
}
