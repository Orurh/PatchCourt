package review

import (
	"fmt"

	depdiff "github.com/orurh/patchcourt/internal/diff/dep"
	"github.com/orurh/patchcourt/internal/model"
)

func worseDependencyChanges(changes []depdiff.DependencyChange, policyIndex policyViolationEdgeIndex) []ReviewImpactItem {
	items := make([]ReviewImpactItem, 0)

	for _, change := range changes {
		if change.Kind != depdiff.DependencyChangeKindAdded || change.After == nil {
			continue
		}

		dep := change.After
		if !isReviewRelevantDependency(*dep) {
			continue
		}

		if !isPolicyViolationDependency(policyIndex, *dep) {
			continue
		}

		items = append(items, ReviewImpactItem{
			Kind:   "dependency_added",
			Title:  "Added forbidden dependency",
			Detail: dependencyImpactDetail(*dep),
			ID:     change.Key,
		})
	}

	return items
}
func betterDependencyChanges(changes []depdiff.DependencyChange) []ReviewImpactItem {
	items := make([]ReviewImpactItem, 0)

	for _, change := range changes {
		if change.Kind != depdiff.DependencyChangeKindRemoved || change.Before == nil {
			continue
		}

		dep := change.Before
		if !isReviewRelevantDependency(*dep) {
			continue
		}

		items = append(items, ReviewImpactItem{
			Kind:   "dependency_removed",
			Title:  "Removed dependency",
			Detail: dependencyImpactDetail(*dep),
			ID:     change.Key,
		})
	}

	return items
}
func isReviewRelevantDependency(dep model.DependencyEdge) bool {
	if dep.External {
		return false
	}

	if dep.FromLayer == "" || dep.ToLayer == "" {
		return false
	}

	if dep.FromLayer == dep.ToLayer {
		return false
	}

	return true
}
func dependencyImpactDetail(dep model.DependencyEdge) string {
	target := dep.ToFile
	if target == "" {
		target = dep.Target
	}

	if dep.FromLayer != "" || dep.ToLayer != "" {
		return fmt.Sprintf("%s -> %s (%s -> %s)", dep.FromFile, target, dep.FromLayer, dep.ToLayer)
	}

	return fmt.Sprintf("%s -> %s", dep.FromFile, target)
}
