package review

import (
	"fmt"

	depdiff "github.com/orurh/patchcourt/internal/diff/dep"
)

func worseLayerEdgeChanges(changes []depdiff.LayerEdgeChange, policyIndex policyViolationEdgeIndex) []ReviewImpactItem {
	items := make([]ReviewImpactItem, 0)

	for _, change := range changes {
		switch change.Kind {
		case depdiff.DependencyChangeKindAdded:
			if !isPolicyViolationLayerEdge(policyIndex, change.FromLayer, change.ToLayer) {
				continue
			}

			items = append(items, ReviewImpactItem{
				Kind:   "layer_edge_added",
				Title:  "Added forbidden layer dependency",
				Detail: fmt.Sprintf("%s -> %s (%d)", change.FromLayer, change.ToLayer, change.AfterCount),
			})

		case depdiff.DependencyChangeKindChanged:
			if change.AfterCount <= change.BeforeCount {
				continue
			}

			if !isPolicyViolationLayerEdge(policyIndex, change.FromLayer, change.ToLayer) {
				continue
			}

			items = append(items, ReviewImpactItem{
				Kind:   "layer_edge_increased",
				Title:  "Increased forbidden layer dependency",
				Detail: fmt.Sprintf("%s -> %s (%d -> %d)", change.FromLayer, change.ToLayer, change.BeforeCount, change.AfterCount),
			})
		}
	}

	return items
}

func betterLayerEdgeChanges(changes []depdiff.LayerEdgeChange, beforePolicyIndex policyViolationEdgeIndex) []ReviewImpactItem {
	items := make([]ReviewImpactItem, 0)

	for _, change := range changes {
		switch change.Kind {
		case depdiff.DependencyChangeKindRemoved:
			if !isPolicyViolationLayerEdge(beforePolicyIndex, change.FromLayer, change.ToLayer) {
				continue
			}

			items = append(items, ReviewImpactItem{
				Kind:   "layer_edge_removed",
				Title:  "Removed forbidden layer dependency",
				Detail: fmt.Sprintf("%s -> %s (%d)", change.FromLayer, change.ToLayer, change.BeforeCount),
			})

		case depdiff.DependencyChangeKindChanged:
			if change.AfterCount >= change.BeforeCount {
				continue
			}

			if !isPolicyViolationLayerEdge(beforePolicyIndex, change.FromLayer, change.ToLayer) {
				continue
			}

			items = append(items, ReviewImpactItem{
				Kind:   "layer_edge_decreased",
				Title:  "Reduced forbidden layer dependency",
				Detail: fmt.Sprintf("%s -> %s (%d -> %d)", change.FromLayer, change.ToLayer, change.BeforeCount, change.AfterCount),
			})
		}
	}

	return items
}

func needsReviewLayerEdgeChanges(
	changes []depdiff.LayerEdgeChange,
	afterPolicyIndex policyViolationEdgeIndex,
	beforePolicyIndex policyViolationEdgeIndex,
) []ReviewImpactItem {
	items := make([]ReviewImpactItem, 0)

	for _, change := range changes {
		switch change.Kind {
		case depdiff.DependencyChangeKindAdded:
			if isPolicyViolationLayerEdge(afterPolicyIndex, change.FromLayer, change.ToLayer) {
				continue
			}

			items = append(items, ReviewImpactItem{
				Kind:       "layer_edge_added_needs_review",
				Title:      "Layer dependency added; verify architecture intent",
				Detail:     fmt.Sprintf("%s -> %s (%d)", change.FromLayer, change.ToLayer, change.AfterCount),
				Suggestion: "This layer dependency is not proven forbidden by policy, but it changes the architecture graph and should be reviewed in context.",
			})

		case depdiff.DependencyChangeKindRemoved:
			if isPolicyViolationLayerEdge(beforePolicyIndex, change.FromLayer, change.ToLayer) {
				continue
			}

			items = append(items, ReviewImpactItem{
				Kind:       "layer_edge_removed_needs_review",
				Title:      "Layer dependency removed; verify whether this is a real improvement",
				Detail:     fmt.Sprintf("%s -> %s (%d)", change.FromLayer, change.ToLayer, change.BeforeCount),
				Suggestion: "A layer dependency disappeared, but PatchCourt cannot prove this is better unless it removed a known policy violation or known debt.",
			})

		case depdiff.DependencyChangeKindChanged:
			if change.AfterCount > change.BeforeCount {
				if isPolicyViolationLayerEdge(afterPolicyIndex, change.FromLayer, change.ToLayer) {
					continue
				}

				items = append(items, ReviewImpactItem{
					Kind:       "layer_edge_increased_needs_review",
					Title:      "Layer dependency count increased; verify architecture intent",
					Detail:     fmt.Sprintf("%s -> %s (%d -> %d)", change.FromLayer, change.ToLayer, change.BeforeCount, change.AfterCount),
					Suggestion: "The dependency count increased, but PatchCourt cannot prove this is a regression without policy or known-debt evidence.",
				})
				continue
			}

			if change.AfterCount < change.BeforeCount {
				if isPolicyViolationLayerEdge(beforePolicyIndex, change.FromLayer, change.ToLayer) {
					continue
				}

				items = append(items, ReviewImpactItem{
					Kind:       "layer_edge_decreased_needs_review",
					Title:      "Layer dependency count decreased; verify whether this is a real improvement",
					Detail:     fmt.Sprintf("%s -> %s (%d -> %d)", change.FromLayer, change.ToLayer, change.BeforeCount, change.AfterCount),
					Suggestion: "The dependency count decreased, but PatchCourt cannot prove this is better unless it reduced a known policy violation or known debt.",
				})
			}
		}
	}

	return items
}
