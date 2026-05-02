package app

import (
	"context"
	"fmt"

	"github.com/orurh/patchcourt/internal/model"
)

type ExplainFormat string

const (
	ExplainFormatText ExplainFormat = "text"
	ExplainFormatJSON ExplainFormat = "json"
)

type ExplainRequest struct {
	FindingID  string
	Root       string
	ConfigPath string
	ModelPath  string
}

type ExplainResult struct {
	Finding model.Finding `json:"finding"`
	Source  string        `json:"source"`
}

func (a *App) RunExplain(ctx context.Context, req ExplainRequest) (*ExplainResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("explain canceled before start: %w", err)
	}

	if req.FindingID == "" {
		return nil, fmt.Errorf("finding id is required")
	}

	project, source, err := a.loadExplainProject(ctx, req)
	if err != nil {
		return nil, err
	}

	finding, ok := findProjectFinding(project.Findings, req.FindingID)
	if !ok {
		return nil, fmt.Errorf("finding %q was not found", req.FindingID)
	}

	return &ExplainResult{
		Finding: finding,
		Source:  source,
	}, nil
}

func (a *App) loadExplainProject(ctx context.Context, req ExplainRequest) (*model.ProjectModel, string, error) {
	if req.ModelPath != "" {
		project, err := readProjectModelJSON(req.ModelPath)
		if err != nil {
			return nil, "", fmt.Errorf("read project model: %w", err)
		}

		return project, req.ModelPath, nil
	}

	root := req.Root
	if root == "" {
		root = "."
	}

	result, err := a.buildProject(ctx, buildProjectRequest{
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
