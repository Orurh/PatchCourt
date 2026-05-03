package report

import (
	"fmt"
	"io"

	"github.com/orurh/patchcourt/internal/model"
	"github.com/orurh/patchcourt/internal/reportmodel"
)

const maxEdgeTopFiles = 10

func WriteEdgeText(w io.Writer, result *reportmodel.EdgeResult) {
	fmt.Fprintln(w, "PatchCourt edge")
	fmt.Fprintln(w)

	if result.Source != "" {
		fmt.Fprintf(w, "Source: %s\n", result.Source)
	}

	fmt.Fprintf(w, "Edge: %s -> %s\n", result.FromLayer, result.ToLayer)
	fmt.Fprintf(w, "Count: %d\n", result.Count)
	fmt.Fprintln(w)

	writeEdgeFindingsText(w, result.Findings)
	fmt.Fprintln(w)

	writeEdgeUsageText(w, result.Usage)
	fmt.Fprintln(w)

	writeEdgeTopFilesText(w, "Top source files:", result.TopFromFiles)
	fmt.Fprintln(w)

	writeEdgeTopFilesText(w, "Top target files:", result.TopToFiles)
	fmt.Fprintln(w)

	writeEdgeDependenciesText(w, result.Dependencies, result.TruncatedDeps)
}

func writeEdgeFindingsText(w io.Writer, findings []model.Finding) {
	fmt.Fprintln(w, "Findings:")

	if len(findings) == 0 {
		fmt.Fprintln(w, "  none")
		return
	}

	for _, finding := range findings {
		if finding.ID != "" {
			fmt.Fprintf(w, "  - [%s/%s] %s — %s\n", finding.Severity, finding.Kind, finding.ID, finding.Title)
		} else {
			fmt.Fprintf(w, "  - [%s/%s] %s\n", finding.Severity, finding.Kind, finding.Title)
		}
	}
}

func writeEdgeUsageText(w io.Writer, usage reportmodel.EdgeUsageSummary) {
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintf(w, "  used:    %d\n", usage.Used)
	fmt.Fprintf(w, "  maybe:   %d\n", usage.Maybe)
	fmt.Fprintf(w, "  unused:  %d\n", usage.Unused)
	fmt.Fprintf(w, "  unknown: %d\n", usage.Unknown)
}

func writeEdgeTopFilesText(w io.Writer, title string, files []reportmodel.EdgeFileCount) {
	fmt.Fprintln(w, title)

	if len(files) == 0 {
		fmt.Fprintln(w, "  none")
		return
	}

	limit := len(files)
	if limit > maxEdgeTopFiles {
		limit = maxEdgeTopFiles
	}

	for i := 0; i < limit; i++ {
		fmt.Fprintf(w, "  %d  %s\n", files[i].Count, files[i].File)
	}

	if len(files) > limit {
		fmt.Fprintf(w, "  ... %d more\n", len(files)-limit)
	}
}

func writeEdgeDependenciesText(w io.Writer, deps []model.DependencyEdge, truncated int) {
	fmt.Fprintln(w, "Dependencies:")

	if len(deps) == 0 {
		fmt.Fprintln(w, "  none")
		return
	}

	currentFrom := ""
	for _, dep := range deps {
		if dep.FromFile != currentFrom {
			currentFrom = dep.FromFile
			fmt.Fprintf(w, "  %s\n", currentFrom)
		}

		target := dep.ToFile
		if target == "" {
			target = dep.Target
		}

		if dep.Usage != "" {
			fmt.Fprintf(w, "    -> %s [%s]\n", target, dep.Usage)
		} else {
			fmt.Fprintf(w, "    -> %s\n", target)
		}
	}

	if truncated > 0 {
		fmt.Fprintf(w, "  ... %d more dependencies\n", truncated)
	}
}
