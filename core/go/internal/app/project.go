package app

import (
	"context"
	"fmt"

	"github.com/orurh/patchcourt/internal/analysis/project"
	"github.com/orurh/patchcourt/internal/analysis/rules"
	"github.com/orurh/patchcourt/internal/config"
	"github.com/orurh/patchcourt/internal/model"
	"github.com/orurh/patchcourt/internal/platform/logx"
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
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("%s canceled before start: %w", req.Operation, err)
	}

	logger := a.logger.With(
		logx.String("operation", req.Operation),
		logx.String("root", req.Root),
		logx.String("config_path", req.ConfigPath),
	)

	logger.Debug("loading config")

	cfg, err := config.Load(req.ConfigPath)
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}

	logger.Debug("scanning project")

	projectModel, err := project.Build(project.Options{
		Root:            req.Root,
		IgnorePaths:     cfg.Ignore.Paths,
		CPPIncludePaths: cfg.CPP.IncludePaths,
	})
	if err != nil {
		return nil, fmt.Errorf("scan project: %w", err)
	}

	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("%s canceled after project indexing: %w", req.Operation, err)
	}

	logger.Debug(
		"applying architecture rules",
		logx.Int("files", len(projectModel.Files)),
		logx.Int("dependencies", len(projectModel.Dependencies)),
	)

	rules.ApplyArchitectureRules(projectModel, cfg)

	logger.Debug("project build completed", logx.Int("findings", len(projectModel.Findings)))

	return &buildProjectResult{
		Project: projectModel,
		Config:  cfg,
	}, nil
}
