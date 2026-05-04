package usecase

import (
	"sort"
	"strings"

	analysisproject "github.com/orurh/patchcourt/internal/analyzer/project"
	"github.com/orurh/patchcourt/internal/model"
	"github.com/orurh/patchcourt/internal/reportmodel"
)

const defaultEdgeDependencyLimit = 50

type EdgeReportOptions struct {
	Root      string
	Source    string
	FromLayer string
	ToLayer   string
	Limit     int
}

func BuildEdgeReport(project *model.ProjectModel, opts EdgeReportOptions) *EdgeResult {
	limit := opts.Limit
	if limit <= 0 {
		limit = defaultEdgeDependencyLimit
	}

	deps := filterEdgeDependencies(project, opts.FromLayer, opts.ToLayer)
	sortEdgeDependencies(deps)

	result := &EdgeResult{
		SchemaVersion: reportmodel.EdgeResultSchemaVersion,
		Root:          opts.Root,
		Source:        opts.Source,
		FromLayer:     opts.FromLayer,
		ToLayer:       opts.ToLayer,
		Count:         len(deps),
		Usage:         summarizeEdgeUsage(deps),
		Findings:      findEdgeFindings(edgeProjectFindings(project), opts.FromLayer, opts.ToLayer),
		TopFromFiles:  topEdgeFiles(deps, true),
		TopToFiles:    topEdgeFiles(deps, false),
	}

	if project != nil && result.Root == "" {
		result.Root = project.Root
	}

	if len(deps) > limit {
		result.Dependencies = deps[:limit]
		result.TruncatedDeps = len(deps) - limit
	} else {
		result.Dependencies = deps
	}

	return result
}

func edgeProjectFindings(project *model.ProjectModel) []model.Finding {
	if project == nil {
		return nil
	}

	return project.Findings
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
	return analysisproject.IgnoredAnalysisFileSet(files)
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
		file := dependencyTarget(dep)
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
			return dependencyTarget(deps[i]) < dependencyTarget(deps[j])
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

func dependencyTarget(dep model.DependencyEdge) string {
	if dep.ToFile != "" {
		return dep.ToFile
	}

	return dep.Target
}
