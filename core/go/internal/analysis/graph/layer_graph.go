package graph

import (
	"fmt"
	"sort"

	"github.com/orurh/patchcourt/internal/config"
	"github.com/orurh/patchcourt/internal/model"
)

const maxLayerEdgeEvidence = 5

type LayerEdge struct {
	From      string           `json:"from"`
	To        string           `json:"to"`
	Count     int              `json:"count"`
	Violation bool             `json:"violation"`
	Evidence  []model.Evidence `json:"evidence,omitempty"`
}

type LayerGraph struct {
	Nodes []string    `json:"nodes"`
	Edges []LayerEdge `json:"edges"`
}

func BuildLayerGraph(project *model.ProjectModel, cfg *config.Config) LayerGraph {
	nodeSet := make(map[string]struct{})
	edgeSet := make(map[string]LayerEdge)

	if project == nil {
		return LayerGraph{}
	}

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

		key := dep.FromLayer + "->" + dep.ToLayer
		edge := edgeSet[key]

		if edge.From == "" {
			edge = LayerEdge{
				From: dep.FromLayer,
				To:   dep.ToLayer,
			}
		}

		edge.Count++
		edge.Violation = edge.Violation || isViolation(dep.FromLayer, dep.ToLayer, cfg)

		if len(edge.Evidence) < maxLayerEdgeEvidence {
			edge.Evidence = append(edge.Evidence, layerEdgeEvidence(dep))
		}

		edgeSet[key] = edge

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

func layerEdgeEvidence(dep model.DependencyEdge) model.Evidence {
	target := dep.ToFile
	if target == "" {
		target = dep.Target
	}

	message := fmt.Sprintf(
		"includes %s, creating layer dependency %s -> %s",
		target,
		dep.FromLayer,
		dep.ToLayer,
	)

	if dep.Usage != "" {
		message = fmt.Sprintf("%s [usage=%s]", message, dep.Usage)
	}

	return model.Evidence{
		File:    dep.FromFile,
		Message: message,
	}
}

func isViolation(fromLayer string, toLayer string, cfg *config.Config) bool {
	if cfg == nil || len(cfg.Layers) == 0 {
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
