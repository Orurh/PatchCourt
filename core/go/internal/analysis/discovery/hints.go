package discovery

import (
	"fmt"
	"sort"

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

		evidence := make([]model.Evidence, 0, maxEvidencePerHint)
		evidence = append(evidence, dependencyEvidence(leftToRight, edges[leftToRight])...)
		evidence = append(evidence, dependencyEvidence(rightToLeft, edges[rightToLeft])...)

		findings = append(findings, model.Finding{
			ID:         fmt.Sprintf("discovery.bidirectional.%s.%s", left, right),
			Kind:       model.FindingKindDiscoveryHint,
			Severity:   model.SeverityMedium,
			Title:      "Bidirectional layer dependency",
			Confidence: model.ConfidenceMedium,
			Risk: fmt.Sprintf(
				"Layers %q and %q depend on each other. This may indicate a cycle, misplaced types, or missing dependency inversion.",
				left,
				right,
			),
			Suggestion: "Review whether one side should depend on an interface, shared model, or lower-level package instead.",
			Evidence:   limitEvidence(evidence, maxEvidencePerHint),
		})
	}

	return findings
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

		evidence = append(evidence, model.Evidence{
			File: dep.FromFile,
			Message: fmt.Sprintf(
				"includes %s, creating discovered layer dependency %s -> %s",
				target,
				pair.from,
				pair.to,
			),
		})
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
