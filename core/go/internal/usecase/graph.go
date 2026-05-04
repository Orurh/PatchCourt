package usecase

import (
	"context"

	graphmodel "github.com/orurh/patchcourt/internal/analyzer/graph"
	"github.com/orurh/patchcourt/internal/config"
	"github.com/orurh/patchcourt/internal/model"
	"github.com/orurh/patchcourt/internal/platform/logx"
)

type GraphFormat string

const (
	GraphFormatJSON    GraphFormat = "json"
	GraphFormatMermaid GraphFormat = "mermaid"
	GraphFormatDOT     GraphFormat = "dot"
)

type GraphRequest struct {
	Root       string
	ConfigPath string
}

type GraphResult struct {
	Project    *model.ProjectModel
	Config     *config.Config
	LayerGraph graphmodel.LayerGraph
}

type GraphService struct {
	Projects ProjectBuilder
	Logger   logx.Logger
}

func NewGraphService(projects ProjectBuilder, logger logx.Logger) GraphService {
	return GraphService{
		Projects: projects,
		Logger:   logger,
	}
}

func (s GraphService) Run(ctx context.Context, req GraphRequest) (*GraphResult, error) {
	result, err := s.Projects.Build(ctx, buildProjectRequest{
		Operation:  "graph",
		Root:       req.Root,
		ConfigPath: req.ConfigPath,
	})
	if err != nil {
		return nil, err
	}

	layerGraph := graphmodel.BuildLayerGraph(result.Project, result.Config)

	if s.Logger != nil {
		s.Logger.Debug(
			"graph completed",
			logx.Int("nodes", len(layerGraph.Nodes)),
			logx.Int("edges", len(layerGraph.Edges)),
		)
	}

	return &GraphResult{
		Project:    result.Project,
		Config:     result.Config,
		LayerGraph: layerGraph,
	}, nil
}

func (a *App) RunGraph(ctx context.Context, req GraphRequest) (*GraphResult, error) {
	return NewGraphService(NewProjectBuilder(a.analysis), a.logger).Run(ctx, req)
}
