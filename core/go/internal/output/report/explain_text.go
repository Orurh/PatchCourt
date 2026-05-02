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

	if len(finding.Evidence) > 0 {
		fmt.Fprintln(w)
		fmt.Fprintln(w, "Evidence:")

		for _, evidence := range finding.Evidence {
			if evidence.File != "" {
				fmt.Fprintf(w, "  - %s: %s\n", evidence.File, evidence.Message)
			} else {
				fmt.Fprintf(w, "  - %s\n", evidence.Message)
			}

			if evidence.LineStart > 0 {
				if evidence.LineEnd > evidence.LineStart {
					fmt.Fprintf(w, "    lines: %d-%d\n", evidence.LineStart, evidence.LineEnd)
				} else {
					fmt.Fprintf(w, "    line: %d\n", evidence.LineStart)
				}
			}

			if evidence.Snippet != "" {
				fmt.Fprintf(w, "    snippet: %s\n", evidence.Snippet)
			}
		}
	}
}
