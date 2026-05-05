package review

import (
	"fmt"
	"html"
	"io"
	"strings"

	"github.com/orurh/patchcourt/internal/model"
	"github.com/orurh/patchcourt/internal/reportmodel"
)

func WriteReviewHTML(w io.Writer, result reportmodel.ReviewResult) error {
	view := BuildReviewView(result)

	var b strings.Builder

	fmt.Fprintln(&b, "<!doctype html>")
	fmt.Fprintln(&b, `<html lang="en">`)
	fmt.Fprintln(&b, "<head>")
	fmt.Fprintln(&b, `<meta charset="utf-8">`)
	fmt.Fprintln(&b, `<meta name="viewport" content="width=device-width, initial-scale=1">`)
	fmt.Fprintln(&b, "<title>PatchCourt Review</title>")
	fmt.Fprintln(&b, "<style>")
	fmt.Fprintln(&b, reviewHTMLCSS())
	fmt.Fprintln(&b, "</style>")
	fmt.Fprintln(&b, "</head>")
	fmt.Fprintln(&b, "<body>")
	fmt.Fprintln(&b, `<main class="page">`)

	fmt.Fprintln(&b, `<section class="hero">`)
	fmt.Fprintln(&b, `<div>`)
	fmt.Fprintln(&b, `<p class="eyebrow">PatchCourt</p>`)
	fmt.Fprintf(&b, `<h1>%s</h1>`, escape(view.Title))
	fmt.Fprintln(&b)
	fmt.Fprintf(&b, `<p class="muted">%s</p>`, escape(view.Description))
	fmt.Fprintln(&b)
	fmt.Fprintln(&b, `</div>`)
	fmt.Fprintf(&b, `<div class="risk risk-%s">`, htmlClass(view.RiskLevel))
	fmt.Fprintf(&b, `<div class="risk-label">%s</div>`, escape(view.RiskLevel))
	fmt.Fprintf(&b, `<div class="risk-points">%d points</div>`, view.RiskPoints)
	fmt.Fprintln(&b, `</div>`)
	fmt.Fprintln(&b, `</section>`)

	writeReviewHTMLSummary(&b, view.SummaryCards)
	writeReviewHTMLImpact(&b, view.Impact)
	writeReviewHTMLLayerImpactGraph(&b, view.LayerGraph)
	writeReviewHTMLChangedFiles(&b, "Changed files", view.ChangedFiles)
	writeReviewHTMLContractChanges(&b, view.ContractRows)
	writeReviewHTMLContractImpacts(&b, view.ContractImpacts)
	writeReviewHTMLDependencyChanges(&b, view.DependencyRows)
	writeReviewHTMLLayerEdgeChanges(&b, view.LayerEdgeRows)
	writeReviewHTMLFindingChanges(&b, view.FindingRows)
	writeReviewHTMLRiskReasons(&b, view.RiskReasons)
	writeReviewHTMLReviewQuestions(&b, view.ReviewQuestions)
	writeReviewHTMLCounts(&b, view.RawCounts)

	fmt.Fprintln(&b, "</main>")
	fmt.Fprintln(&b, "</body>")
	fmt.Fprintln(&b, "</html>")

	_, err := io.WriteString(w, b.String())
	return err
}

func writeReviewHTMLSummary(b *strings.Builder, cards []ReviewMetricCard) {
	fmt.Fprintln(b, `<section class="grid">`)
	for _, card := range cards {
		writeMetricCard(b, card.Title, card.Value)
	}
	fmt.Fprintln(b, `</section>`)
}

func writeMetricCard(b *strings.Builder, title string, value int) {
	fmt.Fprintln(b, `<article class="card metric">`)
	fmt.Fprintf(b, `<div class="metric-value">%d</div>`, value)
	fmt.Fprintf(b, `<div class="metric-title">%s</div>`, escape(title))
	fmt.Fprintln(b, `</article>`)
}

func writeReviewHTMLImpact(b *strings.Builder, columns []ReviewImpactColumn) {
	fmt.Fprintln(b, `<section class="card">`)
	fmt.Fprintln(b, `<h2>Architecture impact</h2>`)

	var worse ReviewImpactColumn
	var better ReviewImpactColumn
	var debt ReviewImpactColumn

	for _, column := range columns {
		switch column.Title {
		case "Worse":
			worse = column
		case "Better":
			better = column
		case "Unchanged debt":
			debt = column
		}
	}

	hasNewImpact := len(worse.Items) > 0 || len(better.Items) > 0

	if !hasNewImpact {
		fmt.Fprintln(b, `<div class="empty-verdict">`)
		fmt.Fprintln(b, `<div class="empty-verdict-icon">✓</div>`)
		fmt.Fprintln(b, `<div>`)
		fmt.Fprintln(b, `<h3>No new architecture impact detected</h3>`)
		fmt.Fprintln(b, `<p class="muted">This patch did not introduce worse or better architecture impact according to deterministic PatchCourt facts.</p>`)
		fmt.Fprintln(b, `</div>`)
		fmt.Fprintln(b, `</div>`)
	} else {
		fmt.Fprintln(b, `<div class="columns two-columns">`)
		writeReviewHTMLImpactColumn(b, worse.Title, worse.Class, worse.Items)
		writeReviewHTMLImpactColumn(b, better.Title, better.Class, better.Items)
		fmt.Fprintln(b, `</div>`)
	}

	fmt.Fprintln(b, `<details class="debt-details">`)
	fmt.Fprintf(b, `<summary>Existing unchanged debt: <strong>%d</strong> item(s)</summary>`, len(debt.Items))
	fmt.Fprintln(b)

	if len(debt.Items) == 0 {
		fmt.Fprintln(b, `<p class="muted">No unchanged debt.</p>`)
	} else {
		writeReviewHTMLImpactList(b, debt.Items)
	}

	fmt.Fprintln(b, `</details>`)
	fmt.Fprintln(b, `</section>`)
}

func writeReviewHTMLImpactList(b *strings.Builder, items []reportmodel.ReviewImpactItem) {
	fmt.Fprintln(b, `<div class="debt-list">`)
	for _, item := range items {
		fmt.Fprintln(b, `<details class="debt-item">`)
		fmt.Fprintln(b, `<summary>`)
		fmt.Fprintf(b, `<span class="tag">%s</span> `, escape(item.Kind))
		fmt.Fprintf(b, `<strong>%s</strong>`, escape(item.Title))
		if item.ID != "" {
			fmt.Fprintf(b, ` <code>%s</code>`, escape(item.ID))
		}
		fmt.Fprintln(b, `</summary>`)

		fmt.Fprintln(b, `<div class="debt-body">`)
		fmt.Fprintln(b, `<p class="muted">Status: existing debt, not introduced by this patch.</p>`)

		if item.Severity != "" {
			fmt.Fprintf(b, `<p><strong>Severity:</strong> <span class="tag">%s</span></p>`, escape(item.Severity))
		}
		if item.Detail != "" {
			fmt.Fprintf(b, `<p><strong>Finding:</strong> %s</p>`, escape(item.Detail))
		}
		if item.Risk != "" {
			fmt.Fprintf(b, `<p><strong>Risk:</strong> %s</p>`, escape(item.Risk))
		}
		if item.Suggestion != "" {
			fmt.Fprintf(b, `<p><strong>Suggestion:</strong> %s</p>`, escape(item.Suggestion))
		}

		writeReviewHTMLEvidenceList(b, item.Evidence, 5)

		fmt.Fprintln(b, `</div>`)
		fmt.Fprintln(b, `</details>`)
	}
	fmt.Fprintln(b, `</div>`)
}

func writeReviewHTMLEvidenceList(b *strings.Builder, evidence []model.Evidence, limit int) {
	if len(evidence) == 0 {
		fmt.Fprintln(b, `<p class="muted">No evidence attached to this finding yet.</p>`)
		return
	}

	fmt.Fprintln(b, `<h4>Evidence</h4>`)
	fmt.Fprintln(b, `<ul class="evidence-list">`)

	count := len(evidence)
	if count > limit {
		count = limit
	}

	for i := 0; i < count; i++ {
		item := evidence[i]
		location := evidenceLocation(item)
		detail := item.Message
		if detail == "" {
			detail = item.Snippet
		}
		if detail == "" && (item.FromFile != "" || item.ToFile != "") {
			detail = item.FromFile + " -> " + item.ToFile
		}

		fmt.Fprintln(b, `<li>`)
		if location != "" {
			fmt.Fprintf(b, `<code>%s</code>`, escape(location))
		}
		if detail != "" {
			fmt.Fprintf(b, `<div class="detail">%s</div>`, escape(detail))
		}
		if item.FromLayer != "" || item.ToLayer != "" {
			fmt.Fprintf(b, `<div class="detail">Layer: <code>%s → %s</code></div>`, escape(item.FromLayer), escape(item.ToLayer))
		}
		fmt.Fprintln(b, `</li>`)
	}

	if len(evidence) > limit {
		fmt.Fprintf(b, `<li class="muted">... %d more evidence item(s)</li>`, len(evidence)-limit)
	}

	fmt.Fprintln(b, `</ul>`)
}

func evidenceLocation(evidence model.Evidence) string {
	location := evidence.File
	if location == "" {
		location = evidence.FromFile
	}

	if location == "" {
		return ""
	}

	switch {
	case evidence.LineStart > 0 && evidence.LineEnd > evidence.LineStart:
		return fmt.Sprintf("%s:%d-%d", location, evidence.LineStart, evidence.LineEnd)
	case evidence.LineStart > 0:
		return fmt.Sprintf("%s:%d", location, evidence.LineStart)
	default:
		return location
	}
}

func writeReviewHTMLImpactColumn(b *strings.Builder, title string, class string, items []reportmodel.ReviewImpactItem) {
	fmt.Fprintf(b, `<div class="impact impact-%s">`, escape(class))
	fmt.Fprintf(b, `<h3>%s</h3>`, escape(title))

	if len(items) == 0 {
		fmt.Fprintln(b, `<p class="muted">None.</p>`)
		fmt.Fprintln(b, `</div>`)
		return
	}

	fmt.Fprintln(b, `<ul>`)
	for _, item := range items {
		fmt.Fprintln(b, `<li>`)
		if item.Kind != "" {
			fmt.Fprintf(b, `<span class="tag">%s</span> `, escape(item.Kind))
		}
		fmt.Fprintf(b, `<strong>%s</strong>`, escape(item.Title))
		if item.ID != "" {
			fmt.Fprintf(b, ` <code>%s</code>`, escape(item.ID))
		}
		if item.Detail != "" {
			fmt.Fprintf(b, `<div class="detail">%s</div>`, escape(item.Detail))
		}
		fmt.Fprintln(b, `</li>`)
	}
	fmt.Fprintln(b, `</ul>`)
	fmt.Fprintln(b, `</div>`)
}

func writeReviewHTMLLayerImpactGraph(b *strings.Builder, graph ReviewLayerGraph) {
	fmt.Fprintln(b, `<section class="card">`)
	fmt.Fprintf(b, `<h2>%s</h2>`, escape(graph.Title))
	fmt.Fprintln(b)

	if len(graph.Rows) == 0 {
		fmt.Fprintln(b, `<p class="muted">No layer edge changes.</p>`)
		fmt.Fprintln(b, `</section>`)
		return
	}

	fmt.Fprintf(b, `<p class="muted">%s</p>`, escape(graph.Description))
	fmt.Fprintln(b)
	fmt.Fprintln(b, `<pre class="graph-block"><code>graph LR`)

	for _, row := range graph.Rows {
		fmt.Fprintf(b, `  %s["%s"] --> %s["%s"]`, row.FromID, escape(row.FromLayer), row.ToID, escape(row.ToLayer))

		if row.Kind != "" {
			fmt.Fprintf(b, `:::edge_%s`, htmlClass(row.Kind))
		}

		fmt.Fprintln(b)
	}

	fmt.Fprintln(b, `</code></pre>`)
	fmt.Fprintln(b, `</section>`)
}

func writeReviewHTMLChangedFiles(b *strings.Builder, title string, files []string) {
	fmt.Fprintln(b, `<section class="card">`)
	fmt.Fprintf(b, `<h2>%s</h2>`, escape(title))
	fmt.Fprintln(b)

	if len(files) == 0 {
		fmt.Fprintln(b, `<p class="muted">None.</p>`)
		fmt.Fprintln(b, `</section>`)
		return
	}

	fmt.Fprintln(b, `<ul class="file-list">`)
	for _, file := range files {
		fmt.Fprintf(b, `<li><code>%s</code></li>`, escape(file))
		fmt.Fprintln(b)
	}
	fmt.Fprintln(b, `</ul>`)
	fmt.Fprintln(b, `</section>`)
}

func writeReviewHTMLContractChanges(b *strings.Builder, rows []ReviewContractRow) {
	fmt.Fprintln(b, `<section class="card">`)
	fmt.Fprintln(b, `<h2>Contract changes</h2>`)

	if len(rows) == 0 {
		fmt.Fprintln(b, `<p class="muted">No contract changes.</p>`)
		fmt.Fprintln(b, `</section>`)
		return
	}

	fmt.Fprintln(b, `<div class="contract-change-list">`)
	for _, row := range rows {
		modifiers := ""
		if row.AddedModifiers != "" {
			modifiers += "added: " + row.AddedModifiers
		}
		if row.RemovedModifiers != "" {
			if modifiers != "" {
				modifiers += "; "
			}
			modifiers += "removed: " + row.RemovedModifiers
		}

		location := contractLocation(row)

		fmt.Fprintln(b, `<article class="contract-change-card">`)
		fmt.Fprintf(b, `<div><span class="tag impact-%s">%s</span> <span class="tag">%s</span></div>`,
			htmlClass(row.Impact),
			escape(row.Impact),
			escape(row.Kind),
		)
		fmt.Fprintf(b, `<h3><code>%s</code></h3>`, escape(row.SymbolKey))

		if location != "" {
			fmt.Fprintf(b, `<p class="detail">Location: <code>%s</code></p>`, escape(location))
		}

		if row.BeforeSignature != "" {
			fmt.Fprintln(b, `<div class="code-diff-block">`)
			fmt.Fprintln(b, `<div class="code-diff-title">Before</div>`)
			fmt.Fprintf(b, `<pre><code>%s</code></pre>`, escape(row.BeforeSignature))
			fmt.Fprintln(b, `</div>`)
		}

		if row.AfterSignature != "" {
			fmt.Fprintln(b, `<div class="code-diff-block">`)
			fmt.Fprintln(b, `<div class="code-diff-title">After</div>`)
			fmt.Fprintf(b, `<pre><code>%s</code></pre>`, escape(row.AfterSignature))
			fmt.Fprintln(b, `</div>`)
		}

		if modifiers != "" {
			fmt.Fprintf(b, `<p class="detail">Modifiers: %s</p>`, escape(modifiers))
		}

		fmt.Fprintln(b, `</article>`)
	}
	fmt.Fprintln(b, `</div>`)
	fmt.Fprintln(b, `</section>`)
}

func contractLocation(row ReviewContractRow) string {
	if row.File == "" {
		return ""
	}

	switch {
	case row.BeforeLine > 0 && row.AfterLine > 0 && row.BeforeLine != row.AfterLine:
		return fmt.Sprintf("%s:%d → %d", row.File, row.BeforeLine, row.AfterLine)
	case row.AfterLine > 0:
		return fmt.Sprintf("%s:%d", row.File, row.AfterLine)
	case row.BeforeLine > 0:
		return fmt.Sprintf("%s:%d", row.File, row.BeforeLine)
	default:
		return row.File
	}
}

func writeReviewHTMLContractImpacts(b *strings.Builder, rows []ReviewContractImpactRow) {
	fmt.Fprintln(b, `<section class="card">`)
	fmt.Fprintln(b, `<h2>Contract impact</h2>`)

	if len(rows) == 0 {
		fmt.Fprintln(b, `<p class="muted">No contract impact detected.</p>`)
		fmt.Fprintln(b, `</section>`)
		return
	}

	fmt.Fprintln(b, `<div class="contract-impact-list">`)
	for _, row := range rows {
		fmt.Fprintln(b, `<article class="contract-impact-card">`)
		fmt.Fprintf(b, `<div><span class="tag impact-%s">%s</span> <span class="tag">%s</span></div>`,
			htmlClass(row.Impact),
			escape(row.Impact),
			escape(row.ChangeKind),
		)
		fmt.Fprintf(b, `<h3><code>%s</code></h3>`, escape(row.SymbolKey))

		if row.Location != "" {
			fmt.Fprintf(b, `<p class="detail">Location: <code>%s</code></p>`, escape(row.Location))
		}

		fmt.Fprintln(b, `<ul class="compact-list">`)
		fmt.Fprintf(b, `<li>Delivery/API impacted: <strong>%t</strong></li>`, row.DeliveryImpacted)
		fmt.Fprintf(b, `<li>Test-like files changed: <strong>%t</strong></li>`, row.TestsChanged)
		if row.Confidence != "" {
			fmt.Fprintf(b, `<li>Confidence: <strong>%s</strong></li>`, escape(row.Confidence))
		}
		fmt.Fprintln(b, `</ul>`)

		if len(row.ImpactedFiles) > 0 {
			fmt.Fprintln(b, `<div class="table-wrap">`)
			fmt.Fprintln(b, `<table>`)
			fmt.Fprintln(b, `<thead><tr><th>File</th><th>Layer</th><th>Reason</th><th>Line</th></tr></thead>`)
			fmt.Fprintln(b, `<tbody>`)
			for _, file := range row.ImpactedFiles {
				fmt.Fprintln(b, `<tr>`)
				fmt.Fprintf(b, `<td><code>%s</code></td>`, escape(file.File))
				fmt.Fprintf(b, `<td>%s</td>`, escape(file.Layer))
				fmt.Fprintf(b, `<td>%s</td>`, escape(file.Reason))
				if file.Line > 0 {
					fmt.Fprintf(b, `<td>%d</td>`, file.Line)
				} else {
					fmt.Fprintln(b, `<td></td>`)
				}
				fmt.Fprintln(b, `</tr>`)
			}
			fmt.Fprintln(b, `</tbody>`)
			fmt.Fprintln(b, `</table>`)
			fmt.Fprintln(b, `</div>`)
		}

		fmt.Fprintln(b, `</article>`)
	}
	fmt.Fprintln(b, `</div>`)
	fmt.Fprintln(b, `</section>`)
}

func writeReviewHTMLDependencyChanges(b *strings.Builder, rows []ReviewDependencyRow) {
	fmt.Fprintln(b, `<section class="card">`)
	fmt.Fprintln(b, `<h2>Dependency changes</h2>`)

	if len(rows) == 0 {
		fmt.Fprintln(b, `<p class="muted">No dependency changes.</p>`)
		fmt.Fprintln(b, `</section>`)
		return
	}

	fmt.Fprintln(b, `<div class="table-wrap">`)
	fmt.Fprintln(b, `<table>`)
	fmt.Fprintln(b, `<thead><tr><th>Kind</th><th>From</th><th>To</th><th>Layer</th><th>Usage</th></tr></thead>`)
	fmt.Fprintln(b, `<tbody>`)

	for _, row := range rows {
		layer := ""
		if row.FromLayer != "" || row.ToLayer != "" {
			layer = row.FromLayer + " → " + row.ToLayer
		}

		fmt.Fprintln(b, `<tr>`)
		fmt.Fprintf(b, `<td><span class="tag">%s</span></td>`, escape(row.Kind))
		fmt.Fprintf(b, `<td><code>%s</code></td>`, escape(row.From))
		fmt.Fprintf(b, `<td><code>%s</code></td>`, escape(row.To))
		fmt.Fprintf(b, `<td>%s</td>`, escape(layer))
		fmt.Fprintf(b, `<td>%s</td>`, escape(row.Usage))
		fmt.Fprintln(b, `</tr>`)
	}

	fmt.Fprintln(b, `</tbody>`)
	fmt.Fprintln(b, `</table>`)
	fmt.Fprintln(b, `</div>`)
	fmt.Fprintln(b, `</section>`)
}

func writeReviewHTMLLayerEdgeChanges(b *strings.Builder, rows []ReviewLayerEdgeRow) {
	fmt.Fprintln(b, `<section class="card">`)
	fmt.Fprintln(b, `<h2>Layer edge changes</h2>`)

	if len(rows) == 0 {
		fmt.Fprintln(b, `<p class="muted">No layer edge changes.</p>`)
		fmt.Fprintln(b, `</section>`)
		return
	}

	fmt.Fprintln(b, `<div class="table-wrap">`)
	fmt.Fprintln(b, `<table>`)
	fmt.Fprintln(b, `<thead><tr><th>Kind</th><th>Edge</th><th>Before</th><th>After</th></tr></thead>`)
	fmt.Fprintln(b, `<tbody>`)

	for _, row := range rows {
		fmt.Fprintln(b, `<tr>`)
		fmt.Fprintf(b, `<td><span class="tag">%s</span></td>`, escape(row.Kind))
		fmt.Fprintf(b, `<td><code>%s → %s</code></td>`, escape(row.FromLayer), escape(row.ToLayer))
		fmt.Fprintf(b, `<td>%d</td>`, row.BeforeCount)
		fmt.Fprintf(b, `<td>%d</td>`, row.AfterCount)
		fmt.Fprintln(b, `</tr>`)
	}

	fmt.Fprintln(b, `</tbody>`)
	fmt.Fprintln(b, `</table>`)
	fmt.Fprintln(b, `</div>`)
	fmt.Fprintln(b, `</section>`)
}

func writeReviewHTMLFindingChanges(b *strings.Builder, rows []ReviewFindingRow) {
	fmt.Fprintln(b, `<section class="card">`)
	fmt.Fprintln(b, `<h2>Finding changes</h2>`)

	if len(rows) == 0 {
		fmt.Fprintln(b, `<p class="muted">No finding changes.</p>`)
		fmt.Fprintln(b, `</section>`)
		return
	}

	fmt.Fprintln(b, `<div class="table-wrap">`)
	fmt.Fprintln(b, `<table>`)
	fmt.Fprintln(b, `<thead><tr><th>Kind</th><th>ID</th><th>Severity</th><th>Title</th></tr></thead>`)
	fmt.Fprintln(b, `<tbody>`)

	for _, row := range rows {
		fmt.Fprintln(b, `<tr>`)
		fmt.Fprintf(b, `<td><span class="tag">%s</span></td>`, escape(row.Kind))
		fmt.Fprintf(b, `<td><code>%s</code></td>`, escape(row.ID))
		fmt.Fprintf(b, `<td>%s</td>`, escape(row.Severity))
		fmt.Fprintf(b, `<td>%s</td>`, escape(row.Title))
		fmt.Fprintln(b, `</tr>`)
	}

	fmt.Fprintln(b, `</tbody>`)
	fmt.Fprintln(b, `</table>`)
	fmt.Fprintln(b, `</div>`)
	fmt.Fprintln(b, `</section>`)
}

func writeReviewHTMLRiskReasons(b *strings.Builder, reasons []ReviewRiskReason) {
	fmt.Fprintln(b, `<section class="card">`)
	fmt.Fprintln(b, `<h2>Risk reasons</h2>`)

	if len(reasons) == 0 {
		fmt.Fprintln(b, `<p class="muted">No risk reasons.</p>`)
		fmt.Fprintln(b, `</section>`)
		return
	}

	fmt.Fprintln(b, `<ul>`)
	for _, reason := range reasons {
		fmt.Fprintf(b, `<li><strong>+%d</strong> %s</li>`, reason.Points, escape(reason.Message))
		fmt.Fprintln(b)
	}
	fmt.Fprintln(b, `</ul>`)
	fmt.Fprintln(b, `</section>`)
}

func writeReviewHTMLReviewQuestions(b *strings.Builder, questions []ReviewQuestion) {
	fmt.Fprintln(b, `<section class="card">`)
	fmt.Fprintln(b, `<h2>Review questions</h2>`)

	if len(questions) == 0 {
		fmt.Fprintln(b, `<p class="muted">No review questions.</p>`)
		fmt.Fprintln(b, `</section>`)
		return
	}

	fmt.Fprintln(b, `<ul>`)
	for _, question := range questions {
		fmt.Fprintf(b, `<li>%s</li>`, escape(question.Text))
		fmt.Fprintln(b)
	}
	fmt.Fprintln(b, `</ul>`)
	fmt.Fprintln(b, `</section>`)
}

func writeReviewHTMLCounts(b *strings.Builder, cards []ReviewMetricCard) {
	fmt.Fprintln(b, `<section class="card">`)
	fmt.Fprintln(b, `<h2>Raw diff sections</h2>`)
	fmt.Fprintln(b, `<ul>`)
	for _, card := range cards {
		fmt.Fprintf(b, `<li>%s: <strong>%d</strong></li>`, escape(card.Title), card.Value)
		fmt.Fprintln(b)
	}
	fmt.Fprintln(b, `</ul>`)
	fmt.Fprintln(b, `</section>`)
}

func escape(value string) string {
	return html.EscapeString(value)
}

func htmlClass(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return "unknown"
	}

	return value
}

func reviewHTMLCSS() string {
	return `
:root {
  color-scheme: light;
  --bg: #f8fafc;
  --card: #ffffff;
  --text: #111827;
  --muted: #64748b;
  --line: #e5e7eb;
  --good: #047857;
  --bad: #b91c1c;
  --neutral: #475569;
}
* { box-sizing: border-box; }
body {
  margin: 0;
  background: var(--bg);
  color: var(--text);
  font: 14px/1.5 system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
}
.page {
  max-width: 1180px;
  margin: 0 auto;
  padding: 32px;
}
.hero {
  display: flex;
  justify-content: space-between;
  gap: 24px;
  align-items: center;
  margin-bottom: 20px;
}
.eyebrow {
  margin: 0 0 6px;
  color: var(--muted);
  font-weight: 700;
  letter-spacing: .08em;
  text-transform: uppercase;
}
h1 {
  margin: 0;
  font-size: 34px;
  line-height: 1.1;
}
h2 {
  margin: 0 0 16px;
  font-size: 20px;
}
h3 {
  margin: 0 0 12px;
  font-size: 16px;
}
.muted {
  color: var(--muted);
}
.card {
  background: var(--card);
  border: 1px solid var(--line);
  border-radius: 18px;
  box-shadow: 0 10px 30px rgb(15 23 42 / 0.06);
  padding: 20px;
  margin: 16px 0;
}
.grid {
  display: grid;
  grid-template-columns: repeat(6, minmax(0, 1fr));
  gap: 12px;
}
.metric {
  margin: 0;
}
.metric-value {
  font-size: 28px;
  font-weight: 800;
}
.metric-title {
  color: var(--muted);
}
.risk {
  min-width: 150px;
  border-radius: 18px;
  padding: 18px 20px;
  background: var(--card);
  border: 1px solid var(--line);
  text-align: right;
}
.risk-label {
  font-size: 22px;
  font-weight: 800;
  text-transform: uppercase;
}
.risk-points {
  color: var(--muted);
}
.risk-high,
.risk-critical {
  color: var(--bad);
}
.risk-low {
  color: var(--good);
}
.columns {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 16px;
}
.two-columns {
  grid-template-columns: repeat(2, minmax(0, 1fr));
}
.empty-verdict {
  display: flex;
  gap: 14px;
  align-items: flex-start;
  border: 1px solid rgb(4 120 87 / 0.25);
  background: rgb(4 120 87 / 0.06);
  border-radius: 14px;
  padding: 16px;
}
.empty-verdict-icon {
  display: grid;
  place-items: center;
  width: 34px;
  height: 34px;
  border-radius: 999px;
  background: rgb(4 120 87 / 0.12);
  color: var(--good);
  font-weight: 900;
}
.empty-verdict h3 {
  margin: 0 0 4px;
}
.debt-details {
  margin-top: 16px;
  border: 1px solid var(--line);
  border-radius: 14px;
  padding: 14px 16px;
}
.debt-details summary {
  cursor: pointer;
  color: var(--muted);
}
.debt-details ul {
  margin-top: 12px;
}
.debt-list {
  display: grid;
  gap: 10px;
  margin-top: 12px;
}
.debt-item {
  border: 1px solid var(--line);
  border-radius: 12px;
  padding: 12px 14px;
  background: #ffffff;
}
.debt-item summary {
  cursor: pointer;
}
.debt-body {
  margin-top: 12px;
}
.debt-body p + p {
  margin-top: 8px;
}
.evidence-list {
  margin-top: 8px;
}
.evidence-list li + li {
  margin-top: 10px;
}
.impact {
  border: 1px solid var(--line);
  border-radius: 14px;
  padding: 16px;
}
.impact-bad h3 { color: var(--bad); }
.impact-good h3 { color: var(--good); }
.impact-neutral h3 { color: var(--neutral); }
.impact-breaking {
  color: var(--bad);
  border-color: rgb(185 28 28 / 0.35);
  background: rgb(185 28 28 / 0.06);
}
.impact-risky {
  color: #b45309;
  border-color: rgb(180 83 9 / 0.35);
  background: rgb(180 83 9 / 0.06);
}
.impact-additive {
  color: var(--good);
  border-color: rgb(4 120 87 / 0.35);
  background: rgb(4 120 87 / 0.06);
}
.impact-informational {
  color: var(--neutral);
}
ul {
  padding-left: 20px;
  margin: 0;
}
li + li {
  margin-top: 8px;
}
.tag {
  display: inline-block;
  border: 1px solid var(--line);
  border-radius: 999px;
  padding: 1px 8px;
  color: var(--muted);
  font-size: 12px;
}
.detail {
  color: var(--muted);
  margin-top: 4px;
}
.contract-change-list,
.contract-impact-list {
  display: grid;
  gap: 14px;
}
.contract-change-card,
.contract-impact-card {
  border: 1px solid var(--line);
  border-radius: 14px;
  padding: 16px;
  background: #ffffff;
}
.contract-change-card h3,
.contract-impact-card h3 {
  margin-top: 10px;
}
.code-diff-block {
  margin-top: 12px;
}
.code-diff-title {
  color: var(--muted);
  font-size: 12px;
  font-weight: 700;
  letter-spacing: .04em;
  text-transform: uppercase;
  margin-bottom: 6px;
}
.code-diff-block pre {
  margin: 0;
  overflow: auto;
  background: #f8fafc;
  border: 1px solid var(--line);
  border-radius: 10px;
  padding: 10px 12px;
}
.code-diff-block code {
  background: transparent;
  padding: 0;
  white-space: pre;
}
.compact-list {
  margin: 10px 0 12px;
}
code {
  background: #f1f5f9;
  border-radius: 6px;
  padding: 2px 5px;
}
.graph-block {
  overflow: auto;
  background: #0f172a;
  color: #e5e7eb;
  border-radius: 14px;
  padding: 16px;
}
.graph-block code {
  background: transparent;
  color: inherit;
  padding: 0;
}
.file-list {
  columns: 2;
}
.table-wrap {
  overflow: auto;
}
table {
  width: 100%;
  border-collapse: collapse;
}
th,
td {
  border-bottom: 1px solid var(--line);
  padding: 8px 10px;
  text-align: left;
  vertical-align: top;
}
th {
  color: var(--muted);
  font-size: 12px;
  text-transform: uppercase;
  letter-spacing: .04em;
}
td code {
  word-break: break-word;
}

/* PatchCourt release override: keep long evidence paths inside cards. */
.card,
.impact,
.impact li,
.contract-change-card,
.contract-impact-card,
.table-wrap,
section {
  min-width: 0;
}

.impact li,
.detail,
.file-list li,
.contract-change-card,
.contract-impact-card,
td,
th {
  overflow-wrap: anywhere;
  word-break: break-word;
}

code {
  max-width: 100%;
  overflow-wrap: anywhere;
  word-break: break-word;
}

.impact code,
.detail code,
.file-list code,
td code {
  display: inline;
  white-space: normal;
  overflow-wrap: anywhere;
  word-break: break-word;
}

.code-diff-block pre,
.graph-block {
  max-width: 100%;
  overflow-x: auto;
}

.code-diff-block code,
.graph-block code {
  white-space: pre-wrap;
  overflow-wrap: anywhere;
  word-break: break-word;
}

table {
  table-layout: fixed;
  width: 100%;
}

@media (max-width: 900px) {
  .hero,
  .columns {
    display: block;
  }
  .grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
  .risk {
    margin-top: 16px;
    text-align: left;
  }
  .file-list {
    columns: 1;
  }
}
`
}
