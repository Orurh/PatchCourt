package app

import (
	"context"
	"fmt"

	"github.com/orurh/patchcourt/internal/model"
)

type EdgeFormat string

const (
	EdgeFormatText EdgeFormat = "text"
	EdgeFormatJSON EdgeFormat = "json"
)

type EdgeRequest struct {
	Root       string `json:"root,omitempty"`
	ConfigPath string `json:"config_path,omitempty"`
	ModelPath  string `json:"model_path,omitempty"`
	FromLayer  string `json:"from_layer"`
	ToLayer    string `json:"to_layer"`
	Limit      int    `json:"limit,omitempty"`
}

type EdgeResult struct {
	Root          string                 `json:"root,omitempty"`
	Source        string                 `json:"source"`
	FromLayer     string                 `json:"from_layer"`
	ToLayer       string                 `json:"to_layer"`
	Count         int                    `json:"count"`
	Usage         EdgeUsageSummary       `json:"usage"`
	Findings      []model.Finding        `json:"findings,omitempty"`
	TopFromFiles  []EdgeFileCount        `json:"top_from_files,omitempty"`
	TopToFiles    []EdgeFileCount        `json:"top_to_files,omitempty"`
	Dependencies  []model.DependencyEdge `json:"dependencies,omitempty"`
	TruncatedDeps int                    `json:"truncated_deps,omitempty"`
}

type EdgeUsageSummary struct {
	Used    int `json:"used"`
	Maybe   int `json:"maybe"`
	Unused  int `json:"unused"`
	Unknown int `json:"unknown"`
}

type EdgeFileCount struct {
	File  string `json:"file"`
	Count int    `json:"count"`
}

func (a *App) RunEdge(ctx context.Context, req EdgeRequest) (*EdgeResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("edge canceled before start: %w", err)
	}

	if req.FromLayer == "" {
		return nil, fmt.Errorf("from layer is required")
	}

	if req.ToLayer == "" {
		return nil, fmt.Errorf("to layer is required")
	}

	project, source, err := a.loadEdgeProject(ctx, req)
	if err != nil {
		return nil, err
	}

	return BuildEdgeReport(project, EdgeReportOptions{
		Root:      project.Root,
		Source:    source,
		FromLayer: req.FromLayer,
		ToLayer:   req.ToLayer,
		Limit:     req.Limit,
	}), nil
}

func (a *App) loadEdgeProject(ctx context.Context, req EdgeRequest) (*model.ProjectModel, string, error) {
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
		Operation:  "edge",
		Root:       root,
		ConfigPath: req.ConfigPath,
	})
	if err != nil {
		return nil, "", err
	}

	return result.Project, root, nil
}
