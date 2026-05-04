package edge

import (
	"context"
	"fmt"

	"github.com/orurh/patchcourt/internal/model"
	"github.com/orurh/patchcourt/internal/reportmodel"
	"github.com/orurh/patchcourt/internal/state"
	"github.com/orurh/patchcourt/internal/usecase/projectbuild"
)

type Result = reportmodel.EdgeResult
type UsageSummary = reportmodel.EdgeUsageSummary
type FileCount = reportmodel.EdgeFileCount

type Format string

const (
	FormatText Format = "text"
	FormatJSON Format = "json"
)

type Request struct {
	Root       string `json:"root,omitempty"`
	ConfigPath string `json:"config_path,omitempty"`
	ModelPath  string `json:"model_path,omitempty"`
	FromLayer  string `json:"from_layer"`
	ToLayer    string `json:"to_layer"`
	Limit      int    `json:"limit,omitempty"`
}

type ReportOptions struct {
	Root      string
	Source    string
	FromLayer string
	ToLayer   string
	Limit     int
}

type Service struct {
	Projects    projectbuild.Builder
	BuildReport func(project *model.ProjectModel, opts ReportOptions) *Result
}

func NewService(projects projectbuild.Builder, buildReport func(project *model.ProjectModel, opts ReportOptions) *Result) Service {
	if buildReport == nil {
		buildReport = func(project *model.ProjectModel, opts ReportOptions) *Result {
			return &Result{}
		}
	}

	return Service{
		Projects:    projects,
		BuildReport: buildReport,
	}
}

func (s Service) Run(ctx context.Context, req Request) (*Result, error) {
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("edge canceled before start: %w", err)
	}

	if req.FromLayer == "" {
		return nil, fmt.Errorf("from layer is required")
	}

	if req.ToLayer == "" {
		return nil, fmt.Errorf("to layer is required")
	}

	project, source, err := s.loadProject(ctx, req)
	if err != nil {
		return nil, err
	}

	return s.BuildReport(project, ReportOptions{
		Root:      project.Root,
		Source:    source,
		FromLayer: req.FromLayer,
		ToLayer:   req.ToLayer,
		Limit:     req.Limit,
	}), nil
}

func (s Service) loadProject(ctx context.Context, req Request) (*model.ProjectModel, string, error) {
	if req.ModelPath != "" {
		project, err := state.ReadProjectModel(req.ModelPath)
		if err != nil {
			return nil, "", fmt.Errorf("read project model: %w", err)
		}

		return project, req.ModelPath, nil
	}

	root := req.Root
	if root == "" {
		root = "."
	}

	result, err := s.Projects.Build(ctx, projectbuild.Request{
		Operation:  "edge",
		Root:       root,
		ConfigPath: req.ConfigPath,
	})
	if err != nil {
		return nil, "", err
	}

	return result.Project, root, nil
}
