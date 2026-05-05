package review

import (
	"strings"

	"github.com/orurh/patchcourt/internal/reportmodel"
)

type ReviewView struct {
	Title       string
	Description string

	RiskLevel  string
	RiskPoints int

	SummaryCards   []ReviewMetricCard
	Impact         []ReviewImpactColumn
	LayerGraph     ReviewLayerGraph
	ChangedFiles   []string
	RiskReasons    []ReviewRiskReason
	DependencyRows []ReviewDependencyRow
	LayerEdgeRows  []ReviewLayerEdgeRow
	RawCounts      []ReviewMetricCard
}

type ReviewMetricCard struct {
	Title string
	Value int
}

type ReviewImpactColumn struct {
	Title string
	Class string
	Items []reportmodel.ReviewImpactItem
}

type ReviewLayerGraph struct {
	Title       string
	Description string
	Rows        []ReviewLayerGraphRow
}

type ReviewLayerGraphRow struct {
	Kind      string
	FromLayer string
	ToLayer   string
	FromID    string
	ToID      string
}

type ReviewRiskReason struct {
	Points  int
	Message string
}

type ReviewDependencyRow struct {
	Kind      string
	Key       string
	From      string
	To        string
	FromLayer string
	ToLayer   string
	Usage     string
}

type ReviewLayerEdgeRow struct {
	Kind        string
	FromLayer   string
	ToLayer     string
	BeforeCount int
	AfterCount  int
}

func BuildReviewView(result reportmodel.ReviewResult) ReviewView {
	return ReviewView{
		Title:       "Review report",
		Description: "Diff-aware architecture review generated from deterministic project facts.",
		RiskLevel:   string(result.Risk.Level),
		RiskPoints:  result.Risk.Points,
		SummaryCards: []ReviewMetricCard{
			{Title: "Contract changes", Value: result.Summary.ContractChanges},
			{Title: "Dependency changes", Value: result.Summary.DependencyChanges},
			{Title: "Layer edge changes", Value: result.Summary.LayerEdgeChanges},
			{Title: "Finding changes", Value: result.Summary.FindingChanges},
			{Title: "Added findings", Value: result.Summary.AddedFindings},
			{Title: "Removed findings", Value: result.Summary.RemovedFindings},
		},
		Impact: []ReviewImpactColumn{
			{Title: "Worse", Class: "bad", Items: result.Impact.Worse},
			{Title: "Better", Class: "good", Items: result.Impact.Better},
			{Title: "Unchanged debt", Class: "neutral", Items: result.Impact.UnchangedDebt},
		},
		LayerGraph: ReviewLayerGraph{
			Title:       "Layer impact graph",
			Description: "Mermaid graph of layer edges touched by this review.",
			Rows:        buildLayerGraphRows(result),
		},
		ChangedFiles:   result.ChangedFiles,
		RiskReasons:    buildRiskReasons(result),
		DependencyRows: buildDependencyRows(result),
		LayerEdgeRows:  buildLayerEdgeRows(result),
		RawCounts: []ReviewMetricCard{
			{Title: "Contract changes", Value: len(result.ContractChanges)},
			{Title: "Dependency changes", Value: len(result.DependencyChanges)},
			{Title: "Layer edge changes", Value: len(result.LayerEdgeChanges)},
			{Title: "Finding changes", Value: len(result.FindingChanges)},
		},
	}
}

func buildLayerGraphRows(result reportmodel.ReviewResult) []ReviewLayerGraphRow {
	rows := make([]ReviewLayerGraphRow, 0, len(result.LayerEdgeChanges))

	for _, change := range result.LayerEdgeChanges {
		if change.FromLayer == "" || change.ToLayer == "" {
			continue
		}

		rows = append(rows, ReviewLayerGraphRow{
			Kind:      string(change.Kind),
			FromLayer: change.FromLayer,
			ToLayer:   change.ToLayer,
			FromID:    mermaidNodeID(change.FromLayer),
			ToID:      mermaidNodeID(change.ToLayer),
		})
	}

	return rows
}

func buildRiskReasons(result reportmodel.ReviewResult) []ReviewRiskReason {
	reasons := make([]ReviewRiskReason, 0, len(result.Risk.Reasons))

	for _, reason := range result.Risk.Reasons {
		reasons = append(reasons, ReviewRiskReason{
			Points:  reason.Points,
			Message: reason.Message,
		})
	}

	return reasons
}

func buildDependencyRows(result reportmodel.ReviewResult) []ReviewDependencyRow {
	rows := make([]ReviewDependencyRow, 0, len(result.DependencyChanges))

	for _, change := range result.DependencyChanges {
		dep := change.After
		if dep == nil {
			dep = change.Before
		}

		if dep == nil {
			continue
		}

		to := dep.ToFile
		if to == "" {
			to = dep.Target
		}

		rows = append(rows, ReviewDependencyRow{
			Kind:      string(change.Kind),
			Key:       change.Key,
			From:      dep.FromFile,
			To:        to,
			FromLayer: dep.FromLayer,
			ToLayer:   dep.ToLayer,
			Usage:     string(dep.Usage),
		})
	}

	return rows
}

func buildLayerEdgeRows(result reportmodel.ReviewResult) []ReviewLayerEdgeRow {
	rows := make([]ReviewLayerEdgeRow, 0, len(result.LayerEdgeChanges))

	for _, change := range result.LayerEdgeChanges {
		rows = append(rows, ReviewLayerEdgeRow{
			Kind:        string(change.Kind),
			FromLayer:   change.FromLayer,
			ToLayer:     change.ToLayer,
			BeforeCount: change.BeforeCount,
			AfterCount:  change.AfterCount,
		})
	}

	return rows
}

func mermaidNodeID(value string) string {
	var b strings.Builder
	b.WriteString("n_")

	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			continue
		}

		b.WriteRune('_')
	}

	return b.String()
}
