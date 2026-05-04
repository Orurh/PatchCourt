package depdiff

import (
	"sort"
	"strings"

	analysisproject "github.com/orurh/patchcourt/internal/analyzer/project"
	"github.com/orurh/patchcourt/internal/model"
)

type DependencyChangeKind string

const (
	DependencyChangeKindAdded   DependencyChangeKind = "added"
	DependencyChangeKindRemoved DependencyChangeKind = "removed"
	DependencyChangeKindChanged DependencyChangeKind = "changed"
)

type DependencyChange struct {
	Kind   DependencyChangeKind  `json:"kind"`
	Key    string                `json:"key"`
	Before *model.DependencyEdge `json:"before,omitempty"`
	After  *model.DependencyEdge `json:"after,omitempty"`
}

type LayerEdgeChange struct {
	Kind        DependencyChangeKind `json:"kind"`
	FromLayer   string               `json:"from_layer"`
	ToLayer     string               `json:"to_layer"`
	BeforeCount int                  `json:"before_count,omitempty"`
	AfterCount  int                  `json:"after_count,omitempty"`
}

func DiffDependencies(before []model.DependencyEdge, after []model.DependencyEdge) []DependencyChange {
	beforeIndex := indexDependencies(before)
	afterIndex := indexDependencies(after)

	keys := mergedSortedKeys(beforeIndex, afterIndex)
	changes := make([]DependencyChange, 0)

	for _, key := range keys {
		beforeDep, hadBefore := beforeIndex[key]
		afterDep, hasAfter := afterIndex[key]

		switch {
		case !hadBefore && hasAfter:
			afterCopy := afterDep
			changes = append(changes, DependencyChange{
				Kind:  DependencyChangeKindAdded,
				Key:   key,
				After: &afterCopy,
			})

		case hadBefore && !hasAfter:
			beforeCopy := beforeDep
			changes = append(changes, DependencyChange{
				Kind:   DependencyChangeKindRemoved,
				Key:    key,
				Before: &beforeCopy,
			})
		}
	}

	return changes
}

func DiffLayerEdges(before []model.DependencyEdge, after []model.DependencyEdge) []LayerEdgeChange {
	beforeCounts := layerEdgeCounts(before)
	afterCounts := layerEdgeCounts(after)

	keys := mergedSortedIntKeys(beforeCounts, afterCounts)
	changes := make([]LayerEdgeChange, 0)

	for _, key := range keys {
		beforeCount, hadBefore := beforeCounts[key]
		afterCount, hasAfter := afterCounts[key]

		from, to := splitLayerEdgeKey(key)

		switch {
		case !hadBefore && hasAfter:
			changes = append(changes, LayerEdgeChange{
				Kind:       DependencyChangeKindAdded,
				FromLayer:  from,
				ToLayer:    to,
				AfterCount: afterCount,
			})

		case hadBefore && !hasAfter:
			changes = append(changes, LayerEdgeChange{
				Kind:        DependencyChangeKindRemoved,
				FromLayer:   from,
				ToLayer:     to,
				BeforeCount: beforeCount,
			})

		case hadBefore && hasAfter && beforeCount != afterCount:
			changes = append(changes, LayerEdgeChange{
				Kind:        DependencyChangeKindChanged,
				FromLayer:   from,
				ToLayer:     to,
				BeforeCount: beforeCount,
				AfterCount:  afterCount,
			})
		}
	}

	return changes
}

func indexDependencies(deps []model.DependencyEdge) map[string]model.DependencyEdge {
	index := make(map[string]model.DependencyEdge)

	for _, dep := range deps {
		key := DependencyKey(dep)
		if key == "" {
			continue
		}

		index[key] = dep
	}

	return index
}

func DependencyKey(dep model.DependencyEdge) string {
	if dep.FromFile == "" || dep.Kind == "" {
		return ""
	}

	target := dependencyTargetKey(dep)
	if target == "" {
		return ""
	}

	return strings.Join([]string{
		string(dep.Kind),
		dep.FromFile,
		target,
	}, "|")
}

func dependencyTargetKey(dep model.DependencyEdge) string {
	if dep.ToFile != "" {
		return dep.ToFile
	}

	return dep.Target
}

func layerEdgeCounts(deps []model.DependencyEdge) map[string]int {
	counts := make(map[string]int)

	for _, dep := range deps {
		if dep.FromFile != "" && analysisproject.IsIgnoredAnalysisPath(dep.FromFile) {
			continue
		}

		if dep.External || !dep.Resolved {
			continue
		}

		if dep.FromLayer == "" || dep.ToLayer == "" {
			continue
		}

		if dep.FromLayer == dep.ToLayer {
			continue
		}

		key := layerEdgeKey(dep.FromLayer, dep.ToLayer)
		counts[key]++
	}

	return counts
}

func layerEdgeKey(from string, to string) string {
	return from + "->" + to
}

func splitLayerEdgeKey(key string) (string, string) {
	parts := strings.SplitN(key, "->", 2)
	if len(parts) != 2 {
		return key, ""
	}

	return parts[0], parts[1]
}

func mergedSortedKeys(left map[string]model.DependencyEdge, right map[string]model.DependencyEdge) []string {
	seen := make(map[string]struct{}, len(left)+len(right))

	for key := range left {
		seen[key] = struct{}{}
	}

	for key := range right {
		seen[key] = struct{}{}
	}

	keys := make([]string, 0, len(seen))
	for key := range seen {
		keys = append(keys, key)
	}

	sort.Strings(keys)
	return keys
}

func mergedSortedIntKeys(left map[string]int, right map[string]int) []string {
	seen := make(map[string]struct{}, len(left)+len(right))

	for key := range left {
		seen[key] = struct{}{}
	}

	for key := range right {
		seen[key] = struct{}{}
	}

	keys := make([]string, 0, len(seen))
	for key := range seen {
		keys = append(keys, key)
	}

	sort.Strings(keys)
	return keys
}
