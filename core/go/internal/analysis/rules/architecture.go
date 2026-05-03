package rules

import (
	"fmt"
	"strings"

	"github.com/orurh/patchcourt/internal/config"
	"github.com/orurh/patchcourt/internal/model"
	"github.com/orurh/patchcourt/internal/platform/pathmatch"
)

type ArchitectureRule struct{}

func (ArchitectureRule) Apply(project *model.ProjectModel, cfg *config.Config) []model.Finding {
	if cfg == nil || len(cfg.Layers) == 0 {
		return nil
	}

	assignLayers(project, cfg)
	enrichDependencyLayers(project)
	return checkLayerDependencies(project, cfg)
}

func ApplyArchitectureRules(project *model.ProjectModel, cfg *config.Config) {
	project.Findings = append(project.Findings, ArchitectureRule{}.Apply(project, cfg)...)
}

func assignLayers(project *model.ProjectModel, cfg *config.Config) {
	for i := range project.Files {
		layer := detectLayer(project.Files[i].Path, cfg)
		project.Files[i].Layer = layer
		if layer != "" {
			project.Files[i].LayerSource = model.LayerAssignmentSourceConfig
		}
	}
}

func enrichDependencyLayers(project *model.ProjectModel) {
	fileLayers := make(map[string]string, len(project.Files))
	for _, file := range project.Files {
		fileLayers[file.Path] = file.Layer
	}

	for i := range project.Dependencies {
		dep := &project.Dependencies[i]
		dep.FromLayer = fileLayers[dep.FromFile]

		if dep.Resolved {
			dep.ToLayer = fileLayers[dep.ToFile]
		}
	}
}

func checkLayerDependencies(project *model.ProjectModel, cfg *config.Config) []model.Finding {
	findings := make([]model.Finding, 0)

	ignoredFromFiles := make(map[string]bool, len(project.Files))
	for _, file := range project.Files {
		switch file.Role {
		case model.FileRoleTest, model.FileRoleGenerated, model.FileRoleExternal:
			ignoredFromFiles[file.Path] = true
		}
	}

	for _, dep := range project.Dependencies {
		if dep.External || !dep.Resolved {
			continue
		}

		if ignoredFromFiles[dep.FromFile] {
			continue
		}

		if dep.FromLayer == "" || dep.ToLayer == "" {
			continue
		}

		if dep.FromLayer == dep.ToLayer {
			continue
		}

		if isAllowedDependency(dep.FromLayer, dep.ToLayer, cfg) {
			continue
		}

		message := fmt.Sprintf(
			"includes %s, creating include dependency %s -> %s",
			dep.Target,
			dep.FromLayer,
			dep.ToLayer,
		)

		findings = append(findings, model.Finding{
			ID:         fmt.Sprintf("architecture.%s.%s", dep.FromLayer, dep.ToLayer),
			Kind:       model.FindingKindPolicyViolation,
			Severity:   model.SeverityHigh,
			Title:      "Include-level architecture boundary violation",
			Confidence: model.ConfidenceHigh,
			Risk: fmt.Sprintf(
				"Layer %q includes a header from layer %q, which is not allowed by .patchcourt.yaml. For C++, this is a compile-time include dependency; actual symbol usage is not verified yet.",
				dep.FromLayer,
				dep.ToLayer,
			),
			Suggestion: "Move the dependency behind an allowed interface, remove the include if it is unused, replace it with a forward declaration if possible, or update the architecture rules if this dependency is intentional.",
			Evidence: []model.Evidence{
				model.DependencyEvidence(dep, message),
			},
		})
	}

	return findings
}

func detectLayer(filePath string, cfg *config.Config) string {
	bestLayer := ""
	bestScore := -1

	for layerName, layer := range cfg.Layers {
		if pathmatch.MatchAny(layer.ExcludePaths, filePath) {
			continue
		}

		for _, pattern := range layer.Paths {
			if !pathmatch.Match(pattern, filePath) {
				continue
			}

			score := layerPatternScore(pattern)
			if score > bestScore {
				bestLayer = layerName
				bestScore = score
			}
		}
	}

	return bestLayer
}

func layerPatternScore(pattern string) int {
	normalized := pathmatch.Normalize(pattern)
	score := len(normalized)

	if !strings.Contains(normalized, "*") {
		score += 10000
	}

	return score
}

func isAllowedDependency(fromLayer string, toLayer string, cfg *config.Config) bool {
	layer, ok := cfg.Layers[fromLayer]
	if !ok {
		return false
	}

	for _, allowed := range layer.MayDependOn {
		if allowed == toLayer {
			return true
		}
	}

	return false
}
