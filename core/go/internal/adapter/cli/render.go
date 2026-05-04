package cli

import (
	"encoding/json"
	"fmt"
	"io"

	renderedge "github.com/orurh/patchcourt/internal/render/edge"
	renderexplain "github.com/orurh/patchcourt/internal/render/explain"
	rendergraph "github.com/orurh/patchcourt/internal/render/graph"
	renderreview "github.com/orurh/patchcourt/internal/render/review"
	renderscan "github.com/orurh/patchcourt/internal/render/scan"
	"github.com/orurh/patchcourt/internal/usecase"
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
func (r *Runner) renderScanResult(format usecase.ScanFormat, result *usecase.ScanResult) error {
	switch format {
	case usecase.ScanFormatJSON:
		return writeJSON(r.stdout, result.Project)
	case usecase.ScanFormatMarkdown:
		renderscan.WriteScanMarkdown(r.stdout, result.Project)
		return nil
	case usecase.ScanFormatText, "":
		renderscan.WriteScanText(r.stdout, result.Project)
		return nil
	default:
		return fmt.Errorf("unknown scan format: %s", format)
	}
}

// renderGraphResult печатает результат генерации графа в выбранном формате.
//
// При неизвестном формате возвращает ошибку.
func (r *Runner) renderGraphResult(format usecase.GraphFormat, result *usecase.GraphResult) error {
	switch format {
	case usecase.GraphFormatJSON:
		return writeJSON(r.stdout, result.LayerGraph)
	case usecase.GraphFormatDOT:
		rendergraph.WriteLayerGraphDOT(r.stdout, result.LayerGraph)
		return nil
	case usecase.GraphFormatMermaid, "":
		rendergraph.WriteLayerGraphMermaid(r.stdout, result.LayerGraph)
		return nil
	default:
		return fmt.Errorf("unknown graph format: %s", format)
	}
}

// renderReviewResult печатает результат сравнения в выбранном формате.
//
// При неизвестном формате возвращает ошибку.
func (r *Runner) renderReviewResult(format usecase.ReviewFormat, req usecase.ReviewRequest, result *usecase.ReviewResult) error {
	switch format {
	case usecase.ReviewFormatJSON:
		return writeJSON(r.stdout, result)
	case usecase.ReviewFormatMarkdown:
		renderreview.WriteReviewMarkdown(r.stdout, renderreview.ReviewMarkdownResult{
			Summary:           result.Summary,
			Risk:              result.Risk,
			Impact:            result.Impact,
			ContractChanges:   result.ContractChanges,
			DependencyChanges: result.DependencyChanges,
			LayerEdgeChanges:  result.LayerEdgeChanges,
			FindingChanges:    result.FindingChanges,
			AfterRoot:         req.AfterRoot,
			ConfigPath:        req.ConfigPath,
		})
		return nil
	case usecase.ReviewFormatText, "":
		renderreview.WriteReviewText(r.stdout, renderreview.ReviewTextResult{
			Summary:           result.Summary,
			Risk:              result.Risk,
			Impact:            result.Impact,
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

func (r *Runner) renderExplainResult(format usecase.ExplainFormat, result *usecase.ExplainResult) error {
	switch format {
	case usecase.ExplainFormatJSON:
		return writeJSON(r.stdout, result)
	case usecase.ExplainFormatText, "":
		renderexplain.WriteExplainText(r.stdout, result)
		return nil
	default:
		return fmt.Errorf("unknown explain format: %s", format)
	}
}

func (r *Runner) renderEdgeResult(format usecase.EdgeFormat, result *usecase.EdgeResult) error {
	switch format {
	case usecase.EdgeFormatJSON:
		return writeJSON(r.stdout, result)
	case usecase.EdgeFormatText, "":
		renderedge.WriteEdgeText(r.stdout, result)
		return nil
	default:
		return fmt.Errorf("unknown edge format: %s", format)
	}
}
