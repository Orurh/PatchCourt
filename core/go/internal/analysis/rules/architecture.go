package rules

import (
	"fmt"
	"sort"
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
	ignoredFromFiles := make(map[string]bool, len(project.Files))
	for _, file := range project.Files {
		switch file.Role {
		case model.FileRoleTest, model.FileRoleGenerated, model.FileRoleExternal:
			ignoredFromFiles[file.Path] = true
		}
	}

	builders := make(map[string]*architectureFindingBuilder)

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

		id := fmt.Sprintf("architecture.%s.%s", dep.FromLayer, dep.ToLayer)

		builder := builders[id]
		if builder == nil {
			builder = &architectureFindingBuilder{
				id:        id,
				fromLayer: dep.FromLayer,
				toLayer:   dep.ToLayer,
			}
			builders[id] = builder
		}

		builder.addDependency(dep)
	}

	ids := make([]string, 0, len(builders))
	for id := range builders {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	findings := make([]model.Finding, 0, len(ids))
	for _, id := range ids {
		findings = append(findings, builders[id].finding())
	}

	return findings
}

type architectureFindingBuilder struct {
	id        string
	fromLayer string
	toLayer   string
	kinds     map[model.DependencyKind]struct{}
	evidence  []model.Evidence
}

func (b *architectureFindingBuilder) addDependency(dep model.DependencyEdge) {
	if b.kinds == nil {
		b.kinds = make(map[model.DependencyKind]struct{})
	}

	b.kinds[dep.Kind] = struct{}{}
	b.evidence = append(b.evidence, model.DependencyEvidence(dep, architectureEvidenceMessage(dep)))
}

func (b *architectureFindingBuilder) finding() model.Finding {
	return model.Finding{
		ID:         b.id,
		Kind:       model.FindingKindPolicyViolation,
		Severity:   model.SeverityHigh,
		Title:      architectureFindingTitle(b.kinds),
		Confidence: model.ConfidenceHigh,
		Risk:       architectureRiskText(b.fromLayer, b.toLayer, b.kinds),
		Suggestion: architectureSuggestionText(b.kinds),
		Evidence:   b.evidence,
	}
}

func architectureFindingTitle(kinds map[model.DependencyKind]struct{}) string {
	if hasOnlyDependencyKind(kinds, model.DependencyKindImport) {
		return "Import-level architecture boundary violation"
	}

	if hasOnlyDependencyKind(kinds, model.DependencyKindInclude) {
		return "Include-level architecture boundary violation"
	}

	return "Architecture boundary violation"
}

func architectureRiskText(fromLayer string, toLayer string, kinds map[model.DependencyKind]struct{}) string {
	if hasOnlyDependencyKind(kinds, model.DependencyKindImport) {
		return fmt.Sprintf(
			"Layer %q imports code from layer %q, which is not allowed by .patchcourt.yaml.",
			fromLayer,
			toLayer,
		)
	}

	if hasOnlyDependencyKind(kinds, model.DependencyKindInclude) {
		return fmt.Sprintf(
			"Layer %q includes a header from layer %q, which is not allowed by .patchcourt.yaml. For C++, this is a compile-time include dependency; actual symbol usage is not verified yet.",
			fromLayer,
			toLayer,
		)
	}

	return fmt.Sprintf(
		"Layer %q depends on layer %q, which is not allowed by .patchcourt.yaml.",
		fromLayer,
		toLayer,
	)
}

func architectureSuggestionText(kinds map[model.DependencyKind]struct{}) string {
	if hasOnlyDependencyKind(kinds, model.DependencyKindImport) {
		return "Move the dependency behind an allowed package boundary, introduce a lower-level shared package, or update the architecture rules if this dependency is intentional."
	}

	if hasOnlyDependencyKind(kinds, model.DependencyKindInclude) {
		return "Move the dependency behind an allowed interface, remove the include if it is unused, replace it with a forward declaration if possible, or update the architecture rules if this dependency is intentional."
	}

	return "Move the dependency behind an allowed boundary, introduce a lower-level shared package/module, or update the architecture rules if this dependency is intentional."
}

func architectureEvidenceMessage(dep model.DependencyEdge) string {
	target := dep.ToFile
	if target == "" {
		target = dep.Target
	}

	switch dep.Kind {
	case model.DependencyKindImport:
		return fmt.Sprintf(
			"imports %s, creating forbidden layer dependency %s -> %s",
			target,
			dep.FromLayer,
			dep.ToLayer,
		)
	case model.DependencyKindInclude:
		return fmt.Sprintf(
			"includes %s, creating forbidden layer dependency %s -> %s",
			target,
			dep.FromLayer,
			dep.ToLayer,
		)
	default:
		return fmt.Sprintf(
			"depends on %s, creating forbidden layer dependency %s -> %s",
			target,
			dep.FromLayer,
			dep.ToLayer,
		)
	}
}

func hasOnlyDependencyKind(kinds map[model.DependencyKind]struct{}, kind model.DependencyKind) bool {
	if len(kinds) != 1 {
		return false
	}

	_, ok := kinds[kind]
	return ok
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
