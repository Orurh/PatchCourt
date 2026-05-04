package usecase

import (
	"context"

	"github.com/orurh/patchcourt/internal/config"
	"github.com/orurh/patchcourt/internal/model"
)

type ScanFormat string

const (
	ScanFormatText     ScanFormat = "text"
	ScanFormatMarkdown ScanFormat = "markdown"
	ScanFormatJSON     ScanFormat = "json"
)

type ScanRequest struct {
	Root       string
	ConfigPath string
}

type ScanResult struct {
	Project *model.ProjectModel
	Config  *config.Config
}

func (a *App) RunScan(ctx context.Context, req ScanRequest) (*ScanResult, error) {
	result, err := a.buildProject(ctx, buildProjectRequest{
		Operation:  "scan",
		Root:       req.Root,
		ConfigPath: req.ConfigPath,
	})
	if err != nil {
		return nil, err
	}

	return &ScanResult{
		Project: result.Project,
		Config:  result.Config,
	}, nil
}
