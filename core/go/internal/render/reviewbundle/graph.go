package reviewbundle

import (
	"github.com/orurh/patchcourt/internal/diff/dep"
	"github.com/orurh/patchcourt/internal/diff/finding"
	"github.com/orurh/patchcourt/internal/model"
	"github.com/orurh/patchcourt/internal/reportmodel"
	"sort"
)

const graphSchemaVersion = "patchcourt.review_graph.v1"

type ReviewGraph struct {
	SchemaVersion string            `json:"schema_version"`
	Nodes         []ReviewGraphNode `json:"nodes"`
	Edges         []ReviewGraphEdge `json:"edges"`
}

type ReviewGraphNode struct {
	ID                    string `json:"id"`
	Label                 string `json:"label"`
	BeforeDependencyCount int    `json:"before_dependency_count,omitempty"`
	AfterDependencyCount  int    `json:"after_dependency_count,omitempty"`
	Changed               bool   `json:"changed,omitempty"`
	RiskPoints            int    `json:"risk_points,omitempty"`
	FindingCount          int    `json:"finding_count,omitempty"`
}

type ReviewGraphEdge struct {
	From        string   `json:"from"`
	To          string   `json:"to"`
	BeforeCount int      `json:"before_count,omitempty"`
	AfterCount  int      `json:"after_count,omitempty"`
	Movement    string   `json:"movement"`
	FindingIDs  []string `json:"finding_ids,omitempty"`
}

func BuildReviewGraph(result reportmodel.ReviewResult) ReviewGraph {
	beforeCounts := layerEdgeCountsFromProject(result.BeforeProject)
	afterCounts := layerEdgeCountsFromProject(result.AfterProject)
	changeIndex := layerEdgeChangeIndex(result.LayerEdgeChanges)
	findingIDsByEdge := findingIDsByLayerEdge(result.FindingChanges)

	keys := mergedGraphKeys(beforeCounts, afterCounts, changeIndex, findingIDsByEdge)

	nodes := make(map[string]*ReviewGraphNode)
	edges := make([]ReviewGraphEdge, 0, len(keys))

	for _, key := range keys {
		from, to := depdiff.SplitLayerEdgeKey(key)
		if from == "" || to == "" {
			continue
		}

		beforeCount := beforeCounts[key]
		afterCount := afterCounts[key]
		movement := graphMovement(beforeCount, afterCount)

		if change, ok := changeIndex[key]; ok {
			movement = string(change.Kind)
			beforeCount = change.BeforeCount
			afterCount = change.AfterCount
		}

		findingIDs := findingIDsByEdge[key]

		edges = append(edges, ReviewGraphEdge{
			From:        from,
			To:          to,
			BeforeCount: beforeCount,
			AfterCount:  afterCount,
			Movement:    movement,
			FindingIDs:  findingIDs,
		})

		fromNode := ensureGraphNode(nodes, from)
		toNode := ensureGraphNode(nodes, to)

		fromNode.BeforeDependencyCount += beforeCount
		fromNode.AfterDependencyCount += afterCount
		toNode.BeforeDependencyCount += beforeCount
		toNode.AfterDependencyCount += afterCount

		if movement != "unchanged" {
			fromNode.Changed = true
			toNode.Changed = true
		}

		if len(findingIDs) > 0 {
			fromNode.FindingCount += len(findingIDs)
			fromNode.RiskPoints += graphFindingRiskPoints(result.FindingChanges, findingIDs)
		}
	}

	nodeRows := make([]ReviewGraphNode, 0, len(nodes))
	for _, node := range nodes {
		nodeRows = append(nodeRows, *node)
	}

	sort.Slice(nodeRows, func(i, j int) bool {
		return nodeRows[i].ID < nodeRows[j].ID
	})

	sort.Slice(edges, func(i, j int) bool {
		left := edges[i].From + "->" + edges[i].To
		right := edges[j].From + "->" + edges[j].To
		return left < right
	})

	return ReviewGraph{
		SchemaVersion: graphSchemaVersion,
		Nodes:         nodeRows,
		Edges:         edges,
	}
}

func layerEdgeCountsFromProject(project *model.ProjectModel) map[string]int {
	counts := make(map[string]int)
	if project == nil {
		return counts
	}

	for _, dependency := range project.Dependencies {
		if dependency.External || !dependency.Resolved {
			continue
		}
		if dependency.FromLayer == "" || dependency.ToLayer == "" {
			continue
		}
		if dependency.FromLayer == dependency.ToLayer {
			continue
		}

		counts[depdiff.LayerEdgeKey(dependency.FromLayer, dependency.ToLayer)]++
	}

	return counts
}

func layerEdgeChangeIndex(changes []depdiff.LayerEdgeChange) map[string]depdiff.LayerEdgeChange {
	index := make(map[string]depdiff.LayerEdgeChange, len(changes))

	for _, change := range changes {
		if change.FromLayer == "" || change.ToLayer == "" {
			continue
		}

		index[depdiff.LayerEdgeKey(change.FromLayer, change.ToLayer)] = change
	}

	return index
}

func findingIDsByLayerEdge(changes []findingdiff.FindingChange) map[string][]string {
	result := make(map[string][]string)

	for _, change := range changes {
		finding := change.After
		if finding == nil {
			finding = change.Before
		}
		if finding == nil || finding.ID == "" {
			continue
		}

		for _, evidence := range finding.Evidence {
			if evidence.FromLayer == "" || evidence.ToLayer == "" {
				continue
			}

			key := depdiff.LayerEdgeKey(evidence.FromLayer, evidence.ToLayer)
			result[key] = appendUniqueString(result[key], finding.ID)
		}
	}

	for key := range result {
		sort.Strings(result[key])
	}

	return result
}

func graphFindingRiskPoints(changes []findingdiff.FindingChange, ids []string) int {
	if len(ids) == 0 {
		return 0
	}

	idSet := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		idSet[id] = struct{}{}
	}

	points := 0
	for _, change := range changes {
		finding := change.After
		if finding == nil {
			finding = change.Before
		}
		if finding == nil {
			continue
		}
		if _, ok := idSet[finding.ID]; !ok {
			continue
		}

		points += severityRiskPoints(finding.Severity)
	}

	return points
}

func severityRiskPoints(severity model.Severity) int {
	switch severity {
	case model.SeverityCritical:
		return 10
	case model.SeverityHigh:
		return 5
	case model.SeverityMedium:
		return 3
	case model.SeverityLow:
		return 1
	default:
		return 0
	}
}

func ensureGraphNode(nodes map[string]*ReviewGraphNode, id string) *ReviewGraphNode {
	if node, ok := nodes[id]; ok {
		return node
	}

	node := &ReviewGraphNode{
		ID:    id,
		Label: id,
	}
	nodes[id] = node

	return node
}

func mergedGraphKeys(
	before map[string]int,
	after map[string]int,
	changes map[string]depdiff.LayerEdgeChange,
	findings map[string][]string,
) []string {
	seen := make(map[string]struct{})

	for key := range before {
		seen[key] = struct{}{}
	}
	for key := range after {
		seen[key] = struct{}{}
	}
	for key := range changes {
		seen[key] = struct{}{}
	}
	for key := range findings {
		seen[key] = struct{}{}
	}

	keys := make([]string, 0, len(seen))
	for key := range seen {
		keys = append(keys, key)
	}

	sort.Strings(keys)
	return keys
}

func graphMovement(beforeCount int, afterCount int) string {
	switch {
	case beforeCount == 0 && afterCount > 0:
		return "added"
	case beforeCount > 0 && afterCount == 0:
		return "removed"
	case beforeCount != afterCount:
		return "changed"
	default:
		return "unchanged"
	}
}

func appendUniqueString(values []string, value string) []string {
	for _, existing := range values {
		if existing == value {
			return values
		}
	}

	return append(values, value)
}
