package explain

import (
	"context"
	"fmt"

	"github.com/orurh/patchcourt/internal/model"
	"github.com/orurh/patchcourt/internal/reportmodel"
	"github.com/orurh/patchcourt/internal/state"
	"github.com/orurh/patchcourt/internal/usecase/projectbuild"
)

type Result = reportmodel.ExplainResult

type Format string

const (
	FormatText Format = "text"
	FormatJSON Format = "json"
)

type Request struct {
	FindingID  string
	Root       string
	ConfigPath string
	ModelPath  string
}

type Service struct {
	Projects projectbuild.Builder
}

func NewService(projects projectbuild.Builder) Service {
	return Service{
		Projects: projects,
	}
}

func (s Service) Run(ctx context.Context, req Request) (*Result, error) {
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("explain canceled before start: %w", err)
	}

	if req.FindingID == "" {
		return nil, fmt.Errorf("finding id is required")
	}

	project, source, err := s.loadProject(ctx, req)
	if err != nil {
		return nil, err
	}

	finding, ok := findProjectFinding(project.Findings, req.FindingID)
	if !ok {
		return nil, fmt.Errorf("finding %q was not found", req.FindingID)
	}

	return &Result{
		Finding: finding,
		Source:  source,
	}, nil
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
		Operation:  "explain",
		Root:       root,
		ConfigPath: req.ConfigPath,
	})
	if err != nil {
		return nil, "", err
	}

	return result.Project, root, nil
}

func findProjectFinding(findings []model.Finding, id string) (model.Finding, bool) {
	for _, finding := range findings {
		if finding.ID == id {
			return finding, true
		}
	}

	return model.Finding{}, false
}
