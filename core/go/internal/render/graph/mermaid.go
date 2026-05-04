package graph

import (
	"fmt"
	"io"
	"strings"

	"github.com/orurh/patchcourt/internal/analyzer/graph"
)

func WriteLayerGraphMermaid(w io.Writer, layerGraph graph.LayerGraph) {
	fmt.Fprintln(w, "graph TD")

	for _, node := range layerGraph.Nodes {
		fmt.Fprintf(w, "  %s[%q]\n", mermaidID(node), node)
	}

	if len(layerGraph.Edges) > 0 {
		fmt.Fprintln(w)
	}

	for _, edge := range layerGraph.Edges {
		if edge.Count > 0 {
			fmt.Fprintf(w, "  %s -->|%d| %s\n", mermaidID(edge.From), edge.Count, mermaidID(edge.To))
		} else {
			fmt.Fprintf(w, "  %s --> %s\n", mermaidID(edge.From), mermaidID(edge.To))
		}
	}
}

func mermaidID(value string) string {
	replacer := strings.NewReplacer(
		"-", "_",
		".", "_",
		"/", "_",
		" ", "_",
	)
	return replacer.Replace(value)
}
