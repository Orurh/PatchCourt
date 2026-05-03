package report

import (
	"fmt"
	"io"
	"strings"

	"github.com/orurh/patchcourt/internal/analysis/contracts"
	"github.com/orurh/patchcourt/internal/analysis/depdiff"
	"github.com/orurh/patchcourt/internal/analysis/findingdiff"
	"github.com/orurh/patchcourt/internal/analysis/risk"
	"github.com/orurh/patchcourt/internal/model"
	"github.com/orurh/patchcourt/internal/reportmodel"
)

type ReviewMarkdownResult struct {
	Summary           reportmodel.ReviewSummary
	Risk              risk.Score
	Impact            reportmodel.ReviewImpactReport
	ContractChanges   []contracts.SymbolChange
	DependencyChanges []depdiff.DependencyChange
	LayerEdgeChanges  []depdiff.LayerEdgeChange
	FindingChanges    []findingdiff.FindingChange
	AfterRoot         string
	ConfigPath        string
}

func WriteReviewMarkdown(w io.Writer, result ReviewMarkdownResult) {
	fmt.Fprintln(w, "# PatchCourt Review")
	fmt.Fprintln(w)

	writeMarkdownSummary(w, result.Summary, result.Risk)
	fmt.Fprintln(w)

	writeMarkdownRiskReasons(w, result.Risk)
	fmt.Fprintln(w)

	writeMarkdownImpact(w, result.Impact)
	fmt.Fprintln(w)

	writeMarkdownFindingChanges(w, result.FindingChanges, result.AfterRoot, result.ConfigPath)
	fmt.Fprintln(w)

	writeMarkdownLayerEdgeChanges(w, result.LayerEdgeChanges)
	fmt.Fprintln(w)

	writeMarkdownDependencyChanges(w, result.DependencyChanges)
	fmt.Fprintln(w)

	writeMarkdownContractChanges(w, result.ContractChanges)
}

func writeMarkdownSummary(w io.Writer, summary reportmodel.ReviewSummary, score risk.Score) {
	fmt.Fprintln(w, "## Summary")
	fmt.Fprintln(w)
	fmt.Fprintf(w, "- **Risk:** `%s`, **%d** points\n", score.Level, score.Points)
	fmt.Fprintf(w, "- **Contract changes:** %d\n", summary.ContractChanges)
	fmt.Fprintf(w, "- **Dependency changes:** %d\n", summary.DependencyChanges)
	fmt.Fprintf(w, "- **Layer edge changes:** %d\n", summary.LayerEdgeChanges)
	fmt.Fprintf(w, "- **Finding changes:** %d\n", summary.FindingChanges)
	fmt.Fprintf(w, "- **Added findings:** %d\n", summary.AddedFindings)
	fmt.Fprintf(w, "- **Removed findings:** %d\n", summary.RemovedFindings)
	fmt.Fprintf(w, "- **Added high findings:** %d\n", summary.AddedHighFindings)
	fmt.Fprintf(w, "- **Added policy findings:** %d\n", summary.AddedPolicyFindings)
}

func writeMarkdownRiskReasons(w io.Writer, score risk.Score) {
	fmt.Fprintln(w, "## Risk reasons")
	fmt.Fprintln(w)

	if len(score.Reasons) == 0 {
		fmt.Fprintln(w, "_No risk reasons._")
		return
	}

	for _, reason := range score.Reasons {
		fmt.Fprintf(w, "- **+%d** %s\n", reason.Points, markdownTextEscape(reason.Message))
	}
}

func writeMarkdownFindingChanges(w io.Writer, changes []findingdiff.FindingChange, afterRoot string, configPath string) {
	fmt.Fprintln(w, "## Finding changes")
	fmt.Fprintln(w)

	if len(changes) == 0 {
		fmt.Fprintln(w, "_No finding changes._")
		return
	}

	for _, change := range changes {
		finding := change.After
		if finding == nil {
			finding = change.Before
		}

		if finding == nil {
			continue
		}

		fmt.Fprintf(w, "### `%s` `%s`\n", markdownTextEscape(change.ID), change.Kind)
		fmt.Fprintln(w)

		if finding.Title != "" {
			fmt.Fprintf(w, "**Title:** %s  \n", markdownTextEscape(finding.Title))
		}

		if finding.Severity != "" {
			fmt.Fprintf(w, "**Severity:** `%s`  \n", finding.Severity)
		}

		if finding.Kind != "" {
			fmt.Fprintf(w, "**Kind:** `%s`  \n", finding.Kind)
		}

		if finding.Confidence != "" {
			fmt.Fprintf(w, "**Confidence:** `%s`  \n", finding.Confidence)
		}

		if finding.Risk != "" {
			fmt.Fprintln(w)
			fmt.Fprintln(w, "**Risk:**")
			fmt.Fprintln(w)
			fmt.Fprintf(w, "> %s\n", markdownTextEscape(finding.Risk))
		}

		if finding.Suggestion != "" {
			fmt.Fprintln(w)
			fmt.Fprintln(w, "**Suggestion:**")
			fmt.Fprintln(w)
			fmt.Fprintf(w, "> %s\n", markdownTextEscape(finding.Suggestion))
		}

		writeMarkdownEvidence(w, finding.Evidence)

		if change.Kind == findingdiff.FindingChangeKindAdded {
			writeMarkdownExplainCommand(w, finding.ID, afterRoot, configPath)
		}

		fmt.Fprintln(w)
	}
}

func writeMarkdownEvidence(w io.Writer, evidence []model.Evidence) {
	if len(evidence) == 0 {
		return
	}

	fmt.Fprintln(w)
	fmt.Fprintln(w, "**Evidence:**")
	fmt.Fprintln(w)

	limit := len(evidence)
	if limit > 10 {
		limit = 10
	}

	for i := 0; i < limit; i++ {
		item := evidence[i]

		if item.File != "" {
			fmt.Fprintf(w, "- `%s`: %s\n", markdownTextEscape(item.File), markdownTextEscape(item.Message))
		} else {
			fmt.Fprintf(w, "- %s\n", markdownTextEscape(item.Message))
		}

		if item.LineStart > 0 {
			if item.LineEnd > item.LineStart {
				fmt.Fprintf(w, "  - lines: `%d-%d`\n", item.LineStart, item.LineEnd)
			} else {
				fmt.Fprintf(w, "  - line: `%d`\n", item.LineStart)
			}
		}

		if item.Snippet != "" {
			fmt.Fprintln(w)
			fmt.Fprintln(w, "  ```")
			fmt.Fprintf(w, "  %s\n", item.Snippet)
			fmt.Fprintln(w, "  ```")
		}
	}

	if len(evidence) > limit {
		fmt.Fprintf(w, "- _... %d more evidence item(s)_\n", len(evidence)-limit)
	}
}

func writeMarkdownExplainCommand(w io.Writer, findingID string, afterRoot string, configPath string) {
	if findingID == "" {
		return
	}

	fmt.Fprintln(w)
	fmt.Fprintln(w, "**Explain:**")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "```bash")

	if afterRoot != "" {
		fmt.Fprintf(w, "patchcourt explain %s --root %s", shellQuote(findingID), shellQuote(afterRoot))

		if configPath != "" {
			fmt.Fprintf(w, " --config %s", shellQuote(configPath))
		}

		fmt.Fprintln(w)
	} else {
		fmt.Fprintf(w, "patchcourt explain %s --model <project-model.json>\n", shellQuote(findingID))
	}

	fmt.Fprintln(w, "```")
}

func writeMarkdownLayerEdgeChanges(w io.Writer, changes []depdiff.LayerEdgeChange) {
	fmt.Fprintln(w, "## Layer edge changes")
	fmt.Fprintln(w)

	if len(changes) == 0 {
		fmt.Fprintln(w, "_No layer edge changes._")
		return
	}

	fmt.Fprintln(w, "| Kind | Edge | Before | After |")
	fmt.Fprintln(w, "| --- | --- | ---: | ---: |")

	for _, change := range changes {
		fmt.Fprintf(
			w,
			"| `%s` | `%s -> %s` | %d | %d |\n",
			change.Kind,
			markdownTableEscape(change.FromLayer),
			markdownTableEscape(change.ToLayer),
			change.BeforeCount,
			change.AfterCount,
		)
	}
}

func writeMarkdownDependencyChanges(w io.Writer, changes []depdiff.DependencyChange) {
	fmt.Fprintln(w, "## Dependency changes")
	fmt.Fprintln(w)

	if len(changes) == 0 {
		fmt.Fprintln(w, "_No dependency changes._")
		return
	}

	limit := len(changes)
	if limit > 20 {
		limit = 20
	}

	fmt.Fprintln(w, "| Kind | From | To | Layer | Usage |")
	fmt.Fprintln(w, "| --- | --- | --- | --- | --- |")

	for i := 0; i < limit; i++ {
		change := changes[i]

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

		layer := ""
		if dep.FromLayer != "" || dep.ToLayer != "" {
			layer = dep.FromLayer + " -> " + dep.ToLayer
		}

		fmt.Fprintf(
			w,
			"| `%s` | `%s` | `%s` | `%s` | `%s` |\n",
			change.Kind,
			markdownTableEscape(dep.FromFile),
			markdownTableEscape(to),
			markdownTableEscape(layer),
			dep.Usage,
		)
	}

	if len(changes) > limit {
		fmt.Fprintf(w, "\n_... %d more dependency change(s)_\n", len(changes)-limit)
	}
}

func writeMarkdownContractChanges(w io.Writer, changes []contracts.SymbolChange) {
	fmt.Fprintln(w, "## Contract changes")
	fmt.Fprintln(w)

	if len(changes) == 0 {
		fmt.Fprintln(w, "_No contract changes._")
		return
	}

	for _, change := range changes {
		fmt.Fprintf(w, "### `%s` `%s`\n", markdownTextEscape(change.SymbolKey), change.Kind)
		fmt.Fprintln(w)

		if change.Before != nil && change.Before.Signature != "" {
			fmt.Fprintln(w, "**Before:**")
			fmt.Fprintln(w)
			fmt.Fprintln(w, "```cpp")
			fmt.Fprintln(w, change.Before.Signature)
			fmt.Fprintln(w, "```")
		}

		if change.After != nil && change.After.Signature != "" {
			fmt.Fprintln(w, "**After:**")
			fmt.Fprintln(w)
			fmt.Fprintln(w, "```cpp")
			fmt.Fprintln(w, change.After.Signature)
			fmt.Fprintln(w, "```")
		}

		if len(change.AddedMods) > 0 {
			fmt.Fprintf(w, "- Added modifiers: `%s`\n", markdownTextEscape(strings.Join(change.AddedMods, "`, `")))
		}

		if len(change.RemovedMods) > 0 {
			fmt.Fprintf(w, "- Removed modifiers: `%s`\n", markdownTextEscape(strings.Join(change.RemovedMods, "`, `")))
		}

		fmt.Fprintln(w)
	}
}

func markdownTextEscape(value string) string {
	return value
}

func markdownTableEscape(value string) string {
	return strings.ReplaceAll(value, "|", "\\|")
}

func shellQuote(value string) string {
	if value == "" {
		return "''"
	}

	if strings.IndexFunc(value, func(r rune) bool {
		return r == ' ' || r == '\t' || r == '\n' || r == '\'' || r == '"' || r == '\\' || r == '$' || r == '`'
	}) == -1 {
		return value
	}

	return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'"
}

func writeMarkdownImpact(w io.Writer, impact reportmodel.ReviewImpactReport) {
	fmt.Fprintln(w, "## Architecture impact")
	fmt.Fprintln(w)

	writeMarkdownImpactSection(w, "### Worse", impact.Worse)
	fmt.Fprintln(w)

	writeMarkdownImpactSection(w, "### Better", impact.Better)
	fmt.Fprintln(w)

	writeMarkdownImpactSection(w, "### Unchanged debt", impact.UnchangedDebt)
}

func writeMarkdownImpactSection(w io.Writer, title string, items []reportmodel.ReviewImpactItem) {
	fmt.Fprintln(w, title)
	fmt.Fprintln(w)

	if len(items) == 0 {
		fmt.Fprintln(w, "_None._")
		return
	}

	for _, item := range items {
		label := item.Title
		if item.Severity != "" {
			label = fmt.Sprintf("%s `%s`", label, item.Severity)
		}

		if item.ID != "" {
			fmt.Fprintf(w, "- **%s:** `%s`", markdownTextEscape(label), markdownTextEscape(item.ID))
		} else {
			fmt.Fprintf(w, "- **%s**", markdownTextEscape(label))
		}

		if item.Detail != "" {
			fmt.Fprintf(w, " — %s", markdownTextEscape(item.Detail))
		}

		fmt.Fprintln(w)
	}
}
