package report

import (
	"fmt"
	"io"
	"strings"

	"github.com/orurh/patchcourt/internal/analysis/graph"
)

func WriteLayerGraphDOT(w io.Writer, layerGraph graph.LayerGraph) {
	fmt.Fprintln(w, "digraph PatchCourtLayers {")
	fmt.Fprintln(w, "  rankdir=LR;")
	fmt.Fprintln(w)

	for _, node := range layerGraph.Nodes {
		fmt.Fprintf(w, "  %q;\n", node)
	}

	if len(layerGraph.Edges) > 0 {
		fmt.Fprintln(w)
	}

	for _, edge := range layerGraph.Edges {
		attrs := dotEdgeAttrs(edge)
		if attrs == "" {
			fmt.Fprintf(w, "  %q -> %q;\n", edge.From, edge.To)
		} else {
			fmt.Fprintf(w, "  %q -> %q [%s];\n", edge.From, edge.To, attrs)
		}
	}

	fmt.Fprintln(w, "}")
}

func dotEdgeAttrs(edge graph.LayerEdge) string {
	var attrs []string

	if edge.Violation {
		attrs = append(attrs, `label="violation"`)
		attrs = append(attrs, `color="red"`)
		attrs = append(attrs, `penwidth="2"`)
	}

	return strings.Join(attrs, ", ")
}
