package app

import (
	"strings"

	"github.com/orurh/patchcourt/internal/diff/dep"
	"github.com/orurh/patchcourt/internal/model"
)

type policyViolationEdgeIndex map[string]struct{}

func buildPolicyViolationEdgeIndex(project *model.ProjectModel) policyViolationEdgeIndex {
	index := make(policyViolationEdgeIndex)

	if project == nil {
		return index
	}

	for _, finding := range project.Findings {
		if finding.Kind != model.FindingKindPolicyViolation {
			continue
		}

		for _, evidence := range finding.Evidence {
			if evidence.FromLayer == "" || evidence.ToLayer == "" {
				continue
			}

			index[layerEdgeKey(evidence.FromLayer, evidence.ToLayer)] = struct{}{}
		}

		from, to, ok := architectureFindingLayers(finding.ID)
		if ok {
			index[layerEdgeKey(from, to)] = struct{}{}
		}
	}

	return index
}

func architectureFindingLayers(id string) (string, string, bool) {
	const prefix = "architecture."
	if !strings.HasPrefix(id, prefix) {
		return "", "", false
	}

	rest := strings.TrimPrefix(id, prefix)
	parts := strings.Split(rest, ".")
	if len(parts) != 2 {
		return "", "", false
	}

	if parts[0] == "" || parts[1] == "" {
		return "", "", false
	}

	return parts[0], parts[1], true
}

func layerEdgeKey(fromLayer string, toLayer string) string {
	return fromLayer + "->" + toLayer
}

func isPolicyViolationLayerEdge(index policyViolationEdgeIndex, fromLayer string, toLayer string) bool {
	if fromLayer == "" || toLayer == "" {
		return false
	}

	_, ok := index[layerEdgeKey(fromLayer, toLayer)]
	return ok
}

func isPolicyViolationDependency(index policyViolationEdgeIndex, dep model.DependencyEdge) bool {
	return isPolicyViolationLayerEdge(index, dep.FromLayer, dep.ToLayer)
}

func policyRelevantRiskDependencyChanges(changes []depdiff.DependencyChange, index policyViolationEdgeIndex) []depdiff.DependencyChange {
	result := make([]depdiff.DependencyChange, 0, len(changes))

	for _, change := range changes {
		if change.Kind == depdiff.DependencyChangeKindAdded && change.After != nil {
			if !isPolicyViolationDependency(index, *change.After) {
				continue
			}
		}

		result = append(result, change)
	}

	return result
}

func policyRelevantRiskLayerEdgeChanges(changes []depdiff.LayerEdgeChange, index policyViolationEdgeIndex) []depdiff.LayerEdgeChange {
	result := make([]depdiff.LayerEdgeChange, 0, len(changes))

	for _, change := range changes {
		switch change.Kind {
		case depdiff.DependencyChangeKindAdded:
			if !isPolicyViolationLayerEdge(index, change.FromLayer, change.ToLayer) {
				continue
			}
		case depdiff.DependencyChangeKindChanged:
			if change.AfterCount > change.BeforeCount && !isPolicyViolationLayerEdge(index, change.FromLayer, change.ToLayer) {
				continue
			}
		}

		result = append(result, change)
	}

	return result
}
