package review

import (
	"fmt"
	"html"
	"io"
	"strings"

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
	fmt.Fprintln(b, `<div class="columns">`)
	for _, column := range columns {
		writeReviewHTMLImpactColumn(b, column.Title, column.Class, column.Items)
	}
	fmt.Fprintln(b, `</div>`)
	fmt.Fprintln(b, `</section>`)
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

	fmt.Fprintln(b, `<div class="table-wrap">`)
	fmt.Fprintln(b, `<table>`)
	fmt.Fprintln(b, `<thead><tr><th>Kind</th><th>Symbol</th><th>File</th><th>Before</th><th>After</th><th>Modifiers</th></tr></thead>`)
	fmt.Fprintln(b, `<tbody>`)

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

		fmt.Fprintln(b, `<tr>`)
		fmt.Fprintf(b, `<td><span class="tag">%s</span></td>`, escape(row.Kind))
		fmt.Fprintf(b, `<td><code>%s</code></td>`, escape(row.SymbolKey))
		fmt.Fprintf(b, `<td><code>%s</code></td>`, escape(row.File))
		fmt.Fprintf(b, `<td><code>%s</code></td>`, escape(row.BeforeSignature))
		fmt.Fprintf(b, `<td><code>%s</code></td>`, escape(row.AfterSignature))
		fmt.Fprintf(b, `<td>%s</td>`, escape(modifiers))
		fmt.Fprintln(b, `</tr>`)
	}

	fmt.Fprintln(b, `</tbody>`)
	fmt.Fprintln(b, `</table>`)
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
.impact {
  border: 1px solid var(--line);
  border-radius: 14px;
  padding: 16px;
}
.impact-bad h3 { color: var(--bad); }
.impact-good h3 { color: var(--good); }
.impact-neutral h3 { color: var(--neutral); }
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
