package graph

import (
	"sort"

	"github.com/orurh/patchcourt/internal/config"
	"github.com/orurh/patchcourt/internal/model"
)

type LayerEdge struct {
	From      string `json:"from"`
	To        string `json:"to"`
	Violation bool   `json:"violation"`
}

type LayerGraph struct {
	Nodes []string    `json:"nodes"`
	Edges []LayerEdge `json:"edges"`
}

func BuildLayerGraph(project *model.ProjectModel, cfg *config.Config) LayerGraph {
	nodeSet := make(map[string]struct{})
	edgeSet := make(map[string]LayerEdge)

	for _, file := range project.Files {
		if file.Layer != "" {
			nodeSet[file.Layer] = struct{}{}
		}
	}

	for _, dep := range project.Dependencies {
		if dep.External || !dep.Resolved {
			continue
		}

		if dep.FromLayer == "" || dep.ToLayer == "" {
			continue
		}

		if dep.FromLayer == dep.ToLayer {
			continue
		}

		edge := LayerEdge{
			From:      dep.FromLayer,
			To:        dep.ToLayer,
			Violation: isViolation(dep.FromLayer, dep.ToLayer, cfg),
		}

		key := edge.From + "->" + edge.To

		if existing, ok := edgeSet[key]; ok {
			existing.Violation = existing.Violation || edge.Violation
			edgeSet[key] = existing
		} else {
			edgeSet[key] = edge
		}

		nodeSet[dep.FromLayer] = struct{}{}
		nodeSet[dep.ToLayer] = struct{}{}
	}

	nodes := make([]string, 0, len(nodeSet))
	for node := range nodeSet {
		nodes = append(nodes, node)
	}
	sort.Strings(nodes)

	edges := make([]LayerEdge, 0, len(edgeSet))
	for _, edge := range edgeSet {
		edges = append(edges, edge)
	}
	sort.Slice(edges, func(i, j int) bool {
		if edges[i].From == edges[j].From {
			return edges[i].To < edges[j].To
		}
		return edges[i].From < edges[j].From
	})

	return LayerGraph{
		Nodes: nodes,
		Edges: edges,
	}
}

func isViolation(fromLayer string, toLayer string, cfg *config.Config) bool {
	if cfg == nil {
		return false
	}

	layer, ok := cfg.Layers[fromLayer]
	if !ok {
		return false
	}

	for _, allowed := range layer.MayDependOn {
		if allowed == toLayer {
			return false
		}
	}

	return true
}
