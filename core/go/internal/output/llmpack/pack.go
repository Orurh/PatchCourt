package llmpack

import (
	"bytes"
	"fmt"
	"io"

	"github.com/orurh/patchcourt/internal/analysis/contracts"
	"github.com/orurh/patchcourt/internal/analysis/depdiff"
	"github.com/orurh/patchcourt/internal/analysis/findingdiff"
	"github.com/orurh/patchcourt/internal/model"
	"github.com/orurh/patchcourt/internal/platform/files"
	"github.com/orurh/patchcourt/internal/reportmodel"
)

const DefaultMaxItems = 10

type ReviewContextInput struct {
	Result   reportmodel.ReviewResult
	MaxItems int
}

func WriteReviewContextFile(path string, input ReviewContextInput) error {
	var buf bytes.Buffer

	WriteReviewContext(&buf, input)

	if err := files.WriteFileAtomic(path, buf.Bytes(), 0o644); err != nil {
		return fmt.Errorf("write LLM context pack %s: %w", path, err)
	}

	return nil
}

func WriteReviewContext(w io.Writer, input ReviewContextInput) {
	limit := input.MaxItems
	if limit <= 0 {
		limit = DefaultMaxItems
	}

	result := input.Result

	fmt.Fprintln(w, "# PatchCourt Review Context")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "This context pack is generated from deterministic PatchCourt facts.")
	fmt.Fprintln(w, "Use it to review the patch. Do not invent files, dependencies, symbols, or findings not listed here.")
	fmt.Fprintln(w)

	writeSummary(w, result)
	fmt.Fprintln(w)

	writeArchitectureImpact(w, result.Impact, limit)
	fmt.Fprintln(w)

	writeContractChanges(w, result.ContractChanges, limit)
	fmt.Fprintln(w)

	writeDependencyChanges(w, result.DependencyChanges, limit)
	fmt.Fprintln(w)

	writeLayerEdgeChanges(w, result.LayerEdgeChanges, limit)
	fmt.Fprintln(w)

	writeFindingChanges(w, result.FindingChanges, limit)
	fmt.Fprintln(w)

	writeReviewQuestions(w, result, limit)
}

func writeSummary(w io.Writer, result reportmodel.ReviewResult) {
	fmt.Fprintln(w, "## Summary")
	fmt.Fprintln(w)
	fmt.Fprintf(w, "- Schema: `%s`\n", result.SchemaVersion)
	fmt.Fprintf(w, "- Risk: `%s`, %d points\n", result.Risk.Level, result.Risk.Points)
	fmt.Fprintf(w, "- Contract changes: %d\n", result.Summary.ContractChanges)
	fmt.Fprintf(w, "- Dependency changes: %d\n", result.Summary.DependencyChanges)
	fmt.Fprintf(w, "- Layer edge changes: %d\n", result.Summary.LayerEdgeChanges)
	fmt.Fprintf(w, "- Finding changes: %d\n", result.Summary.FindingChanges)
	fmt.Fprintf(w, "- Added findings: %d\n", result.Summary.AddedFindings)
	fmt.Fprintf(w, "- Removed findings: %d\n", result.Summary.RemovedFindings)

	if len(result.Risk.Reasons) == 0 {
		return
	}

	fmt.Fprintln(w)
	fmt.Fprintln(w, "### Risk reasons")
	fmt.Fprintln(w)

	for _, reason := range result.Risk.Reasons {
		fmt.Fprintf(w, "- +%d %s\n", reason.Points, reason.Message)
	}
}

func writeArchitectureImpact(w io.Writer, impact reportmodel.ReviewImpactReport, limit int) {
	fmt.Fprintln(w, "## Architecture impact")
	fmt.Fprintln(w)

	writeImpactItems(w, "### Worse", impact.Worse, limit)
	fmt.Fprintln(w)

	writeImpactItems(w, "### Better", impact.Better, limit)
	fmt.Fprintln(w)

	writeImpactItems(w, "### Unchanged debt", impact.UnchangedDebt, limit)
}

func writeImpactItems(w io.Writer, title string, items []reportmodel.ReviewImpactItem, limit int) {
	fmt.Fprintln(w, title)
	fmt.Fprintln(w)

	if len(items) == 0 {
		fmt.Fprintln(w, "- none")
		return
	}

	for i, item := range limited(items, limit) {
		fmt.Fprintf(w, "- `%s` %s", item.Kind, item.Title)

		if item.ID != "" {
			fmt.Fprintf(w, ": `%s`", item.ID)
		}

		if item.Detail != "" {
			fmt.Fprintf(w, " — %s", item.Detail)
		}

		if item.Severity != "" {
			fmt.Fprintf(w, " _(severity: %s)_", item.Severity)
		}

		fmt.Fprintln(w)

		if i == limit-1 && len(items) > limit {
			fmt.Fprintf(w, "- ... %d more\n", len(items)-limit)
			return
		}
	}
}

func writeContractChanges(w io.Writer, changes []contracts.SymbolChange, limit int) {
	fmt.Fprintln(w, "## Contract changes")
	fmt.Fprintln(w)

	if len(changes) == 0 {
		fmt.Fprintln(w, "- none")
		return
	}

	for _, change := range limited(changes, limit) {
		fmt.Fprintf(w, "- `%s` `%s`\n", change.Kind, change.SymbolKey)

		if change.Before != nil && change.Before.Signature != "" {
			fmt.Fprintf(w, "  - before: `%s`\n", change.Before.Signature)
		}

		if change.After != nil && change.After.Signature != "" {
			fmt.Fprintf(w, "  - after: `%s`\n", change.After.Signature)
		}
	}

	writeMore(w, len(changes), limit)
}

func writeDependencyChanges(w io.Writer, changes []depdiff.DependencyChange, limit int) {
	fmt.Fprintln(w, "## Dependency changes")
	fmt.Fprintln(w)

	if len(changes) == 0 {
		fmt.Fprintln(w, "- none")
		return
	}

	for _, change := range limited(changes, limit) {
		fmt.Fprintf(w, "- `%s` `%s`\n", change.Kind, change.Key)

		if change.Before != nil {
			fmt.Fprintf(w, "  - before: `%s -> %s` `%s -> %s`\n",
				change.Before.FromFile,
				dependencyTarget(change.Before),
				change.Before.FromLayer,
				change.Before.ToLayer,
			)
		}

		if change.After != nil {
			fmt.Fprintf(w, "  - after: `%s -> %s` `%s -> %s`\n",
				change.After.FromFile,
				dependencyTarget(change.After),
				change.After.FromLayer,
				change.After.ToLayer,
			)
		}
	}

	writeMore(w, len(changes), limit)
}

func writeLayerEdgeChanges(w io.Writer, changes []depdiff.LayerEdgeChange, limit int) {
	fmt.Fprintln(w, "## Layer edge changes")
	fmt.Fprintln(w)

	if len(changes) == 0 {
		fmt.Fprintln(w, "- none")
		return
	}

	for _, change := range limited(changes, limit) {
		fmt.Fprintf(
			w,
			"- `%s` `%s -> %s` `%d -> %d`\n",
			change.Kind,
			change.FromLayer,
			change.ToLayer,
			change.BeforeCount,
			change.AfterCount,
		)
	}

	writeMore(w, len(changes), limit)
}

func writeFindingChanges(w io.Writer, changes []findingdiff.FindingChange, limit int) {
	fmt.Fprintln(w, "## Finding changes")
	fmt.Fprintln(w)

	if len(changes) == 0 {
		fmt.Fprintln(w, "- none")
		return
	}

	for _, change := range limited(changes, limit) {
		fmt.Fprintf(w, "- `%s` `%s`\n", change.Kind, change.ID)

		if change.Before != nil {
			fmt.Fprintf(w, "  - before: `%s/%s` %s\n", change.Before.Severity, change.Before.Kind, change.Before.Title)
		}

		if change.After != nil {
			fmt.Fprintf(w, "  - after: `%s/%s` %s\n", change.After.Severity, change.After.Kind, change.After.Title)
		}
	}

	writeMore(w, len(changes), limit)
}

func writeReviewQuestions(w io.Writer, result reportmodel.ReviewResult, limit int) {
	fmt.Fprintln(w, "## Review questions")
	fmt.Fprintln(w)

	if len(result.Impact.Worse) == 0 && len(result.ContractChanges) == 0 {
		fmt.Fprintln(w, "- No specific high-signal questions generated from the current facts.")
		return
	}

	count := 0

	for _, item := range result.Impact.Worse {
		if count >= limit {
			writeMore(w, len(result.Impact.Worse), limit)
			return
		}

		fmt.Fprintf(w, "- Check whether this regression is intentional: %s", item.Title)
		if item.ID != "" {
			fmt.Fprintf(w, " `%s`", item.ID)
		}
		if item.Detail != "" {
			fmt.Fprintf(w, " — %s", item.Detail)
		}
		fmt.Fprintln(w)
		count++
	}

	for _, change := range result.ContractChanges {
		if count >= limit {
			return
		}

		switch change.Kind {
		case contracts.ChangeKindRemoved, contracts.ChangeKindSignatureChanged, contracts.ChangeKindModifiersChanged:
			fmt.Fprintf(w, "- Verify callers and tests for public contract change `%s`.\n", change.SymbolKey)
			count++
		}
	}

	if count == 0 {
		fmt.Fprintln(w, "- No specific high-signal questions generated from the current facts.")
	}
}

func dependencyTarget(dep *model.DependencyEdge) string {
	if dep == nil {
		return ""
	}

	if dep.ToFile != "" {
		return dep.ToFile
	}

	return dep.Target
}

func writeMore(w io.Writer, total int, limit int) {
	if total > limit {
		fmt.Fprintf(w, "- ... %d more\n", total-limit)
	}
}

func limited[T any](values []T, limit int) []T {
	if limit <= 0 || len(values) <= limit {
		return values
	}

	return values[:limit]
}
