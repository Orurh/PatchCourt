package cli

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/orurh/patchcourt/internal/app"
	"github.com/orurh/patchcourt/internal/output/report"
)

// writeJSON печатает значение в формате JSON.
func writeJSON(w io.Writer, value any) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}

// renderScanResult печатает результат сканирования в выбранном формате.
//
// При неизвестном формате возвращает ошибку.
func (r *Runner) renderScanResult(format app.ScanFormat, result *app.ScanResult) error {
	switch format {
	case app.ScanFormatJSON:
		return writeJSON(r.stdout, result.Project)
	case app.ScanFormatMarkdown:
		report.WriteScanMarkdown(r.stdout, result.Project)
		return nil
	case app.ScanFormatText, "":
		report.WriteScanText(r.stdout, result.Project)
		return nil
	default:
		return fmt.Errorf("unknown scan format: %s", format)
	}
}

// renderGraphResult печатает результат генерации графа в выбранном формате.
//
// При неизвестном формате возвращает ошибку.
func (r *Runner) renderGraphResult(format app.GraphFormat, result *app.GraphResult) error {
	switch format {
	case app.GraphFormatJSON:
		return writeJSON(r.stdout, result.LayerGraph)
	case app.GraphFormatDOT:
		report.WriteLayerGraphDOT(r.stdout, result.LayerGraph)
		return nil
	case app.GraphFormatMermaid, "":
		report.WriteLayerGraphMermaid(r.stdout, result.LayerGraph)
		return nil
	default:
		return fmt.Errorf("unknown graph format: %s", format)
	}
}

// renderReviewResult печатает результат сравнения в выбранном формате.
//
// При неизвестном формате возвращает ошибку.
func (r *Runner) renderReviewResult(format app.ReviewFormat, result *app.ReviewResult) error {
	switch format {
	case app.ReviewFormatJSON:
		return writeJSON(r.stdout, result)
	case app.ReviewFormatText, "":
		report.WriteReviewText(r.stdout, report.ReviewTextResult{
			Summary:           result.Summary,
			Risk:              result.Risk,
			ContractChanges:   result.ContractChanges,
			DependencyChanges: result.DependencyChanges,
			LayerEdgeChanges:  result.LayerEdgeChanges,
			FindingChanges:    result.FindingChanges,
		})
		return nil
	default:
		return fmt.Errorf("unknown review format: %s", format)
	}
}

func (r *Runner) renderExplainResult(format app.ExplainFormat, result *app.ExplainResult) error {
	switch format {
	case app.ExplainFormatJSON:
		return writeJSON(r.stdout, result)
	case app.ExplainFormatText, "":
		report.WriteExplainText(r.stdout, result)
		return nil
	default:
		return fmt.Errorf("unknown explain format: %s", format)
	}
}
