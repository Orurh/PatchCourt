package review

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/orurh/patchcourt/internal/diff/contract"
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
	SymbolKey        string
	File             string
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

		beforeSignature := ""
		if change.Before != nil {
			beforeSignature = change.Before.Signature
		}

		afterSignature := ""
		if change.After != nil {
			afterSignature = change.After.Signature
		}

		rows = append(rows, ReviewContractRow{
			Kind:             string(change.Kind),
			SymbolKey:        change.SymbolKey,
			File:             file,
			BeforeSignature:  beforeSignature,
			AfterSignature:   afterSignature,
			AddedModifiers:   strings.Join(change.AddedMods, ", "),
			RemovedModifiers: strings.Join(change.RemovedMods, ", "),
		})
	}

	return rows
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
	if limit <= 0 {
		return nil
	}

	questions := make([]ReviewQuestion, 0, limit)

	for _, item := range result.Impact.Worse {
		if len(questions) >= limit {
			return questions
		}

		text := fmt.Sprintf("Check whether this regression is intentional: %s", item.Title)
		if item.ID != "" {
			text += fmt.Sprintf(" `%s`", item.ID)
		}
		if item.Detail != "" {
			text += " — " + item.Detail
		}

		questions = append(questions, ReviewQuestion{Text: text})
	}

	for _, change := range result.ContractChanges {
		if len(questions) >= limit {
			return questions
		}

		switch change.Kind {
		case contracts.ChangeKindRemoved, contracts.ChangeKindSignatureChanged, contracts.ChangeKindModifiersChanged:
			if hasRelatedChangedTest(result.ChangedFiles, change) {
				questions = append(questions, ReviewQuestion{
					Text: fmt.Sprintf("Public contract changed `%s`; test-like files changed in this patch. Verify they actually cover this contract migration.", change.SymbolKey),
				})
			} else {
				questions = append(questions, ReviewQuestion{
					Text: fmt.Sprintf("Public contract changed `%s`, but no test-like files changed. Verify callers and add or update tests.", change.SymbolKey),
				})
			}
		}
	}

	if len(questions) == 0 {
		questions = append(questions, ReviewQuestion{
			Text: "No specific high-signal questions generated from the current facts.",
		})
	}

	return questions
}

func hasRelatedChangedTest(changedFiles []string, change contracts.SymbolChange) bool {
	candidates := contractFiles(change)
	if len(candidates) == 0 {
		return anyTestLikeFileChanged(changedFiles)
	}

	for _, changedFile := range changedFiles {
		if !isTestLikeFile(changedFile) {
			continue
		}

		changedBase := normalizedBaseName(changedFile)
		for _, candidate := range candidates {
			candidateBase := normalizedBaseName(candidate)
			if changedBase == candidateBase || strings.Contains(changedBase, candidateBase) || strings.Contains(candidateBase, changedBase) {
				return true
			}
		}
	}

	return false
}

func contractFiles(change contracts.SymbolChange) []string {
	seen := make(map[string]struct{})
	files := make([]string, 0, 2)

	add := func(file string) {
		if file == "" {
			return
		}
		if _, ok := seen[file]; ok {
			return
		}
		seen[file] = struct{}{}
		files = append(files, file)
	}

	if change.Before != nil {
		add(change.Before.File)
	}
	if change.After != nil {
		add(change.After.File)
	}

	return files
}

func anyTestLikeFileChanged(changedFiles []string) bool {
	for _, file := range changedFiles {
		if isTestLikeFile(file) {
			return true
		}
	}

	return false
}

func isTestLikeFile(file string) bool {
	normalized := strings.ToLower(strings.ReplaceAll(file, "\\", "/"))
	base := filepath.Base(normalized)

	if strings.Contains(normalized, "/test/") ||
		strings.Contains(normalized, "/tests/") ||
		strings.Contains(normalized, "/mocks/") ||
		strings.Contains(normalized, "/mock/") {
		return true
	}

	return strings.HasSuffix(base, "_test.go") ||
		strings.HasSuffix(base, "_test.cc") ||
		strings.HasSuffix(base, "_test.cpp") ||
		strings.HasSuffix(base, "_test.cxx") ||
		strings.HasSuffix(base, "_test.h") ||
		strings.HasSuffix(base, "_test.hpp")
}

func normalizedBaseName(file string) string {
	base := strings.ToLower(filepath.Base(strings.ReplaceAll(file, "\\", "/")))
	ext := filepath.Ext(base)
	base = strings.TrimSuffix(base, ext)
	base = strings.TrimSuffix(base, "_test")
	base = strings.TrimPrefix(base, "test_")
	base = strings.TrimPrefix(base, "mock_")
	base = strings.TrimSuffix(base, "_mock")
	return base
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
