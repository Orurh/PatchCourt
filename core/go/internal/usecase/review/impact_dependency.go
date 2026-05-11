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

func betterDependencyChanges(changes []depdiff.DependencyChange, beforePolicyIndex policyViolationEdgeIndex) []ReviewImpactItem {
	items := make([]ReviewImpactItem, 0)

	for _, change := range changes {
		if change.Kind != depdiff.DependencyChangeKindRemoved || change.Before == nil {
			continue
		}

		dep := change.Before
		if !isReviewRelevantDependency(*dep) {
			continue
		}

		if !isPolicyViolationDependency(beforePolicyIndex, *dep) {
			continue
		}

		items = append(items, ReviewImpactItem{
			Kind:   "dependency_removed",
			Title:  "Removed forbidden dependency",
			Detail: dependencyImpactDetail(*dep),
			ID:     change.Key,
		})
	}

	return items
}

func needsReviewDependencyChanges(
	changes []depdiff.DependencyChange,
	afterPolicyIndex policyViolationEdgeIndex,
	beforePolicyIndex policyViolationEdgeIndex,
) []ReviewImpactItem {
	items := make([]ReviewImpactItem, 0)

	for _, change := range changes {
		switch change.Kind {
		case depdiff.DependencyChangeKindAdded:
			if change.After == nil {
				continue
			}

			dep := change.After
			if !isReviewRelevantDependency(*dep) {
				continue
			}

			if isPolicyViolationDependency(afterPolicyIndex, *dep) {
				continue
			}

			items = append(items, ReviewImpactItem{
				Kind:       "dependency_added_needs_review",
				Title:      "Dependency added; verify architecture intent",
				Detail:     dependencyImpactDetail(*dep),
				ID:         change.Key,
				Suggestion: "This dependency is not proven forbidden by policy, but it changes the architecture graph and should be reviewed in context.",
			})

		case depdiff.DependencyChangeKindRemoved:
			if change.Before == nil {
				continue
			}

			dep := change.Before
			if !isReviewRelevantDependency(*dep) {
				continue
			}

			if isPolicyViolationDependency(beforePolicyIndex, *dep) {
				continue
			}

			items = append(items, ReviewImpactItem{
				Kind:       "dependency_removed_needs_review",
				Title:      "Dependency removed; verify whether this is a real improvement",
				Detail:     dependencyImpactDetail(*dep),
				ID:         change.Key,
				Suggestion: "A dependency disappeared, but PatchCourt cannot prove this is better unless it removed a known policy violation or known debt.",
			})
		}
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

	return fmt.Sprintf("%s -> %s (%s -> %s)", dep.FromFile, target, dep.FromLayer, dep.ToLayer)
}
