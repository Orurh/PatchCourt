package report

import (
	"fmt"
	"io"
	"strings"

	"github.com/orurh/patchcourt/internal/model"
)

func WriteScanMarkdown(w io.Writer, project *model.ProjectModel) {
	summary := model.BuildScanSummary(project)

	fmt.Fprintln(w, "# PatchCourt Scan Report")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "## Summary")
	fmt.Fprintln(w)
	fmt.Fprintf(w, "- Root: `%s`\n", project.Root)
	fmt.Fprintf(w, "- C++ headers: %d\n", summary.CPPHeaders)
	fmt.Fprintf(w, "- C++ sources: %d\n", summary.CPPSources)
	fmt.Fprintf(w, "- C++ tests: %d\n", summary.CPPTests)
	fmt.Fprintf(w, "- Go files: %d\n", summary.GoFiles)
	fmt.Fprintf(w, "- Symbols: %d\n", summary.Symbols)
	fmt.Fprintln(w)
	fmt.Fprintln(w, "## File roles")
	fmt.Fprintln(w)
	fmt.Fprintf(w, "- Production: %d\n", summary.ProductionFiles)
	fmt.Fprintf(w, "- Test: %d\n", summary.TestFiles)
	fmt.Fprintf(w, "- Generated: %d\n", summary.GeneratedFiles)
	fmt.Fprintf(w, "- External: %d\n", summary.ExternalFiles)
	fmt.Fprintf(w, "- Config: %d\n", summary.ConfigFiles)
	fmt.Fprintf(w, "- Unknown: %d\n", summary.UnknownFiles)
	fmt.Fprintln(w)
	fmt.Fprintln(w, "## Dependencies")
	fmt.Fprintln(w)
	fmt.Fprintf(w, "- Total edges: %d\n", summary.TotalEdges)
	fmt.Fprintf(w, "- Resolved: %d\n", summary.Resolved)
	fmt.Fprintf(w, "- Unresolved: %d\n", summary.Unresolved)
	fmt.Fprintf(w, "- External: %d\n", summary.External)
	fmt.Fprintln(w)
	fmt.Fprintln(w, "## Findings")
	fmt.Fprintln(w)

	if len(project.Findings) == 0 {
		fmt.Fprintln(w, "No findings.")
		return
	}

	for _, finding := range project.Findings {
		writeFindingMarkdown(w, finding)
	}
}

func writeFindingMarkdown(w io.Writer, finding model.Finding) {
	fmt.Fprintf(w, "### %s: %s\n", strings.ToUpper(string(finding.Severity)), finding.Title)
	fmt.Fprintln(w)

	if finding.Risk != "" {
		fmt.Fprintf(w, "**Risk:** %s\n", finding.Risk)
		fmt.Fprintln(w)
	}

	if len(finding.Evidence) > 0 {
		fmt.Fprintln(w, "**Evidence:**")
		fmt.Fprintln(w)

		for _, evidence := range finding.Evidence {
			if evidence.File != "" {
				fmt.Fprintf(w, "- `%s`: %s\n", evidence.File, evidence.Message)
			} else {
				fmt.Fprintf(w, "- %s\n", evidence.Message)
			}
		}

		fmt.Fprintln(w)
	}

	if finding.Suggestion != "" {
		fmt.Fprintf(w, "**Suggestion:** %s\n", finding.Suggestion)
		fmt.Fprintln(w)
	}

	fmt.Fprintf(w, "**Confidence:** %s\n", finding.Confidence)
	fmt.Fprintln(w)
}
