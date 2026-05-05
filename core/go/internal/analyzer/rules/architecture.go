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
	kind := model.FindingKindPolicyViolation
	severity := model.SeverityHigh
	title := architectureFindingTitle(b.kinds)
	riskText := architectureRiskText(b.fromLayer, b.toLayer, b.kinds)
	suggestion := architectureSuggestionText(b.fromLayer, b.toLayer, b.kinds)

	if isCompositionRootLoggingReview(b.fromLayer, b.toLayer, b.kinds, b.evidence) {
		kind = model.FindingKindPolicyReview
		severity = model.SeverityMedium
		title = "Composition-root logging dependency review"
		riskText = fmt.Sprintf(
			"Layer %q imports logging infrastructure from layer %q. This is often acceptable in a CLI/composition root, but it is not allowed by the selected preset.",
			b.fromLayer,
			b.toLayer,
		)
		suggestion = "If this layer is intentionally wiring application dependencies, use a CLI/composition-root preset or keep this as an explicit architecture decision. Otherwise move logging setup behind an allowed boundary."
	}

	return model.Finding{
		ID:         b.id,
		Kind:       kind,
		Severity:   severity,
		Title:      title,
		Confidence: model.ConfidenceHigh,
		Risk:       riskText,
		Suggestion: suggestion,
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

func architectureSuggestionText(fromLayer string, toLayer string, kinds map[model.DependencyKind]struct{}) string {
	if suggestion := cleanArchitectureSuggestion(fromLayer, toLayer); suggestion != "" {
		return suggestion
	}

	if hasOnlyDependencyKind(kinds, model.DependencyKindImport) {
		return "Move the dependency behind an allowed package boundary, introduce a lower-level shared package, or update the architecture rules if this dependency is intentional."
	}

	if hasOnlyDependencyKind(kinds, model.DependencyKindInclude) {
		return "Move the dependency behind an allowed interface, remove the include if it is unused, replace it with a forward declaration if possible, or update the architecture rules if this dependency is intentional."
	}

	return "Move the dependency behind an allowed boundary, introduce a lower-level shared package/module, or update the architecture rules if this dependency is intentional."
}

func cleanArchitectureSuggestion(fromLayer string, toLayer string) string {
	from := normalizeLayerName(fromLayer)
	to := normalizeLayerName(toLayer)

	if isDomainLayer(from) && isOuterLayer(to) {
		return "Keep domain/model/contracts independent from outer implementation details. Move the referenced type into domain/shared, introduce a domain-owned abstraction, or reverse the dependency so infrastructure/adapters depend inward."
	}

	if isDeliveryLayer(from) && isConcreteImplementationLayer(to) {
		return "Keep delivery/API code from depending directly on concrete infrastructure or vendor implementations. Route the call through an application/usecase boundary and depend on a domain/application port instead."
	}

	if isApplicationLayer(from) && isConcreteImplementationLayer(to) {
		return "Keep application/usecase policy independent from concrete infrastructure or vendor implementations. Depend on a port/interface owned by the application/domain layer, and let infrastructure/adapters implement it."
	}

	return ""
}

func normalizeLayerName(layer string) string {
	return strings.ToLower(strings.TrimSpace(layer))
}

func isDomainLayer(layer string) bool {
	switch layer {
	case "domain", "model", "models", "contract", "contracts", "entities", "entity":
		return true
	default:
		return false
	}
}

func isApplicationLayer(layer string) bool {
	switch layer {
	case "application", "app", "usecase", "usecases", "service", "services":
		return true
	default:
		return false
	}
}

func isDeliveryLayer(layer string) bool {
	switch layer {
	case "api", "server", "web", "http", "grpc", "controllers", "controller", "delivery", "adapter", "adapters":
		return true
	default:
		return false
	}
}

func isConcreteImplementationLayer(layer string) bool {
	switch layer {
	case "infrastructure", "infra", "vendor", "vendors", "camera", "cameras", "database", "db", "persistence", "framework", "frameworks":
		return true
	default:
		return false
	}
}

func isOuterLayer(layer string) bool {
	return isDeliveryLayer(layer) || isApplicationLayer(layer) || isConcreteImplementationLayer(layer)
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

func isCompositionRootLoggingReview(fromLayer string, toLayer string, kinds map[model.DependencyKind]struct{}, evidence []model.Evidence) bool {
	if !hasOnlyDependencyKind(kinds, model.DependencyKindImport) {
		return false
	}

	if !isCompositionRootLayer(fromLayer) {
		return false
	}

	if !isPlatformLayer(toLayer) {
		return false
	}

	if len(evidence) == 0 {
		return false
	}

	for _, item := range evidence {
		target := item.ToFile
		if target == "" {
			target = item.Message
		}

		if !looksLikeLoggingTarget(target) {
			return false
		}
	}

	return true
}

func isCompositionRootLayer(layer string) bool {
	switch layer {
	case "cmd", "cli", "appmain", "main", "entrypoint":
		return true
	default:
		return false
	}
}

func isPlatformLayer(layer string) bool {
	switch layer {
	case "platform", "infra", "infrastructure":
		return true
	default:
		return false
	}
}

func looksLikeLoggingTarget(target string) bool {
	normalized := strings.ToLower(target)
	return strings.Contains(normalized, "/log") ||
		strings.Contains(normalized, "logx") ||
		strings.Contains(normalized, "logger") ||
		strings.Contains(normalized, "slog")
}
