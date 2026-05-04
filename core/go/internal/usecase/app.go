package usecase

import (
	"github.com/orurh/patchcourt/internal/engine"
	"github.com/orurh/patchcourt/internal/platform/logx"
)

type App struct {
	logger   logx.Logger
	analysis AnalysisService

	projects ProjectBuilder
	scan     ScanService
	graph    GraphService
	edge     EdgeService
	explain  ExplainService
	check    CheckService
	init     InitService
	review   ReviewService
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

	projects := NewProjectBuilder(analysis)
	scan := NewScanService(projects)

	return &App{
		logger:   logger,
		analysis: analysis,

		projects: projects,
		scan:     scan,
		graph:    NewGraphService(projects, logger),
		edge:     NewEdgeService(projects),
		explain:  NewExplainService(projects),
		check:    NewCheckService(scan, logger),
		init:     NewInitService(logger),
		review:   NewReviewService(NewReviewProjectLoader(analysis)),
	}
}
