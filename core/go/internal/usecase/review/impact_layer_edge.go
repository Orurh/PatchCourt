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
func betterLayerEdgeChanges(changes []depdiff.LayerEdgeChange) []ReviewImpactItem {
	items := make([]ReviewImpactItem, 0)

	for _, change := range changes {
		switch change.Kind {
		case depdiff.DependencyChangeKindRemoved:
			items = append(items, ReviewImpactItem{
				Kind:   "layer_edge_removed",
				Title:  "Removed layer dependency",
				Detail: fmt.Sprintf("%s -> %s (%d)", change.FromLayer, change.ToLayer, change.BeforeCount),
			})

		case depdiff.DependencyChangeKindChanged:
			if change.AfterCount < change.BeforeCount {
				items = append(items, ReviewImpactItem{
					Kind:   "layer_edge_decreased",
					Title:  "Reduced layer dependency",
					Detail: fmt.Sprintf("%s -> %s (%d -> %d)", change.FromLayer, change.ToLayer, change.BeforeCount, change.AfterCount),
				})
			}
		}
	}

	return items
}
