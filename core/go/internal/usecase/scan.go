package usecase

import (
	"context"

	"github.com/orurh/patchcourt/internal/config"
	"github.com/orurh/patchcourt/internal/model"
)

type ScanFormat string

const (
	ScanFormatText     ScanFormat = "text"
	ScanFormatJSON     ScanFormat = "json"
	ScanFormatMarkdown ScanFormat = "markdown"
)

type ScanRequest struct {
	Root       string
	ConfigPath string
	Format     ScanFormat
}

type ScanResult struct {
	Project *model.ProjectModel
	Config  *config.Config
}

type ScanService struct {
	Projects ProjectBuilder
}

func NewScanService(projects ProjectBuilder) ScanService {
	return ScanService{
		Projects: projects,
	}
}

func (s ScanService) Run(ctx context.Context, req ScanRequest) (*ScanResult, error) {
	result, err := s.Projects.Build(ctx, buildProjectRequest{
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

func (a *App) RunScan(ctx context.Context, req ScanRequest) (*ScanResult, error) {
	return NewScanService(NewProjectBuilder(a.analysis)).Run(ctx, req)
}
