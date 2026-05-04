package review

import (
	"fmt"
	"html"
	"io"
	"strings"

	"github.com/orurh/patchcourt/internal/reportmodel"
)

func WriteReviewHTML(w io.Writer, result reportmodel.ReviewResult) error {
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
	fmt.Fprintln(&b, `<h1>Review report</h1>`)
	fmt.Fprintln(&b, `<p class="muted">Diff-aware architecture review generated from deterministic project facts.</p>`)
	fmt.Fprintln(&b, `</div>`)
	fmt.Fprintf(&b, `<div class="risk risk-%s">`, htmlClass(string(result.Risk.Level)))
	fmt.Fprintf(&b, `<div class="risk-label">%s</div>`, escape(string(result.Risk.Level)))
	fmt.Fprintf(&b, `<div class="risk-points">%d points</div>`, result.Risk.Points)
	fmt.Fprintln(&b, `</div>`)
	fmt.Fprintln(&b, `</section>`)

	writeReviewHTMLSummary(&b, result)
	writeReviewHTMLImpact(&b, result.Impact)
	writeReviewHTMLChangedFiles(&b, "Changed files", result.ChangedFiles)
	writeReviewHTMLRiskReasons(&b, result)
	writeReviewHTMLCounts(&b, result)

	fmt.Fprintln(&b, "</main>")
	fmt.Fprintln(&b, "</body>")
	fmt.Fprintln(&b, "</html>")

	_, err := io.WriteString(w, b.String())
	return err
}

func writeReviewHTMLSummary(b *strings.Builder, result reportmodel.ReviewResult) {
	fmt.Fprintln(b, `<section class="grid">`)
	writeMetricCard(b, "Contract changes", result.Summary.ContractChanges)
	writeMetricCard(b, "Dependency changes", result.Summary.DependencyChanges)
	writeMetricCard(b, "Layer edge changes", result.Summary.LayerEdgeChanges)
	writeMetricCard(b, "Finding changes", result.Summary.FindingChanges)
	writeMetricCard(b, "Added findings", result.Summary.AddedFindings)
	writeMetricCard(b, "Removed findings", result.Summary.RemovedFindings)
	fmt.Fprintln(b, `</section>`)
}

func writeMetricCard(b *strings.Builder, title string, value int) {
	fmt.Fprintln(b, `<article class="card metric">`)
	fmt.Fprintf(b, `<div class="metric-value">%d</div>`, value)
	fmt.Fprintf(b, `<div class="metric-title">%s</div>`, escape(title))
	fmt.Fprintln(b, `</article>`)
}

func writeReviewHTMLImpact(b *strings.Builder, impact reportmodel.ReviewImpactReport) {
	fmt.Fprintln(b, `<section class="card">`)
	fmt.Fprintln(b, `<h2>Architecture impact</h2>`)
	fmt.Fprintln(b, `<div class="columns">`)
	writeReviewHTMLImpactColumn(b, "Worse", "bad", impact.Worse)
	writeReviewHTMLImpactColumn(b, "Better", "good", impact.Better)
	writeReviewHTMLImpactColumn(b, "Unchanged debt", "neutral", impact.UnchangedDebt)
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

func writeReviewHTMLChangedFiles(b *strings.Builder, title string, files []string) {
	fmt.Fprintln(b, `<section class="card">`)
	fmt.Fprintf(b, `<h2>%s</h2>\n`, escape(title))

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

func writeReviewHTMLRiskReasons(b *strings.Builder, result reportmodel.ReviewResult) {
	fmt.Fprintln(b, `<section class="card">`)
	fmt.Fprintln(b, `<h2>Risk reasons</h2>`)

	if len(result.Risk.Reasons) == 0 {
		fmt.Fprintln(b, `<p class="muted">No risk reasons.</p>`)
		fmt.Fprintln(b, `</section>`)
		return
	}

	fmt.Fprintln(b, `<ul>`)
	for _, reason := range result.Risk.Reasons {
		fmt.Fprintf(b, `<li><strong>+%d</strong> %s</li>`, reason.Points, escape(reason.Message))
	}
	fmt.Fprintln(b, `</ul>`)
	fmt.Fprintln(b, `</section>`)
}

func writeReviewHTMLCounts(b *strings.Builder, result reportmodel.ReviewResult) {
	fmt.Fprintln(b, `<section class="card">`)
	fmt.Fprintln(b, `<h2>Raw diff sections</h2>`)
	fmt.Fprintln(b, `<ul>`)
	fmt.Fprintf(b, `<li>Contract changes: <strong>%d</strong></li>\n`, len(result.ContractChanges))
	fmt.Fprintf(b, `<li>Dependency changes: <strong>%d</strong></li>\n`, len(result.DependencyChanges))
	fmt.Fprintf(b, `<li>Layer edge changes: <strong>%d</strong></li>\n`, len(result.LayerEdgeChanges))
	fmt.Fprintf(b, `<li>Finding changes: <strong>%d</strong></li>\n`, len(result.FindingChanges))
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
.file-list {
  columns: 2;
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
