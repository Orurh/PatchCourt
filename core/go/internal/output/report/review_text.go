package report

import (
	"fmt"
	"io"
	"strings"

	"github.com/orurh/patchcourt/internal/analysis/contracts"
	"github.com/orurh/patchcourt/internal/analysis/depdiff"
	"github.com/orurh/patchcourt/internal/analysis/findingdiff"
	"github.com/orurh/patchcourt/internal/analysis/risk"
	"github.com/orurh/patchcourt/internal/app"
	"github.com/orurh/patchcourt/internal/model"
)

func WriteReviewText(w io.Writer, result ReviewTextResult) {
	fmt.Fprintln(w, "PatchCourt review")
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
	Summary           app.ReviewSummary
	Risk              risk.Score
	Impact            app.ReviewImpactReport
	ContractChanges   []contracts.SymbolChange
	DependencyChanges []depdiff.DependencyChange
	LayerEdgeChanges  []depdiff.LayerEdgeChange
	FindingChanges    []findingdiff.FindingChange
}

func writeReviewSummaryText(w io.Writer, summary app.ReviewSummary) {
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

func writeReviewImpactText(w io.Writer, impact app.ReviewImpactReport) {
	fmt.Fprintln(w, "Architecture impact:")
	writeReviewImpactSectionText(w, "  Worse:", impact.Worse)
	writeReviewImpactSectionText(w, "  Better:", impact.Better)
	writeReviewImpactSectionText(w, "  Unchanged debt:", impact.UnchangedDebt)
}

func writeReviewImpactSectionText(w io.Writer, title string, items []app.ReviewImpactItem) {
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
