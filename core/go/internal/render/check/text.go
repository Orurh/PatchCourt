package check

import (
	"fmt"
	"io"

	"github.com/orurh/patchcourt/internal/reportmodel"
)

func WriteCheckReportText(w io.Writer, result reportmodel.CheckReport) {
	fmt.Fprintln(w, "PatchCourt check")
	fmt.Fprintln(w)

	fmt.Fprintf(w, "Root: %s\n", result.Root)
	if result.ConfigPath != "" {
		fmt.Fprintf(w, "Config: %s\n", result.ConfigPath)
	} else {
		fmt.Fprintln(w, "Config: defaults")
	}
	fmt.Fprintf(w, "Out: %s\n", result.OutDir)
	if result.StatePath != "" {
		fmt.Fprintln(w)
		fmt.Fprintln(w, "State:")
		fmt.Fprintf(w, "  saved latest: %s\n", result.StatePath)
	}
	fmt.Fprintln(w)

	writeStructuredCheckSummary(w, result)
	fmt.Fprintln(w)

	writeStructuredCheckArtifacts(w, result.Artifacts)
	fmt.Fprintln(w)

	writeStructuredCheckTopFindings(w, result.TopFindings)
	fmt.Fprintln(w)

	writeStructuredCheckEdges(w, "Most coupled edges:", result.MostCoupledEdges, false)
	fmt.Fprintln(w)

	writeStructuredCheckEdges(w, "Suspicious edges:", result.SuspiciousEdges, true)
	fmt.Fprintln(w)

	writeStructuredCheckNextSteps(w, result.NextSteps)
}

func writeStructuredCheckSummary(w io.Writer, result reportmodel.CheckReport) {
	summary := result.Summary

	fmt.Fprintln(w, "Summary:")
	fmt.Fprintf(w, "  production files: %d\n", summary.ProductionFiles)
	fmt.Fprintf(w, "  test files:       %d\n", summary.TestFiles)
	fmt.Fprintf(w, "  dependencies:     %d\n", summary.TotalEdges)
	fmt.Fprintf(w, "  resolved:         %d\n", summary.Resolved)
	fmt.Fprintf(w, "  unresolved:       %d\n", summary.Unresolved)
	fmt.Fprintf(w, "  findings:         %d\n", result.FindingCount)
	fmt.Fprintf(w, "  graph nodes:      %d\n", result.GraphNodeCount)
	fmt.Fprintf(w, "  graph edges:      %d\n", result.GraphEdgeCount)
}

func writeStructuredCheckArtifacts(w io.Writer, artifacts []reportmodel.CheckArtifact) {
	fmt.Fprintln(w, "Artifacts:")

	if len(artifacts) == 0 {
		fmt.Fprintln(w, "  none")
		return
	}

	for _, artifact := range artifacts {
		fmt.Fprintf(w, "  - %s: %s\n", artifact.Name, artifact.Path)
	}
}

func writeStructuredCheckTopFindings(w io.Writer, findings []reportmodel.FindingSummary) {
	fmt.Fprintln(w, "Top findings:")

	if len(findings) == 0 {
		fmt.Fprintln(w, "  none")
		return
	}

	for _, finding := range findings {
		fmt.Fprintf(
			w,
			"  - [%s/%s] %s — %s\n",
			finding.Severity,
			finding.Kind,
			finding.ID,
			finding.Title,
		)
	}
}

func writeStructuredCheckEdges(w io.Writer, title string, edges []reportmodel.EdgeSummary, includeFinding bool) {
	fmt.Fprintln(w, title)

	if len(edges) == 0 {
		fmt.Fprintln(w, "  none")
		return
	}

	for _, edge := range edges {
		if includeFinding && edge.FindingID != "" {
			fmt.Fprintf(w, "  %d  %s -> %s  [%s]\n", edge.Count, edge.From, edge.To, edge.FindingID)
			continue
		}

		fmt.Fprintf(w, "  %d  %s -> %s\n", edge.Count, edge.From, edge.To)
	}
}

func writeStructuredCheckNextSteps(w io.Writer, steps []reportmodel.NextStep) {
	fmt.Fprintln(w, "Next:")

	if len(steps) == 0 {
		fmt.Fprintln(w, "  none")
		return
	}

	for _, step := range steps {
		fmt.Fprintf(w, "  %s\n", step.Command)
	}
}
