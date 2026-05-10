package llmpack

import (
	"bytes"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/orurh/patchcourt/internal/diff/contract"
	"github.com/orurh/patchcourt/internal/diff/dep"
	"github.com/orurh/patchcourt/internal/diff/finding"
	"github.com/orurh/patchcourt/internal/model"
	"github.com/orurh/patchcourt/internal/platform/files"
	"github.com/orurh/patchcourt/internal/render/reviewcontract"
	"github.com/orurh/patchcourt/internal/render/reviewquestions"
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

	writeChangedFiles(w, result, limit)
	fmt.Fprintln(w)

	writeAnalyzedChangedFiles(w, result, limit)
	fmt.Fprintln(w)

	writeTouchedLayers(w, result, limit)
	fmt.Fprintln(w)

	writeArchitectureImpact(w, result.Impact, limit)
	fmt.Fprintln(w)

	writeAIFollowUpPrompt(w, result, limit)
	fmt.Fprintln(w)

	writeContractChanges(w, result.ContractChanges, limit)
	fmt.Fprintln(w)

	writeContractImpacts(w, result.ContractImpacts, limit)
	fmt.Fprintln(w)

	writeDependencyChanges(w, result.DependencyChanges, limit)
	fmt.Fprintln(w)

	writeLayerEdgeChanges(w, result.LayerEdgeChanges, limit)
	fmt.Fprintln(w)

	writeFindingChanges(w, result.FindingChanges, limit)
	fmt.Fprintln(w)

	writeRuntimeRiskChanges(w, result.FindingChanges, limit)
	fmt.Fprintln(w)

	writeReviewQuestions(w, result, limit)
}

func writeSummary(w io.Writer, result reportmodel.ReviewResult) {
	fmt.Fprintln(w, "## Summary")
	fmt.Fprintln(w)
	fmt.Fprintf(w, "- Schema: `%s`\n", result.SchemaVersion)
	fmt.Fprintf(w, "- Risk: `%s`, %d points\n", result.Risk.Level, result.Risk.Points)
	fmt.Fprintf(w, "- Changed files: %d\n", len(changedFiles(result)))
	fmt.Fprintf(w, "- Analyzed changed files: %d\n", len(analyzedChangedFiles(result)))
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

func writeChangedFiles(w io.Writer, result reportmodel.ReviewResult, limit int) {
	files := changedFiles(result)

	fmt.Fprintln(w, "## Changed files")
	fmt.Fprintln(w)

	if len(files) == 0 {
		fmt.Fprintln(w, "- none")
		return
	}

	for _, file := range limited(files, limit) {
		fmt.Fprintf(w, "- `%s`\n", file)
	}

	writeMore(w, len(files), limit)
}

func writeAnalyzedChangedFiles(w io.Writer, result reportmodel.ReviewResult, limit int) {
	files := analyzedChangedFiles(result)

	fmt.Fprintln(w, "## Analyzed changed files")
	fmt.Fprintln(w)

	if len(files) == 0 {
		fmt.Fprintln(w, "- none")
		return
	}

	for _, file := range limited(files, limit) {
		fmt.Fprintf(w, "- `%s`\n", file)
	}

	writeMore(w, len(files), limit)
}

func analyzedChangedFiles(result reportmodel.ReviewResult) []string {
	seen := make(map[string]struct{})

	addDependencyFiles(seen, result.DependencyChanges)
	addFindingFiles(seen, result.FindingChanges)

	for _, change := range result.ContractChanges {
		if change.Before != nil {
			addNonEmpty(seen, change.Before.File)
		}
		if change.After != nil {
			addNonEmpty(seen, change.After.File)
		}
	}

	return sortedKeys(seen)
}

func writeTouchedLayers(w io.Writer, result reportmodel.ReviewResult, limit int) {
	layers := touchedLayers(result)

	fmt.Fprintln(w, "## Touched layers")
	fmt.Fprintln(w)

	if len(layers) == 0 {
		fmt.Fprintln(w, "- none")
		return
	}

	for _, layer := range limited(layers, limit) {
		fmt.Fprintf(w, "- `%s`\n", layer)
	}

	writeMore(w, len(layers), limit)
}

func changedFiles(result reportmodel.ReviewResult) []string {
	if len(result.ChangedFiles) > 0 {
		seen := make(map[string]struct{}, len(result.ChangedFiles))
		for _, file := range result.ChangedFiles {
			addNonEmpty(seen, file)
		}
		return sortedKeys(seen)
	}

	seen := make(map[string]struct{})

	addDependencyFiles(seen, result.DependencyChanges)
	addFindingFiles(seen, result.FindingChanges)

	for _, change := range result.ContractChanges {
		if change.Before != nil {
			addNonEmpty(seen, change.Before.File)
		}
		if change.After != nil {
			addNonEmpty(seen, change.After.File)
		}
	}

	return sortedKeys(seen)
}

func addDependencyFiles(seen map[string]struct{}, changes []depdiff.DependencyChange) {
	for _, change := range changes {
		if change.Before != nil {
			addNonEmpty(seen, change.Before.FromFile)
			addNonEmpty(seen, change.Before.ToFile)
		}

		if change.After != nil {
			addNonEmpty(seen, change.After.FromFile)
			addNonEmpty(seen, change.After.ToFile)
		}
	}
}

func addFindingFiles(seen map[string]struct{}, changes []findingdiff.FindingChange) {
	for _, change := range changes {
		if change.Before != nil {
			for _, evidence := range change.Before.Evidence {
				addNonEmpty(seen, evidence.File)
				addNonEmpty(seen, evidence.FromFile)
				addNonEmpty(seen, evidence.ToFile)
			}
		}

		if change.After != nil {
			for _, evidence := range change.After.Evidence {
				addNonEmpty(seen, evidence.File)
				addNonEmpty(seen, evidence.FromFile)
				addNonEmpty(seen, evidence.ToFile)
			}
		}
	}
}

func touchedLayers(result reportmodel.ReviewResult) []string {
	seen := make(map[string]struct{})

	for _, change := range result.LayerEdgeChanges {
		addNonEmpty(seen, change.FromLayer)
		addNonEmpty(seen, change.ToLayer)
	}

	for _, change := range result.DependencyChanges {
		if change.Before != nil {
			addNonEmpty(seen, change.Before.FromLayer)
			addNonEmpty(seen, change.Before.ToLayer)
		}

		if change.After != nil {
			addNonEmpty(seen, change.After.FromLayer)
			addNonEmpty(seen, change.After.ToLayer)
		}
	}

	for _, change := range result.FindingChanges {
		if change.Before != nil {
			for _, evidence := range change.Before.Evidence {
				addNonEmpty(seen, evidence.FromLayer)
				addNonEmpty(seen, evidence.ToLayer)
			}
		}

		if change.After != nil {
			for _, evidence := range change.After.Evidence {
				addNonEmpty(seen, evidence.FromLayer)
				addNonEmpty(seen, evidence.ToLayer)
			}
		}
	}

	return sortedKeys(seen)
}

func addNonEmpty(values map[string]struct{}, value string) {
	if value == "" {
		return
	}

	values[value] = struct{}{}
}

func sortedKeys(values map[string]struct{}) []string {
	keys := make([]string, 0, len(values))

	for value := range values {
		keys = append(keys, value)
	}

	sort.Strings(keys)
	return keys
}

func writeArchitectureImpact(w io.Writer, impact reportmodel.ReviewImpactReport, limit int) {
	fmt.Fprintln(w, "## Architecture impact")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "PatchCourt only marks problems and improvements when it has policy-backed or high-confidence evidence.")
	fmt.Fprintln(w, "Other architecture movements are listed as review items for human or AI follow-up.")
	fmt.Fprintln(w)

	writeImpactItems(w, "### Real problems introduced", impact.Worse, limit)
	fmt.Fprintln(w)

	writeImpactItems(w, "### Verified improvements", impact.Better, limit)
	fmt.Fprintln(w)

	writeImpactItems(w, "### Needs review / AI follow-up", impact.NeedsReview, limit)
	fmt.Fprintln(w)

	writeImpactItems(w, "### Existing debt", impact.UnchangedDebt, limit)
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
func writeAIFollowUpPrompt(w io.Writer, result reportmodel.ReviewResult, limit int) {
	fmt.Fprintln(w, "## AI follow-up prompt")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Use this prompt for a follow-up AI review or fix pass:")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "```text")
	fmt.Fprintln(w, "You are reviewing a patch using PatchCourt evidence.")
	fmt.Fprintln(w, "Do not invent files, dependencies, symbols, call sites, tests, or findings not listed in this context pack.")
	fmt.Fprintln(w)

	if len(result.Impact.Worse) == 0 {
		fmt.Fprintln(w, "PatchCourt found no proven architecture regression in this patch.")
	} else {
		fmt.Fprintf(w, "PatchCourt found %d proven architecture problem(s). Treat these as the highest-priority issues.\n", len(result.Impact.Worse))
	}

	if len(result.Impact.Better) == 0 {
		fmt.Fprintln(w, "PatchCourt found no policy-backed verified architecture improvement.")
	} else {
		fmt.Fprintf(w, "PatchCourt found %d verified architecture improvement(s). Preserve these while making fixes.\n", len(result.Impact.Better))
	}

	if len(result.Impact.NeedsReview) > 0 {
		fmt.Fprintln(w)
		fmt.Fprintf(w, "The patch has %d architecture-relevant item(s) that need review. They are not proven bad by PatchCourt, but they may require migration, tests, explanation, or architecture cleanup.\n", len(result.Impact.NeedsReview))
		fmt.Fprintln(w)
		fmt.Fprintln(w, "Review these themes:")

		groups := buildAIFollowUpGroups(result.Impact.NeedsReview)
		for i, group := range groups {
			writeAIFollowUpGroup(w, i+1, group, limit)
		}
	} else {
		fmt.Fprintln(w)
		fmt.Fprintln(w, "PatchCourt found no needs-review architecture items.")
	}

	if len(result.Impact.UnchangedDebt) > 0 {
		fmt.Fprintln(w)
		fmt.Fprintf(w, "There are %d existing debt item(s) not introduced by this patch. Do not blame the patch for them, but avoid making them worse.\n", len(result.Impact.UnchangedDebt))
	}

	fmt.Fprintln(w)
	fmt.Fprintln(w, "Your task:")
	if len(result.Impact.NeedsReview) > 0 {
		fmt.Fprintln(w, "1. Explain whether each review theme is intentional refactor/migration or accidental architecture drift.")
		fmt.Fprintln(w, "2. Identify missing tests, compatibility checks, or architecture cleanup steps using only the evidence listed here.")
		fmt.Fprintln(w, "3. Propose minimal code changes only when the evidence supports them.")
		fmt.Fprintln(w, "4. If evidence is insufficient, say exactly what file/call site/test should be inspected next.")
	} else {
		fmt.Fprintln(w, "1. Confirm that there is no patch-specific architecture action required from the listed evidence.")
		fmt.Fprintln(w, "2. Do not propose code changes unless they are supported by listed facts.")
		fmt.Fprintln(w, "3. Note existing debt separately and do not blame this patch for it.")
	}
	fmt.Fprintln(w, "```")
}

type aiFollowUpGroup struct {
	Key   string
	Title string
	Why   string
	Ask   string
	Items []reportmodel.ReviewImpactItem
}

func buildAIFollowUpGroups(items []reportmodel.ReviewImpactItem) []aiFollowUpGroup {
	groups := make(map[string]*aiFollowUpGroup)
	order := make([]string, 0)

	for _, item := range items {
		key, title, why, ask := aiFollowUpGroupInfo(item)

		group, ok := groups[key]
		if !ok {
			group = &aiFollowUpGroup{
				Key:   key,
				Title: title,
				Why:   why,
				Ask:   ask,
				Items: make([]reportmodel.ReviewImpactItem, 0),
			}
			groups[key] = group
			order = append(order, key)
		}

		group.Items = append(group.Items, item)
	}

	result := make([]aiFollowUpGroup, 0, len(order))
	for _, key := range order {
		result = append(result, *groups[key])
	}

	return result
}

func aiFollowUpGroupInfo(item reportmodel.ReviewImpactItem) (key string, title string, why string, ask string) {
	kind := item.Kind

	switch {
	case strings.HasPrefix(kind, "contract_"):
		return "contract",
			"Contract boundary / API migration",
			"Public contracts changed, but PatchCourt cannot prove this is a regression without migration, compatibility, and call-site context.",
			"Verify whether behavior was intentionally migrated to replacement contracts, whether API compatibility is preserved, and what tests should cover the migration."

	case strings.HasPrefix(kind, "dependency_"):
		return "dependency",
			"Dependency movement",
			"Include/import dependencies changed. PatchCourt can prove the movement, but not whether it is better or worse without architecture intent.",
			"Explain whether the dependency movement is intentional, whether it follows the intended layering, and whether any moved dependency should be replaced, isolated, or documented."

	case strings.HasPrefix(kind, "layer_edge_"):
		return "layer_edge",
			"Layer graph movement",
			"Layer edge counts changed. Count movement alone is not proof of improvement or regression.",
			"Check whether the layer movement matches the intended architecture and whether it reduced known debt or introduced accidental coupling."

	case strings.HasPrefix(kind, "discovery_signal_"):
		return "discovery",
			"Discovery signals",
			"Heuristic architecture signals changed. These are review candidates, not proven violations.",
			"Verify whether the signal reflects real architecture drift, misplaced shared/config code, or an intentional project-specific layout."

	default:
		return "other",
			"Other review items",
			"PatchCourt found architecture-relevant review items that do not fit a stronger proven category.",
			"Review the listed evidence and decide whether follow-up code changes, tests, or documentation are needed."
	}
}

func writeAIFollowUpGroup(w io.Writer, index int, group aiFollowUpGroup, limit int) {
	fmt.Fprintf(w, "\nTheme %d: %s\n", index, group.Title)
	fmt.Fprintf(w, "Why it matters: %s\n", group.Why)
	fmt.Fprintf(w, "Ask AI: %s\n", group.Ask)
	fmt.Fprintln(w, "Items:")

	for _, item := range limited(group.Items, limit) {
		writeAIFollowUpItem(w, item)
	}

	writeIndentedMore(w, len(group.Items), limit, "")
}

func writeAIFollowUpItem(w io.Writer, item reportmodel.ReviewImpactItem) {
	fmt.Fprintf(w, "- %s: %s", item.Kind, item.Title)

	if item.ID != "" {
		fmt.Fprintf(w, " [%s]", item.ID)
	}

	if item.Detail != "" {
		fmt.Fprintf(w, " — %s", item.Detail)
	}

	fmt.Fprintln(w)

	if item.Suggestion != "" {
		fmt.Fprintf(w, "  Suggested review: %s\n", item.Suggestion)
	}

	if len(item.Evidence) > 0 {
		fmt.Fprintf(w, "  Evidence items: %d\n", len(item.Evidence))
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
		fmt.Fprintf(w, "- `%s` `%s` `%s`\n", change.Kind, reviewcontract.ClassifyImpact(change), change.SymbolKey)

		if location := reviewcontract.Location(change); location != "" {
			fmt.Fprintf(w, "  - location: `%s`\n", location)
		}

		if change.Before != nil && change.Before.Signature != "" {
			fmt.Fprintf(w, "  - before: `%s`\n", change.Before.Signature)
		}

		if change.After != nil && change.After.Signature != "" {
			fmt.Fprintf(w, "  - after: `%s`\n", change.After.Signature)
		}
	}

	writeMore(w, len(changes), limit)
}

func writeContractImpacts(w io.Writer, impacts []reportmodel.ContractImpact, limit int) {
	fmt.Fprintln(w, "## Contract impact")
	fmt.Fprintln(w)

	if len(impacts) == 0 {
		fmt.Fprintln(w, "- none")
		return
	}

	for _, impact := range limited(impacts, limit) {
		fmt.Fprintf(w, "- `%s` `%s` `%s`\n", impact.ChangeKind, impact.Impact, impact.SymbolKey)

		if impact.Location != "" {
			fmt.Fprintf(w, "  - location: `%s`\n", impact.Location)
		}

		if impact.ParentName != "" || impact.MethodName != "" {
			fmt.Fprintf(w, "  - symbol: `%s::%s`\n", impact.ParentName, impact.MethodName)
		}

		fmt.Fprintf(w, "  - delivery impacted: `%t`\n", impact.DeliveryImpacted)
		fmt.Fprintf(w, "  - tests changed: `%t`\n", impact.TestsChanged)

		if impact.Confidence != "" {
			fmt.Fprintf(w, "  - confidence: `%s`\n", impact.Confidence)
		}

		if len(impact.ImpactedFiles) == 0 {
			continue
		}

		fmt.Fprintln(w, "  - impacted files:")
		for _, file := range limited(impact.ImpactedFiles, 5) {
			layer := file.Layer
			if layer == "" {
				layer = "unknown"
			}

			if file.Line > 0 {
				fmt.Fprintf(w, "    - `%s:%d` `%s` %s\n", file.File, file.Line, layer, file.Reason)
			} else {
				fmt.Fprintf(w, "    - `%s` `%s` %s\n", file.File, layer, file.Reason)
			}
		}

		writeIndentedMore(w, len(impact.ImpactedFiles), 5, "    ")
	}

	writeMore(w, len(impacts), limit)
}

func writeDependencyChanges(w io.Writer, changes []depdiff.DependencyChange, limit int) {
	fmt.Fprintln(w, "## Dependency changes")
	fmt.Fprintln(w)

	rawCount := len(changes)
	changes = reviewRelevantDependencyChanges(changes)

	if len(changes) == 0 {
		if rawCount > 0 {
			fmt.Fprintf(w, "- none review-relevant; raw dependency changes: %d\n", rawCount)
			return
		}

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
			writeFindingGuidance(w, "before", *change.Before)
			writeFindingEvidence(w, "before evidence", change.Before.Evidence, 3)
		}

		if change.After != nil {
			fmt.Fprintf(w, "  - after: `%s/%s` %s\n", change.After.Severity, change.After.Kind, change.After.Title)
			writeFindingGuidance(w, "after", *change.After)
			writeFindingEvidence(w, "after evidence", change.After.Evidence, 3)
		}
	}

	writeMore(w, len(changes), limit)
}

func writeFindingGuidance(w io.Writer, prefix string, finding model.Finding) {
	if finding.Risk != "" {
		fmt.Fprintf(w, "  - %s risk: %s\n", prefix, finding.Risk)
	}

	if finding.Suggestion != "" {
		fmt.Fprintf(w, "  - %s suggestion: %s\n", prefix, finding.Suggestion)
	}
}

func writeFindingEvidence(w io.Writer, title string, evidence []model.Evidence, limit int) {
	if len(evidence) == 0 {
		return
	}

	fmt.Fprintf(w, "  - %s:\n", title)

	for _, item := range limited(evidence, limit) {
		fmt.Fprintf(w, "    - %s\n", evidenceText(item))
	}

	writeIndentedMore(w, len(evidence), limit, "    ")
}

func evidenceText(evidence model.Evidence) string {
	location := evidence.File
	if location == "" {
		location = evidence.FromFile
	}

	if evidence.LineStart > 0 {
		if evidence.LineEnd > evidence.LineStart {
			location = fmt.Sprintf("%s:%d-%d", location, evidence.LineStart, evidence.LineEnd)
		} else {
			location = fmt.Sprintf("%s:%d", location, evidence.LineStart)
		}
	}

	detail := evidence.Message
	if detail == "" {
		detail = evidence.Snippet
	}
	if detail == "" && (evidence.FromFile != "" || evidence.ToFile != "") {
		detail = fmt.Sprintf("%s -> %s", evidence.FromFile, evidence.ToFile)
	}
	if detail == "" && (evidence.FromLayer != "" || evidence.ToLayer != "") {
		detail = fmt.Sprintf("%s -> %s", evidence.FromLayer, evidence.ToLayer)
	}

	switch {
	case location != "" && detail != "":
		return fmt.Sprintf("`%s` — %s%s", location, detail, evidenceLayerSuffix(evidence))
	case location != "":
		return fmt.Sprintf("`%s`%s", location, evidenceLayerSuffix(evidence))
	case detail != "":
		return detail + evidenceLayerSuffix(evidence)
	default:
		return "evidence item"
	}
}

func evidenceLayerSuffix(evidence model.Evidence) string {
	if evidence.FromLayer == "" && evidence.ToLayer == "" {
		return ""
	}

	return fmt.Sprintf(" `%s -> %s`", evidence.FromLayer, evidence.ToLayer)
}

func writeIndentedMore(w io.Writer, total int, limit int, indent string) {
	if total > limit {
		fmt.Fprintf(w, "%s- ... %d more\n", indent, total-limit)
	}
}

func writeRuntimeRiskChanges(w io.Writer, changes []findingdiff.FindingChange, limit int) {
	runtimeChanges := runtimeRiskChanges(changes)

	fmt.Fprintln(w, "## Runtime architecture risks")
	fmt.Fprintln(w)

	if len(runtimeChanges) == 0 {
		fmt.Fprintln(w, "- none")
		return
	}

	for _, change := range limited(runtimeChanges, limit) {
		finding := change.After
		if finding == nil {
			finding = change.Before
		}
		if finding == nil {
			continue
		}

		fmt.Fprintf(w, "- `%s` `%s` `%s`", change.Kind, finding.Severity, change.ID)
		if finding.Title != "" {
			fmt.Fprintf(w, " — %s", finding.Title)
		}
		if finding.Confidence != "" {
			fmt.Fprintf(w, " _(confidence: %s)_", finding.Confidence)
		}
		fmt.Fprintln(w)

		if finding.Risk != "" {
			fmt.Fprintf(w, "  - risk: %s\n", finding.Risk)
		}
		if finding.Suggestion != "" {
			fmt.Fprintf(w, "  - suggestion: %s\n", finding.Suggestion)
		}

		writeFindingEvidence(w, "evidence", runtimeChangeEvidence(change, *finding), 5)
	}

	writeMore(w, len(runtimeChanges), limit)
}

func runtimeRiskChanges(changes []findingdiff.FindingChange) []findingdiff.FindingChange {
	result := make([]findingdiff.FindingChange, 0)

	for _, change := range changes {
		finding := change.After
		if finding == nil {
			finding = change.Before
		}
		if finding == nil {
			continue
		}

		if finding.Kind != model.FindingKindRuntimeRisk {
			continue
		}

		result = append(result, change)
	}

	return result
}

func runtimeChangeEvidence(change findingdiff.FindingChange, finding model.Finding) []model.Evidence {
	switch change.Kind {
	case findingdiff.FindingChangeKindAdded:
		if len(change.AddedEvidence) > 0 {
			return change.AddedEvidence
		}
	case findingdiff.FindingChangeKindRemoved:
		if len(change.RemovedEvidence) > 0 {
			return change.RemovedEvidence
		}
	case findingdiff.FindingChangeKindChanged:
		if len(change.AddedEvidence) > 0 {
			return change.AddedEvidence
		}
	}

	return finding.Evidence
}

func writeReviewQuestions(w io.Writer, result reportmodel.ReviewResult, limit int) {
	fmt.Fprintln(w, "## Review questions")
	fmt.Fprintln(w)

	questions := reviewquestions.Build(result)
	if len(questions) == 0 {
		fmt.Fprintln(w, "- No specific high-signal questions generated from the current facts.")
		return
	}

	for _, question := range limited(questions, limit) {
		fmt.Fprintf(w, "- %s\n", question)
	}

	writeIndentedMore(w, len(questions), limit, "")
}

func reviewRelevantDependencyChanges(changes []depdiff.DependencyChange) []depdiff.DependencyChange {
	filtered := make([]depdiff.DependencyChange, 0, len(changes))

	for _, change := range changes {
		if isReviewRelevantDependencyChange(change) {
			filtered = append(filtered, change)
		}
	}

	return filtered
}

func isReviewRelevantDependencyChange(change depdiff.DependencyChange) bool {
	if change.Before != nil && isReviewRelevantDependency(*change.Before) {
		return true
	}

	if change.After != nil && isReviewRelevantDependency(*change.After) {
		return true
	}

	return false
}

func isReviewRelevantDependency(dep model.DependencyEdge) bool {
	if dep.External {
		return false
	}

	if dep.FromLayer == "" && dep.ToLayer == "" {
		return false
	}

	return dep.ToFile != "" || dep.ToLayer != ""
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
