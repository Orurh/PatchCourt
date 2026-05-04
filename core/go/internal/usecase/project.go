package usecase

import (
	"context"

	"github.com/orurh/patchcourt/internal/config"
	"github.com/orurh/patchcourt/internal/engine"
	"github.com/orurh/patchcourt/internal/model"
	"github.com/orurh/patchcourt/internal/usecase/ports"
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

type ProjectBuilder struct {
	Analysis ports.AnalysisService
}

func NewProjectBuilder(analysis ports.AnalysisService) ProjectBuilder {
	return ProjectBuilder{
		Analysis: analysis,
	}
}

func (b ProjectBuilder) Build(ctx context.Context, req buildProjectRequest) (*buildProjectResult, error) {
	result, err := b.Analysis.Analyze(ctx, engine.AnalyzeRequest{
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

// buildProject is kept as a temporary compatibility wrapper while use cases
// are still methods on App. New split usecase packages should use ProjectBuilder.
func (a *App) buildProject(ctx context.Context, req buildProjectRequest) (*buildProjectResult, error) {
	return NewProjectBuilder(a.analysis).Build(ctx, req)
}
