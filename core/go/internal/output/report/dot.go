package report

import (
	"fmt"
	"io"
	"strconv"
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

	if edge.Count > 0 {
		attrs = append(attrs, fmt.Sprintf(`label="%d"`, edge.Count))

		penWidth := 1
		switch {
		case edge.Count >= 20:
			penWidth = 5
		case edge.Count >= 10:
			penWidth = 4
		case edge.Count >= 5:
			penWidth = 3
		case edge.Count >= 2:
			penWidth = 2
		}

		attrs = append(attrs, `penwidth="`+strconv.Itoa(penWidth)+`"`)
	}

	if edge.Violation {
		if edge.Count > 0 {
			attrs[0] = fmt.Sprintf(`label="%d violation"`, edge.Count)
		} else {
			attrs = append(attrs, `label="violation"`)
		}

		attrs = append(attrs, `color="red"`)
		attrs = append(attrs, `fontcolor="red"`)

		if edge.Count <= 1 {
			attrs = append(attrs, `penwidth="2"`)
		}
	}

	return strings.Join(attrs, ", ")
}
