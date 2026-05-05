package review

import (
	"strings"

	contracts "github.com/orurh/patchcourt/internal/diff/contract"
	"github.com/orurh/patchcourt/internal/render/reviewquestions"
	"github.com/orurh/patchcourt/internal/reportmodel"
)

type ReviewView struct {
	Title       string
	Description string

	RiskLevel  string
	RiskPoints int

	SummaryCards    []ReviewMetricCard
	Impact          []ReviewImpactColumn
	LayerGraph      ReviewLayerGraph
	ChangedFiles    []string
	RiskReasons     []ReviewRiskReason
	ContractRows    []ReviewContractRow
	DependencyRows  []ReviewDependencyRow
	LayerEdgeRows   []ReviewLayerEdgeRow
	FindingRows     []ReviewFindingRow
	ReviewQuestions []ReviewQuestion
	RawCounts       []ReviewMetricCard
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

type ReviewContractRow struct {
	Kind             string
	Impact           string
	SymbolKey        string
	File             string
	BeforeLine       int
	AfterLine        int
	BeforeSignature  string
	AfterSignature   string
	AddedModifiers   string
	RemovedModifiers string
}

type ReviewFindingRow struct {
	Kind     string
	ID       string
	Severity string
	Title    string
}

type ReviewQuestion struct {
	Text string
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
		ChangedFiles:    result.ChangedFiles,
		RiskReasons:     buildRiskReasons(result),
		ContractRows:    buildContractRows(result),
		DependencyRows:  buildDependencyRows(result),
		LayerEdgeRows:   buildLayerEdgeRows(result),
		FindingRows:     buildFindingRows(result),
		ReviewQuestions: buildReviewQuestions(result, 10),
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

func buildContractRows(result reportmodel.ReviewResult) []ReviewContractRow {
	rows := make([]ReviewContractRow, 0, len(result.ContractChanges))

	for _, change := range result.ContractChanges {
		file := ""
		if change.After != nil {
			file = change.After.File
		}
		if file == "" && change.Before != nil {
			file = change.Before.File
		}

		beforeLine := 0
		beforeSignature := ""
		if change.Before != nil {
			beforeLine = change.Before.Line
			beforeSignature = change.Before.Signature
		}

		afterLine := 0
		afterSignature := ""
		if change.After != nil {
			afterLine = change.After.Line
			afterSignature = change.After.Signature
		}

		rows = append(rows, ReviewContractRow{
			Kind:             string(change.Kind),
			Impact:           contractChangeImpact(change),
			SymbolKey:        change.SymbolKey,
			File:             file,
			BeforeLine:       beforeLine,
			AfterLine:        afterLine,
			BeforeSignature:  beforeSignature,
			AfterSignature:   afterSignature,
			AddedModifiers:   strings.Join(change.AddedMods, ", "),
			RemovedModifiers: strings.Join(change.RemovedMods, ", "),
		})
	}

	return rows
}

func contractChangeImpact(change contracts.SymbolChange) string {
	switch change.Kind {
	case contracts.ChangeKindRemoved:
		return "breaking"

	case contracts.ChangeKindSignatureChanged:
		return "breaking"

	case contracts.ChangeKindAdded:
		return "additive"

	case contracts.ChangeKindModifiersChanged:
		if containsString(change.AddedMods, "pure_virtual") {
			return "breaking"
		}

		if containsAnyString(change.RemovedMods, []string{
			"virtual",
			"const",
			"noexcept",
			"override",
			"final",
			"pure_virtual",
		}) {
			return "risky"
		}

		if containsAnyString(change.AddedMods, []string{
			"final",
			"override",
			"noexcept",
		}) {
			return "risky"
		}

		return "informational"

	default:
		return "informational"
	}
}

func containsAnyString(values []string, targets []string) bool {
	for _, target := range targets {
		if containsString(values, target) {
			return true
		}
	}

	return false
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}

	return false
}

func buildFindingRows(result reportmodel.ReviewResult) []ReviewFindingRow {
	rows := make([]ReviewFindingRow, 0, len(result.FindingChanges))

	for _, change := range result.FindingChanges {
		finding := change.After
		if finding == nil {
			finding = change.Before
		}

		if finding == nil {
			rows = append(rows, ReviewFindingRow{
				Kind: string(change.Kind),
				ID:   change.ID,
			})
			continue
		}

		rows = append(rows, ReviewFindingRow{
			Kind:     string(change.Kind),
			ID:       change.ID,
			Severity: string(finding.Severity),
			Title:    finding.Title,
		})
	}

	return rows
}

func buildReviewQuestions(result reportmodel.ReviewResult, limit int) []ReviewQuestion {
	questions := reviewquestions.Build(result, limit)
	rows := make([]ReviewQuestion, 0, len(questions))

	for _, question := range questions {
		rows = append(rows, ReviewQuestion{Text: question.Text})
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
