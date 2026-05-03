package discovery

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/orurh/patchcourt/internal/model"
)

const maxEvidencePerHint = 5

type layerPair struct {
	from string
	to   string
}

// AnalyzeHints returns heuristic, non-policy findings derived from discovered
// layer dependencies.
//
// These findings are review signals, not hard architecture violations.
// They must remain evidence-based and should not depend on .patchcourt.yaml.
func AnalyzeHints(project *model.ProjectModel) []model.Finding {
	if project == nil {
		return nil
	}

	ignoredFiles := ignoredFromFiles(project)
	edges := collectLayerEdges(project, ignoredFiles)
	findings := make([]model.Finding, 0, len(edges)/2)

	findings = append(findings, bidirectionalLayerHints(edges)...)
	findings = append(findings, domainDependencyHints(edges)...)
	findings = append(findings, reverseControllerServerHints(edges)...)
	findings = append(findings, sharedDomainAwareHints(edges)...)
	findings = append(findings, misplacedSharedCandidateHints(edges)...)
	findings = append(findings, unusedIncludeHints(project)...)

	sort.SliceStable(findings, func(i, j int) bool {
		return findings[i].ID < findings[j].ID
	})

	return findings
}

func collectLayerEdges(project *model.ProjectModel, ignoredFiles map[string]bool) map[layerPair][]model.DependencyEdge {
	edges := make(map[layerPair][]model.DependencyEdge)

	for _, dep := range project.Dependencies {
		if dep.External || !dep.Resolved {
			continue
		}

		if ignoredFiles[dep.FromFile] {
			continue
		}

		if dep.FromLayer == "" || dep.ToLayer == "" {
			continue
		}

		if dep.FromLayer == dep.ToLayer {
			continue
		}

		pair := layerPair{
			from: dep.FromLayer,
			to:   dep.ToLayer,
		}

		edges[pair] = append(edges[pair], dep)
	}

	return edges
}

func bidirectionalLayerHints(edges map[layerPair][]model.DependencyEdge) []model.Finding {
	findings := make([]model.Finding, 0)
	seen := make(map[string]struct{})

	for pair := range edges {
		reverse := layerPair{
			from: pair.to,
			to:   pair.from,
		}

		if _, ok := edges[reverse]; !ok {
			continue
		}

		left, right := canonicalLayerPair(pair.from, pair.to)
		key := canonicalPairKey(left, right)
		if _, alreadySeen := seen[key]; alreadySeen {
			continue
		}
		seen[key] = struct{}{}

		leftToRight := layerPair{from: left, to: right}
		rightToLeft := layerPair{from: right, to: left}

		severity := model.SeverityMedium
		title := "Bidirectional layer dependency"
		risk := fmt.Sprintf(
			"Layers %q and %q depend on each other. This may indicate a cycle, misplaced types, or missing dependency inversion.",
			left,
			right,
		)
		suggestion := "Review whether one side should depend on an interface, shared model, or lower-level package instead."

		evidence := make([]model.Evidence, 0, maxEvidencePerHint)

		if expectedSide, suspiciousSide, ok := compositionRootBidirectionalContext(
			leftToRight,
			edges[leftToRight],
			rightToLeft,
			edges[rightToLeft],
		); ok {
			severity = model.SeverityLow
			title = "Bidirectional layer dependency with composition-root side"
			risk = fmt.Sprintf(
				"Layer %q depends on %q, while the reverse dependency appears to come only from composition-root files. The composition-root side is often expected; inspect %s -> %s as the likely architectural concern.",
				suspiciousSide.from,
				suspiciousSide.to,
				suspiciousSide.from,
				suspiciousSide.to,
			)
			suggestion = fmt.Sprintf(
				"Keep composition wiring in %q if intentional, but review whether %q should depend on an interface, shared model, or lower-level package instead of %q.",
				expectedSide.from,
				suspiciousSide.from,
				suspiciousSide.to,
			)

			evidence = append(evidence, dependencyEvidence(suspiciousSide, edges[suspiciousSide])...)
			evidence = append(evidence, dependencyEvidence(expectedSide, edges[expectedSide])...)
		} else {
			evidence = append(evidence, dependencyEvidence(leftToRight, edges[leftToRight])...)
			evidence = append(evidence, dependencyEvidence(rightToLeft, edges[rightToLeft])...)
		}

		findings = append(findings, model.Finding{
			ID:         fmt.Sprintf("discovery.bidirectional.%s.%s", left, right),
			Kind:       model.FindingKindDiscoveryHint,
			Severity:   severity,
			Title:      title,
			Confidence: model.ConfidenceMedium,
			Risk:       risk,
			Suggestion: suggestion,
			Evidence:   limitEvidence(evidence, maxEvidencePerHint),
		})
	}

	return findings
}

func compositionRootBidirectionalContext(
	leftToRight layerPair,
	leftToRightDeps []model.DependencyEdge,
	rightToLeft layerPair,
	rightToLeftDeps []model.DependencyEdge,
) (expectedSide layerPair, suspiciousSide layerPair, ok bool) {
	if isCompositionRootEdge(leftToRight, leftToRightDeps) {
		return leftToRight, rightToLeft, true
	}

	if isCompositionRootEdge(rightToLeft, rightToLeftDeps) {
		return rightToLeft, leftToRight, true
	}

	return layerPair{}, layerPair{}, false
}

func isCompositionRootEdge(pair layerPair, deps []model.DependencyEdge) bool {
	if pair.from != "application" && pair.from != "entrypoint" {
		return false
	}

	if len(deps) == 0 {
		return false
	}

	for _, dep := range deps {
		if !isCompositionRootFile(dep.FromFile) {
			return false
		}
	}

	return true
}

func isCompositionRootFile(filePath string) bool {
	normalized := strings.ToLower(strings.ReplaceAll(filePath, "\\", "/"))
	base := filepath.Base(normalized)

	switch base {
	case "main.c", "main.cc", "main.cpp", "main.cxx", "main.go":
		return true
	}

	if strings.Contains(normalized, "/bootstrapper.") {
		return true
	}

	if strings.HasPrefix(normalized, "cmd/") && base == "main.go" {
		return true
	}

	return false
}

func domainDependencyHints(edges map[layerPair][]model.DependencyEdge) []model.Finding {
	findings := make([]model.Finding, 0)

	for pair, deps := range edges {
		if pair.from != "domain" {
			continue
		}

		if pair.to == "shared" {
			continue
		}

		findings = append(findings, model.Finding{
			ID:         fmt.Sprintf("discovery.domain.depends_on.%s", pair.to),
			Kind:       model.FindingKindDiscoveryHint,
			Severity:   model.SeverityMedium,
			Title:      "Domain layer depends on an outer layer",
			Confidence: model.ConfidenceMedium,
			Risk: fmt.Sprintf(
				"Domain layer includes files from %q. Domain code is usually expected to stay independent from outer implementation or orchestration layers.",
				pair.to,
			),
			Suggestion: "Consider moving the referenced type into domain/shared, introducing a domain interface, or reversing the dependency.",
			Evidence:   dependencyEvidence(pair, deps),
		})
	}

	return findings
}

func reverseControllerServerHints(edges map[layerPair][]model.DependencyEdge) []model.Finding {
	pair := layerPair{from: "controllers", to: "server"}
	deps, ok := edges[pair]
	if !ok {
		return nil
	}

	return []model.Finding{
		{
			ID:         "discovery.controllers.depends_on.server",
			Kind:       model.FindingKindDiscoveryHint,
			Severity:   model.SeverityMedium,
			Title:      "Controller layer depends on server layer",
			Confidence: model.ConfidenceMedium,
			Risk:       "Controllers include server-layer files. Usually server/API adapters should call controllers, not the other way around.",
			Suggestion: "Move shared mapper/DTO code out of server, or keep server-specific mapping behind the server adapter.",
			Evidence:   dependencyEvidence(pair, deps),
		},
	}
}

func misplacedSharedCandidateHints(edges map[layerPair][]model.DependencyEdge) []model.Finding {
	findings := make([]model.Finding, 0)

	for pair, deps := range edges {
		if pair.to != "application" {
			continue
		}

		targetGroups := groupDepsByTargetFile(deps)
		for target, targetDeps := range targetGroups {
			if !isSharedCandidateTarget(target) {
				continue
			}

			findings = append(findings, model.Finding{
				ID:         fmt.Sprintf("discovery.shared_candidate.%s.%s", pair.to, sharedCandidateIDPart(target)),
				Kind:       model.FindingKindDiscoveryHint,
				Severity:   model.SeverityLow,
				Title:      "Application file looks like shared dependency candidate",
				Confidence: model.ConfidenceMedium,
				Risk: fmt.Sprintf(
					"Layer %q depends on %q only through %q. This file name looks shared/config-like, so the dependency may be caused by a misplaced constants/config header rather than a real dependency on the application layer.",
					pair.from,
					pair.to,
					target,
				),
				Suggestion: "Consider moving this file to a lower-level shared/config/domain layer, or override its layer assignment in .patchcourt.yaml if it is intentionally shared.",
				Evidence:   dependencyEvidence(pair, targetDeps),
			})
		}
	}

	return findings
}

func groupDepsByTargetFile(deps []model.DependencyEdge) map[string][]model.DependencyEdge {
	result := make(map[string][]model.DependencyEdge)

	for _, dep := range deps {
		target := dep.ToFile
		if target == "" {
			target = dep.Target
		}

		if target == "" {
			continue
		}

		result[target] = append(result[target], dep)
	}

	return result
}

func isSharedCandidateTarget(filePath string) bool {
	normalized := strings.ToLower(strings.ReplaceAll(filePath, "\\", "/"))
	base := filepath.Base(normalized)

	switch base {
	case "constants.h", "constants.hpp", "constants.hh", "constants.hxx",
		"config.h", "config.hpp", "types.h", "types.hpp",
		"common.h", "common.hpp":
		return true
	}

	return strings.Contains(normalized, "/constants/") ||
		strings.Contains(normalized, "/config/") ||
		strings.Contains(normalized, "/configs/") ||
		strings.Contains(normalized, "/common/") ||
		strings.Contains(normalized, "/shared/")
}

func sharedCandidateIDPart(filePath string) string {
	base := strings.ToLower(filepath.Base(strings.ReplaceAll(filePath, "\\", "/")))
	base = strings.TrimSuffix(base, filepath.Ext(base))

	if base == "" {
		return "file"
	}

	var b strings.Builder
	lastUnderscore := false

	for _, r := range base {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
			lastUnderscore = false
		case r >= '0' && r <= '9':
			b.WriteRune(r)
			lastUnderscore = false
		default:
			if !lastUnderscore {
				b.WriteRune('_')
				lastUnderscore = true
			}
		}
	}

	result := strings.Trim(b.String(), "_")
	if result == "" {
		return "file"
	}

	return result
}

func sharedDomainAwareHints(edges map[layerPair][]model.DependencyEdge) []model.Finding {
	pair := layerPair{from: "shared", to: "domain"}
	deps, ok := edges[pair]
	if !ok {
		return nil
	}

	return []model.Finding{
		{
			ID:         "discovery.shared.depends_on.domain",
			Kind:       model.FindingKindDiscoveryHint,
			Severity:   model.SeverityLow,
			Title:      "Shared layer depends on domain models",
			Confidence: model.ConfidenceMedium,
			Risk:       "Files classified as shared include domain models. This may mean the shared layer contains domain-aware serializers, mappers, or utilities rather than generic shared code.",
			Suggestion: "Consider splitting generic shared utilities from domain-aware adapters such as serialization or mapping code.",
			Evidence:   dependencyEvidence(pair, deps),
		},
	}
}

func dependencyEvidence(pair layerPair, deps []model.DependencyEdge) []model.Evidence {
	evidence := make([]model.Evidence, 0, minInt(len(deps), maxEvidencePerHint))

	for _, dep := range deps {
		if len(evidence) >= maxEvidencePerHint {
			break
		}

		target := dep.ToFile
		if target == "" {
			target = dep.Target
		}

		message := fmt.Sprintf(
			"includes %s, creating discovered layer dependency %s -> %s",
			target,
			pair.from,
			pair.to,
		)

		edgeDep := dep
		edgeDep.FromLayer = pair.from
		edgeDep.ToLayer = pair.to

		evidence = append(evidence, model.DependencyEvidence(edgeDep, message))
	}

	return evidence
}

func limitEvidence(evidence []model.Evidence, limit int) []model.Evidence {
	if len(evidence) <= limit {
		return evidence
	}

	return evidence[:limit]
}

func canonicalLayerPair(left string, right string) (string, string) {
	if left <= right {
		return left, right
	}

	return right, left
}

func canonicalPairKey(left string, right string) string {
	left, right = canonicalLayerPair(left, right)
	return left + "<->" + right
}

func minInt(left int, right int) int {
	if left < right {
		return left
	}

	return right
}
