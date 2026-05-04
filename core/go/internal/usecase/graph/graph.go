package graph

import (
	"context"

	graphmodel "github.com/orurh/patchcourt/internal/analyzer/graph"
	"github.com/orurh/patchcourt/internal/config"
	"github.com/orurh/patchcourt/internal/model"
	"github.com/orurh/patchcourt/internal/platform/logx"
	"github.com/orurh/patchcourt/internal/usecase/projectbuild"
)

type Format string

const (
	FormatJSON    Format = "json"
	FormatMermaid Format = "mermaid"
	FormatDOT     Format = "dot"
)

type Request struct {
	Root       string
	ConfigPath string
}

type Result struct {
	Project    *model.ProjectModel
	Config     *config.Config
	LayerGraph graphmodel.LayerGraph
}

type Service struct {
	Projects projectbuild.Builder
	Logger   logx.Logger
}

func NewService(projects projectbuild.Builder, logger logx.Logger) Service {
	if logger == nil {
		logger = logx.Nop()
	}

	return Service{
		Projects: projects,
		Logger:   logger,
	}
}

func (s Service) Run(ctx context.Context, req Request) (*Result, error) {
	result, err := s.Projects.Build(ctx, projectbuild.Request{
		Operation:  "graph",
		Root:       req.Root,
		ConfigPath: req.ConfigPath,
	})
	if err != nil {
		return nil, err
	}

	layerGraph := graphmodel.BuildLayerGraph(result.Project, result.Config)

	s.Logger.Debug(
		"graph completed",
		logx.Int("nodes", len(layerGraph.Nodes)),
		logx.Int("edges", len(layerGraph.Edges)),
	)

	return &Result{
		Project:    result.Project,
		Config:     result.Config,
		LayerGraph: layerGraph,
	}, nil
}
