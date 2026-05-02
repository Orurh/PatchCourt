package report

import (
	"fmt"
	"io"

	"github.com/orurh/patchcourt/internal/app"
	"github.com/orurh/patchcourt/internal/model"
)

func WriteExplainText(w io.Writer, result *app.ExplainResult) {
	fmt.Fprintln(w, "PatchCourt explain")
	fmt.Fprintln(w)

	if result.Source != "" {
		fmt.Fprintf(w, "Source: %s\n", result.Source)
		fmt.Fprintln(w)
	}

	writeExplainFindingText(w, result.Finding)
}

func writeExplainFindingText(w io.Writer, finding model.Finding) {
	fmt.Fprintf(w, "Finding: %s\n", finding.ID)

	if finding.Title != "" {
		fmt.Fprintf(w, "Title:   %s\n", finding.Title)
	}

	if finding.Kind != "" {
		fmt.Fprintf(w, "Kind:    %s\n", finding.Kind)
	}

	if finding.Severity != "" {
		fmt.Fprintf(w, "Severity: %s\n", finding.Severity)
	}

	if finding.Confidence != "" {
		fmt.Fprintf(w, "Confidence: %s\n", finding.Confidence)
	}

	if finding.Risk != "" {
		fmt.Fprintln(w)
		fmt.Fprintln(w, "Risk:")
		fmt.Fprintf(w, "  %s\n", finding.Risk)
	}

	if finding.Suggestion != "" {
		fmt.Fprintln(w)
		fmt.Fprintln(w, "Suggestion:")
		fmt.Fprintf(w, "  %s\n", finding.Suggestion)
	}
	writeExplainEvidence(w, finding.Evidence)
}

func writeExplainEvidence(w io.Writer, evidence []model.Evidence) {
	if len(evidence) == 0 {
		return
	}

	fmt.Fprintln(w)
	fmt.Fprintln(w, "Evidence:")

	for _, item := range evidence {
		if item.File != "" {
			fmt.Fprintf(w, "  - %s: %s\n", item.File, item.Message)
		} else {
			fmt.Fprintf(w, "  - %s\n", item.Message)
		}

		if item.LineStart > 0 {
			if item.LineEnd > item.LineStart {
				fmt.Fprintf(w, "    lines: %d-%d\n", item.LineStart, item.LineEnd)
			} else {
				fmt.Fprintf(w, "    line: %d\n", item.LineStart)
			}
		}

		if item.Snippet != "" {
			fmt.Fprintln(w, "    snippet:")
			fmt.Fprintf(w, "      %s\n", item.Snippet)
		}
	}
}
