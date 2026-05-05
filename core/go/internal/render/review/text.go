package review

import (
	"fmt"
	"io"
	"strings"

	"github.com/orurh/patchcourt/internal/diff/contract"
	"github.com/orurh/patchcourt/internal/diff/dep"
	"github.com/orurh/patchcourt/internal/diff/finding"
	"github.com/orurh/patchcourt/internal/model"
	"github.com/orurh/patchcourt/internal/render/reviewcontract"
	"github.com/orurh/patchcourt/internal/reportmodel"
	"github.com/orurh/patchcourt/internal/reviewrisk"
)

func WriteReviewText(w io.Writer, result ReviewTextResult) {
	fmt.Fprintln(w, "PatchCourt review")
	fmt.Fprintln(w)

	writeReviewVerdictText(w, result)
	fmt.Fprintln(w)

	writeReviewSummaryText(w, result.Summary)
	fmt.Fprintln(w)

	writeRiskText(w, result.Risk)
	fmt.Fprintln(w)

	writeReviewImpactText(w, result.Impact)
	fmt.Fprintln(w)

	writeFindingChangesText(w, result.FindingChanges)
	fmt.Fprintln(w)

	writeDependencyChangesText(w, result.DependencyChanges)
	fmt.Fprintln(w)

	writeLayerEdgeChangesText(w, result.LayerEdgeChanges)
	fmt.Fprintln(w)

	writeContractChangesText(w, result.ContractChanges)
}

type ReviewTextResult struct {
	Summary           reportmodel.ReviewSummary
	Risk              risk.Score
	Impact            reportmodel.ReviewImpactReport
	ContractChanges   []contracts.SymbolChange
	DependencyChanges []depdiff.DependencyChange
	LayerEdgeChanges  []depdiff.LayerEdgeChange
	FindingChanges    []findingdiff.FindingChange
}

func writeReviewVerdictText(w io.Writer, result ReviewTextResult) {
	fmt.Fprintln(w, "Verdict:")
	fmt.Fprintf(w, "  architecture: %s\n", architectureVerdict(result.Impact))
	fmt.Fprintf(w, "  risk:         %s\n", result.Risk.Level)

	writeVerdictItems(w, "  main concerns:", verdictItems(result.Impact.Worse), 3)
	writeVerdictItems(w, "  improvements:", verdictItems(result.Impact.Better), 3)
}

func architectureVerdict(impact reportmodel.ReviewImpactReport) string {
	hasWorse := len(impact.Worse) > 0
	hasBetter := len(impact.Better) > 0

	switch {
	case hasWorse && hasBetter:
		return "mixed"
	case hasWorse:
		return "worsened"
	case hasBetter:
		return "improved"
	default:
		return "unchanged"
	}
}

func verdictItems(items []reportmodel.ReviewImpactItem) []reportmodel.ReviewImpactItem {
	highSignal := make([]reportmodel.ReviewImpactItem, 0, len(items))

	for _, item := range items {
		if isHighSignalVerdictItem(item) {
			highSignal = append(highSignal, item)
		}
	}

	if len(highSignal) > 0 {
		return highSignal
	}

	return items
}

func isHighSignalVerdictItem(item reportmodel.ReviewImpactItem) bool {
	switch item.Kind {
	case "finding_added",
		"finding_removed",
		"layer_edge_added",
		"layer_edge_removed",
		"layer_edge_increased",
		"layer_edge_decreased",
		"contract_removed",
		"contract_signature_changed",
		"contract_modifiers_changed":
		return true
	default:
		return false
	}
}

func writeVerdictItems(w io.Writer, title string, items []reportmodel.ReviewImpactItem, limit int) {
	fmt.Fprintln(w, title)

	if len(items) == 0 {
		fmt.Fprintln(w, "    none")
		return
	}

	if limit <= 0 || limit > len(items) {
		limit = len(items)
	}

	for i := 0; i < limit; i++ {
		fmt.Fprintf(w, "    - %s\n", verdictItemText(items[i]))
	}

	if hidden := len(items) - limit; hidden > 0 {
		fmt.Fprintf(w, "    ... %d more\n", hidden)
	}
}

func verdictItemText(item reportmodel.ReviewImpactItem) string {
	switch {
	case item.ID != "" && item.Detail != "":
		return fmt.Sprintf("%s: %s — %s", item.Title, item.ID, item.Detail)
	case item.ID != "":
		return fmt.Sprintf("%s: %s", item.Title, item.ID)
	case item.Detail != "":
		return fmt.Sprintf("%s — %s", item.Title, item.Detail)
	default:
		return item.Title
	}
}

func writeReviewSummaryText(w io.Writer, summary reportmodel.ReviewSummary) {
	fmt.Fprintln(w, "Summary:")
	fmt.Fprintf(w, "  contract changes:      %d\n", summary.ContractChanges)
	fmt.Fprintf(w, "  dependency changes:    %d\n", summary.DependencyChanges)
	fmt.Fprintf(w, "  layer edge changes:    %d\n", summary.LayerEdgeChanges)
	fmt.Fprintf(w, "  finding changes:       %d\n", summary.FindingChanges)
	fmt.Fprintf(w, "  added findings:        %d\n", summary.AddedFindings)
	fmt.Fprintf(w, "  removed findings:      %d\n", summary.RemovedFindings)
	fmt.Fprintf(w, "  added high findings:   %d\n", summary.AddedHighFindings)
	fmt.Fprintf(w, "  added policy findings: %d\n", summary.AddedPolicyFindings)
}

func writeRiskText(w io.Writer, score risk.Score) {
	fmt.Fprintln(w, "Risk:")
	fmt.Fprintf(w, "  level:  %s\n", score.Level)
	fmt.Fprintf(w, "  points: %d\n", score.Points)

	if len(score.Reasons) == 0 {
		return
	}

	fmt.Fprintln(w, "  reasons:")
	for _, reason := range score.Reasons {
		fmt.Fprintf(w, "    - +%d %s\n", reason.Points, reason.Message)
	}
}

func writeFindingChangesText(w io.Writer, changes []findingdiff.FindingChange) {
	fmt.Fprintln(w, "Finding changes:")
	fmt.Fprintf(w, "  total: %d\n", len(changes))

	for _, change := range changes {
		finding := change.After
		if finding == nil {
			finding = change.Before
		}

		if finding == nil {
			continue
		}

		fmt.Fprintln(w)
		fmt.Fprintf(w, "  [%s] %s\n", change.Kind, change.ID)
		fmt.Fprintf(w, "    severity: %s\n", finding.Severity)

		if finding.Kind != "" {
			fmt.Fprintf(w, "    kind:     %s\n", finding.Kind)
		}

		if finding.Title != "" {
			fmt.Fprintf(w, "    title:    %s\n", finding.Title)
		}

		if finding.Risk != "" {
			fmt.Fprintf(w, "    risk:     %s\n", finding.Risk)
		}

		if finding.Suggestion != "" {
			fmt.Fprintf(w, "    suggest:  %s\n", finding.Suggestion)
		}

		writeFindingEvidenceText(w, finding.Evidence)
	}
}

func writeFindingEvidenceText(w io.Writer, evidence []model.Evidence) {
	if len(evidence) == 0 {
		return
	}

	limit := len(evidence)
	if limit > 5 {
		limit = 5
	}

	fmt.Fprintln(w, "    evidence:")
	for i := 0; i < limit; i++ {
		item := evidence[i]
		if item.File != "" {
			fmt.Fprintf(w, "      - %s: %s\n", item.File, item.Message)
		} else {
			fmt.Fprintf(w, "      - %s\n", item.Message)
		}
	}

	if len(evidence) > limit {
		fmt.Fprintf(w, "      ... %d more\n", len(evidence)-limit)
	}
}

func writeDependencyChangesText(w io.Writer, changes []depdiff.DependencyChange) {
	relevant := reviewRelevantDependencyChanges(changes)
	hidden := len(changes) - len(relevant)

	fmt.Fprintln(w, "Dependency changes:")
	fmt.Fprintf(w, "  review-relevant: %d\n", len(relevant))
	fmt.Fprintf(w, "  raw total:       %d\n", len(changes))

	if hidden > 0 {
		fmt.Fprintf(w, "  hidden low-level: %d\n", hidden)
	}

	for _, change := range relevant {
		dep := change.After
		if dep == nil {
			dep = change.Before
		}

		if dep == nil {
			continue
		}

		target := dep.ToFile
		if target == "" {
			target = dep.Target
		}

		fmt.Fprintln(w)
		fmt.Fprintf(w, "  [%s] %s\n", change.Kind, change.Key)
		fmt.Fprintf(w, "    from: %s\n", dep.FromFile)
		fmt.Fprintf(w, "    to:   %s\n", target)

		if dep.FromLayer != "" || dep.ToLayer != "" {
			fmt.Fprintf(w, "    layer: %s -> %s\n", dep.FromLayer, dep.ToLayer)
		}

		if dep.Usage != "" {
			fmt.Fprintf(w, "    usage: %s\n", dep.Usage)
		}
	}
}

func reviewRelevantDependencyChanges(changes []depdiff.DependencyChange) []depdiff.DependencyChange {
	relevant := make([]depdiff.DependencyChange, 0, len(changes))

	for _, change := range changes {
		dep := change.After
		if dep == nil {
			dep = change.Before
		}

		if dep == nil {
			continue
		}

		if !isReviewRelevantDependency(*dep) {
			continue
		}

		relevant = append(relevant, change)
	}

	return relevant
}

func isReviewRelevantDependency(dep model.DependencyEdge) bool {
	if dep.External {
		return false
	}

	if dep.FromLayer == "" || dep.ToLayer == "" {
		return false
	}

	if dep.FromLayer == dep.ToLayer {
		return false
	}

	return true
}

func writeLayerEdgeChangesText(w io.Writer, changes []depdiff.LayerEdgeChange) {
	fmt.Fprintln(w, "Layer edge changes:")
	fmt.Fprintf(w, "  total: %d\n", len(changes))

	for _, change := range changes {
		fmt.Fprintln(w)
		fmt.Fprintf(w, "  [%s] %s -> %s\n", change.Kind, change.FromLayer, change.ToLayer)

		if change.BeforeCount > 0 {
			fmt.Fprintf(w, "    before count: %d\n", change.BeforeCount)
		}

		if change.AfterCount > 0 {
			fmt.Fprintf(w, "    after count:  %d\n", change.AfterCount)
		}
	}
}

func writeContractChangesText(w io.Writer, changes []contracts.SymbolChange) {
	fmt.Fprintln(w, "Contract changes:")
	fmt.Fprintf(w, "  total: %d\n", len(changes))

	for _, change := range changes {
		fmt.Fprintln(w)
		fmt.Fprintf(w, "  [%s] %s\n", change.Kind, change.SymbolKey)
		fmt.Fprintf(w, "    impact: %s\n", reviewcontract.ClassifyImpact(change))

		if location := reviewcontract.Location(change); location != "" {
			fmt.Fprintf(w, "    location: %s\n", location)
		}

		if change.Before != nil && change.Before.Signature != "" {
			fmt.Fprintf(w, "    before: %s\n", change.Before.Signature)
		}

		if change.After != nil && change.After.Signature != "" {
			fmt.Fprintf(w, "    after:  %s\n", change.After.Signature)
		}

		if len(change.AddedMods) > 0 {
			fmt.Fprintf(w, "    added modifiers: %s\n", strings.Join(change.AddedMods, ", "))
		}

		if len(change.RemovedMods) > 0 {
			fmt.Fprintf(w, "    removed modifiers: %s\n", strings.Join(change.RemovedMods, ", "))
		}
	}
}

func writeReviewImpactText(w io.Writer, impact reportmodel.ReviewImpactReport) {
	fmt.Fprintln(w, "Architecture impact:")
	writeReviewImpactSectionText(w, "  Worse:", impact.Worse)
	writeReviewImpactSectionText(w, "  Better:", impact.Better)
	writeReviewImpactSectionText(w, "  Unchanged debt:", impact.UnchangedDebt)
}

func writeReviewImpactSectionText(w io.Writer, title string, items []reportmodel.ReviewImpactItem) {
	fmt.Fprintln(w, title)

	if len(items) == 0 {
		fmt.Fprintln(w, "    none")
		return
	}

	for _, item := range items {
		if item.ID != "" && item.Detail != "" {
			fmt.Fprintf(w, "    - [%s] %s: %s — %s\n", item.Kind, item.Title, item.ID, item.Detail)
			continue
		}

		if item.ID != "" {
			fmt.Fprintf(w, "    - [%s] %s: %s\n", item.Kind, item.Title, item.ID)
			continue
		}

		if item.Detail != "" {
			fmt.Fprintf(w, "    - [%s] %s — %s\n", item.Kind, item.Title, item.Detail)
			continue
		}

		fmt.Fprintf(w, "    - [%s] %s\n", item.Kind, item.Title)
	}
}
