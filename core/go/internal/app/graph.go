package app

import (
	"context"

	graphmodel "github.com/orurh/patchcourt/internal/analysis/graph"
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

func (a *App) RunGraph(ctx context.Context, req GraphRequest) (*GraphResult, error) {
	result, err := a.buildProject(ctx, buildProjectRequest{
		Operation:  "graph",
		Root:       req.Root,
		ConfigPath: req.ConfigPath,
	})
	if err != nil {
		return nil, err
	}

	layerGraph := graphmodel.BuildLayerGraph(result.Project, result.Config)

	a.logger.Debug(
		"graph completed",
		logx.Int("nodes", len(layerGraph.Nodes)),
		logx.Int("edges", len(layerGraph.Edges)),
	)

	return &GraphResult{
		Project:    result.Project,
		Config:     result.Config,
		LayerGraph: layerGraph,
	}, nil
}
