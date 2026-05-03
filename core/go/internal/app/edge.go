package app

import (
	"context"
	"fmt"
	"sort"
	"strings"

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

	limit := req.Limit
	if limit <= 0 {
		limit = 50
	}

	deps := filterEdgeDependencies(project, req.FromLayer, req.ToLayer)
	sortEdgeDependencies(deps)

	result := &EdgeResult{
		Root:         project.Root,
		Source:       source,
		FromLayer:    req.FromLayer,
		ToLayer:      req.ToLayer,
		Count:        len(deps),
		Usage:        summarizeEdgeUsage(deps),
		Findings:     findEdgeFindings(project.Findings, req.FromLayer, req.ToLayer),
		TopFromFiles: topEdgeFiles(deps, true),
		TopToFiles:   topEdgeFiles(deps, false),
	}

	if len(deps) > limit {
		result.Dependencies = deps[:limit]
		result.TruncatedDeps = len(deps) - limit
	} else {
		result.Dependencies = deps
	}

	return result, nil
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

func filterEdgeDependencies(project *model.ProjectModel, fromLayer string, toLayer string) []model.DependencyEdge {
	if project == nil {
		return nil
	}

	ignoredFiles := ignoredEdgeFromFiles(project.Files)
	result := make([]model.DependencyEdge, 0)

	for _, dep := range project.Dependencies {
		if ignoredFiles[dep.FromFile] {
			continue
		}

		if dep.External || !dep.Resolved {
			continue
		}

		if dep.FromLayer != fromLayer || dep.ToLayer != toLayer {
			continue
		}

		result = append(result, dep)
	}

	return result
}

func ignoredEdgeFromFiles(files []model.FileModel) map[string]bool {
	ignored := make(map[string]bool)

	for _, file := range files {
		switch file.Role {
		case model.FileRoleTest, model.FileRoleGenerated, model.FileRoleExternal:
			ignored[file.Path] = true
		}
	}

	return ignored
}

func summarizeEdgeUsage(deps []model.DependencyEdge) EdgeUsageSummary {
	var summary EdgeUsageSummary

	for _, dep := range deps {
		switch dep.Usage {
		case model.DependencyUsageUsed:
			summary.Used++
		case model.DependencyUsageMaybe:
			summary.Maybe++
		case model.DependencyUsageUnused:
			summary.Unused++
		default:
			summary.Unknown++
		}
	}

	return summary
}

func topEdgeFiles(deps []model.DependencyEdge, from bool) []EdgeFileCount {
	counts := make(map[string]int)

	for _, dep := range deps {
		file := dep.ToFile
		if from {
			file = dep.FromFile
		}

		if file == "" {
			continue
		}

		counts[file]++
	}

	result := make([]EdgeFileCount, 0, len(counts))
	for file, count := range counts {
		result = append(result, EdgeFileCount{
			File:  file,
			Count: count,
		})
	}

	sort.Slice(result, func(i, j int) bool {
		if result[i].Count == result[j].Count {
			return result[i].File < result[j].File
		}

		return result[i].Count > result[j].Count
	})

	return result
}

func sortEdgeDependencies(deps []model.DependencyEdge) {
	sort.Slice(deps, func(i, j int) bool {
		if deps[i].FromFile == deps[j].FromFile {
			return edgeTarget(deps[i]) < edgeTarget(deps[j])
		}

		return deps[i].FromFile < deps[j].FromFile
	})
}

func findEdgeFindings(findings []model.Finding, fromLayer string, toLayer string) []model.Finding {
	needle := fromLayer + " -> " + toLayer
	idNeedle := "." + fromLayer + "." + toLayer

	result := make([]model.Finding, 0)
	for _, finding := range findings {
		if strings.Contains(finding.ID, idNeedle) {
			result = append(result, finding)
			continue
		}

		if findingEvidenceMentionsEdge(finding, needle) {
			result = append(result, finding)
		}
	}

	sort.SliceStable(result, func(i, j int) bool {
		return result[i].ID < result[j].ID
	})

	return result
}

func findingEvidenceMentionsEdge(finding model.Finding, needle string) bool {
	for _, evidence := range finding.Evidence {
		if strings.Contains(evidence.Message, needle) {
			return true
		}
	}

	return false
}

func edgeTarget(dep model.DependencyEdge) string {
	if dep.ToFile != "" {
		return dep.ToFile
	}

	return dep.Target
}
