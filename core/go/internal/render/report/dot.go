package report

import (
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/orurh/patchcourt/internal/analysis/graph"
)

func WriteLayerGraphDOT(w io.Writer, layerGraph graph.LayerGraph) {
	connectedNodes := connectedLayerNodes(layerGraph)

	fmt.Fprintln(w, "digraph PatchCourtLayers {")
	fmt.Fprintln(w, "  rankdir=LR;")
	fmt.Fprintln(w, `  graph [bgcolor="white", overlap=false, splines=true, concentrate=true];`)
	fmt.Fprintln(w, `  node [shape=ellipse, style="filled", fillcolor="white", color="#111827", fontcolor="#111827", fontsize=11];`)
	fmt.Fprintln(w, `  edge [color="#6b7280", fontcolor="#374151", fontsize=10, arrowsize=0.8];`)
	fmt.Fprintln(w)

	for _, node := range layerGraph.Nodes {
		attrs := dotNodeAttrs(node, connectedNodes[node])
		if attrs == "" {
			fmt.Fprintf(w, "  %q;\n", node)
		} else {
			fmt.Fprintf(w, "  %q [%s];\n", node, attrs)
		}
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

func connectedLayerNodes(layerGraph graph.LayerGraph) map[string]bool {
	result := make(map[string]bool, len(layerGraph.Nodes))

	for _, edge := range layerGraph.Edges {
		if edge.From != "" {
			result[edge.From] = true
		}

		if edge.To != "" {
			result[edge.To] = true
		}
	}

	return result
}

func dotNodeAttrs(node string, connected bool) string {
	attrs := []string{
		dotAttr("tooltip", node),
	}

	if !connected {
		attrs = append(attrs,
			dotAttr("style", "filled,dashed"),
			dotAttr("color", "#9ca3af"),
			dotAttr("fontcolor", "#6b7280"),
		)
	}

	return strings.Join(attrs, ", ")
}

func dotEdgeAttrs(edge graph.LayerEdge) string {
	var attrs []string

	label := dotEdgeLabel(edge)
	if label != "" {
		attrs = append(attrs, dotAttr("label", label))
	}

	attrs = append(attrs, dotAttr("penwidth", strconv.Itoa(dotEdgePenWidth(edge.Count))))

	color := "#6b7280"
	fontColor := "#374151"

	switch {
	case edge.Violation:
		color = "#dc2626"
		fontColor = "#dc2626"
	case len(edge.Evidence) > 0:
		color = "#f97316"
		fontColor = "#f97316"
	}

	attrs = append(attrs,
		dotAttr("color", color),
		dotAttr("fontcolor", fontColor),
		dotAttr("tooltip", dotEdgeTooltip(edge)),
		dotAttr("URL", dotEdgeURL(edge)),
	)

	return strings.Join(attrs, ", ")
}

func dotEdgeLabel(edge graph.LayerEdge) string {
	switch {
	case edge.Count > 0 && edge.Violation:
		return fmt.Sprintf("%d violation", edge.Count)
	case edge.Count > 0 && len(edge.Evidence) > 0:
		return fmt.Sprintf("%d ⚠", edge.Count)
	case edge.Count > 0:
		return strconv.Itoa(edge.Count)
	case edge.Violation:
		return "violation"
	case len(edge.Evidence) > 0:
		return "⚠"
	default:
		return ""
	}
}

func dotEdgeTooltip(edge graph.LayerEdge) string {
	var b strings.Builder

	if edge.From != "" || edge.To != "" {
		fmt.Fprintf(&b, "%s -> %s", edge.From, edge.To)
	}

	if edge.Count > 0 {
		if b.Len() > 0 {
			b.WriteString(": ")
		}
		fmt.Fprintf(&b, "%d dependencies", edge.Count)
	}

	if edge.Violation {
		if b.Len() > 0 {
			b.WriteString("; ")
		}
		b.WriteString("policy violation")
	} else if len(edge.Evidence) > 0 {
		if b.Len() > 0 {
			b.WriteString("; ")
		}
		b.WriteString("has evidence")
	}

	for _, evidence := range edge.Evidence {
		message := strings.TrimSpace(evidence.Message)
		if message == "" {
			continue
		}

		if b.Len() > 0 {
			b.WriteString("; ")
		}

		b.WriteString(message)
		break
	}

	return b.String()
}

func dotEdgeURL(edge graph.LayerEdge) string {
	return "#edge-" + dotURLPart(edge.From) + "-" + dotURLPart(edge.To)
}

func dotURLPart(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))

	replacer := strings.NewReplacer(
		" ", "-",
		"_", "-",
		".", "-",
		"/", "-",
		"\\", "-",
		":", "-",
	)

	value = replacer.Replace(value)

	var b strings.Builder
	lastDash := false

	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
			lastDash = false
		case r >= '0' && r <= '9':
			b.WriteRune(r)
			lastDash = false
		case r == '-':
			if !lastDash {
				b.WriteRune(r)
				lastDash = true
			}
		}
	}

	return strings.Trim(b.String(), "-")
}

func dotEdgePenWidth(count int) int {
	switch {
	case count >= 50:
		return 6
	case count >= 20:
		return 5
	case count >= 10:
		return 4
	case count >= 5:
		return 3
	case count >= 2:
		return 2
	default:
		return 1
	}
}

func dotAttr(name string, value string) string {
	return fmt.Sprintf(`%s=%q`, name, dotEscape(value))
}

func dotEscape(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, `"`, `\"`)
	value = strings.ReplaceAll(value, "\n", `\n`)
	return value
}
