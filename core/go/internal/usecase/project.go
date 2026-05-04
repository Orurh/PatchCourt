package usecase

import (
	"context"

	"github.com/orurh/patchcourt/internal/config"
	"github.com/orurh/patchcourt/internal/engine"
	"github.com/orurh/patchcourt/internal/model"
)

type buildProjectRequest struct {
	Operation  string
	Root       string
	ConfigPath string
}

type buildProjectResult struct {
	Project *model.ProjectModel
	Config  *config.Config
}

func (a *App) buildProject(ctx context.Context, req buildProjectRequest) (*buildProjectResult, error) {
	result, err := a.analysis.Analyze(ctx, engine.AnalyzeRequest{
		Operation:  req.Operation,
		Root:       req.Root,
		ConfigPath: req.ConfigPath,
	})
	if err != nil {
		return nil, err
	}

	return &buildProjectResult{
		Project: result.Project,
		Config:  result.Config,
	}, nil
}
