package report

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/orurh/patchcourt/internal/model"
)

type checkHTMLPayload struct {
	Root         string                 `json:"root"`
	ConfigPath   string                 `json:"config_path,omitempty"`
	OutDir       string                 `json:"out_dir"`
	Summary      model.ScanSummary      `json:"summary"`
	LayerGraph   any                    `json:"layer_graph"`
	Findings     []model.Finding        `json:"findings"`
	Dependencies []model.DependencyEdge `json:"dependencies"`
}

func WriteCheckHTML(w io.Writer, result CheckTextResult) error {
	payload := checkHTMLPayload{
		Root:       result.Root,
		ConfigPath: result.ConfigPath,
		OutDir:     result.OutDir,
		Summary:    result.Summary,
		LayerGraph: result.LayerGraph,
	}

	if result.Project != nil {
		payload.Findings = result.Project.Findings
		payload.Dependencies = result.Project.Dependencies
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal check html payload: %w", err)
	}

	jsonPayload := strings.ReplaceAll(string(data), "</script", "<\\/script")
	page := strings.Replace(checkHTMLTemplate, "__PATCHCOURT_DATA__", jsonPayload, 1)

	if _, err := io.WriteString(w, page); err != nil {
		return fmt.Errorf("write check html: %w", err)
	}

	return nil
}

const checkHTMLTemplate = `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <title>PatchCourt Report</title>
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <style>
    :root {
      --bg: #0b1020;
      --panel: #121a2f;
      --panel2: #18223b;
      --text: #e8eefc;
      --muted: #9aa7c7;
      --line: #2c3858;
      --accent: #8ab4ff;
      --danger: #ff6b6b;
      --warn: #ffb86b;
      --code: #0f172a;
    }

    * { box-sizing: border-box; }

    body {
      margin: 0;
      background: var(--bg);
      color: var(--text);
      font: 14px/1.5 system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
    }

    header {
      padding: 24px 28px;
      border-bottom: 1px solid var(--line);
      background: linear-gradient(135deg, #111a32, #0b1020);
    }

    h1, h2, h3 { margin: 0; }

    h1 {
      font-size: 26px;
      letter-spacing: -0.03em;
    }

    h2 {
      font-size: 17px;
      margin-bottom: 12px;
    }

    h3 {
      font-size: 12px;
      margin-bottom: 8px;
      color: var(--muted);
      text-transform: uppercase;
      letter-spacing: 0.08em;
    }

    .subtitle {
      margin-top: 6px;
      color: var(--muted);
    }

    .layout {
      display: grid;
      grid-template-columns: 360px 1fr;
      gap: 16px;
      padding: 16px;
    }

    .panel {
      background: var(--panel);
      border: 1px solid var(--line);
      border-radius: 14px;
      padding: 16px;
      overflow: hidden;
    }

    .stack {
      display: grid;
      gap: 16px;
      align-content: start;
    }

    .summary {
      display: grid;
      grid-template-columns: repeat(2, minmax(0, 1fr));
      gap: 8px;
    }

    .metric {
      background: var(--panel2);
      border: 1px solid var(--line);
      border-radius: 10px;
      padding: 10px;
    }

    .metric .value {
      font-size: 22px;
      font-weight: 700;
    }

    .metric .label {
      color: var(--muted);
      font-size: 12px;
    }

    .edge-list {
      display: grid;
      gap: 8px;
      max-height: 520px;
      overflow: auto;
    }

    .edge-row {
      width: 100%;
      text-align: left;
      cursor: pointer;
      color: var(--text);
      background: var(--panel2);
      border: 1px solid var(--line);
      border-radius: 10px;
      padding: 10px;
    }

    .edge-row:hover,
    .edge-row.selected {
      border-color: var(--accent);
      outline: 1px solid var(--accent);
    }

    .edge-title {
      display: flex;
      justify-content: space-between;
      gap: 12px;
      font-weight: 700;
    }

    .edge-meta {
      margin-top: 4px;
      color: var(--muted);
      font-size: 12px;
    }

    .badge {
      display: inline-block;
      border-radius: 999px;
      padding: 2px 8px;
      font-size: 12px;
      background: var(--panel2);
      border: 1px solid var(--line);
      color: var(--muted);
    }

    .badge.danger {
      color: var(--danger);
      border-color: rgba(255, 107, 107, 0.5);
    }

    .badge.warn {
      color: var(--warn);
      border-color: rgba(255, 184, 107, 0.5);
    }

    .finding {
      border-top: 1px solid var(--line);
      padding-top: 12px;
      margin-top: 12px;
    }

    .finding:first-child {
      border-top: 0;
      padding-top: 0;
      margin-top: 0;
    }

    .muted { color: var(--muted); }

    code, pre {
      font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
    }

    code {
      background: var(--code);
      border: 1px solid var(--line);
      border-radius: 6px;
      padding: 1px 5px;
    }

    pre {
      background: var(--code);
      border: 1px solid var(--line);
      border-radius: 10px;
      padding: 12px;
      overflow: auto;
      max-height: 520px;
      white-space: pre-wrap;
    }

    .columns {
      display: grid;
      grid-template-columns: 1fr 1fr;
      gap: 16px;
    }

    .small-list {
      display: grid;
      gap: 4px;
      margin: 0;
      padding: 0;
      list-style: none;
    }

    .small-list li {
      display: flex;
      justify-content: space-between;
      gap: 12px;
      border-bottom: 1px solid rgba(255,255,255,0.06);
      padding: 4px 0;
    }

    .empty {
      color: var(--muted);
      border: 1px dashed var(--line);
      border-radius: 10px;
      padding: 14px;
    }

    @media (max-width: 980px) {
      .layout { grid-template-columns: 1fr; }
      .columns { grid-template-columns: 1fr; }
    }
  </style>
</head>
<body>
  <header>
    <h1>PatchCourt Report</h1>
    <div class="subtitle" id="root"></div>
  </header>

  <main class="layout">
    <section class="stack">
      <div class="panel">
        <h2>Summary</h2>
        <div class="summary" id="summary"></div>
      </div>

      <div class="panel">
        <h2>Layer edges</h2>
        <div class="muted" style="margin-bottom: 10px;">Click an edge to inspect exact include dependencies.</div>
        <div class="edge-list" id="edges"></div>
      </div>
    </section>

    <section class="stack">
      <div class="panel">
        <h2>Selected edge</h2>
        <div id="edgeDetails" class="empty">Select an edge from the list.</div>
      </div>

      <div class="panel">
        <h2>All findings</h2>
        <div id="findings"></div>
      </div>
    </section>
  </main>

  <script type="application/json" id="patchcourt-data">__PATCHCOURT_DATA__</script>

  <script>
    const data = JSON.parse(document.getElementById("patchcourt-data").textContent);

    function get(obj, snake, camel, fallback) {
      if (fallback === undefined) fallback = "";
      if (!obj) return fallback;
      if (obj[snake] !== undefined && obj[snake] !== null) return obj[snake];
      if (obj[camel] !== undefined && obj[camel] !== null) return obj[camel];
      return fallback;
    }

    function edgeKey(from, to) {
      return String(from) + "->" + String(to);
    }

    function escapeHTML(value) {
      return String(value == null ? "" : value)
        .replaceAll("&", "&amp;")
        .replaceAll("<", "&lt;")
        .replaceAll(">", "&gt;")
        .replaceAll("\"", "&quot;")
        .replaceAll("'", "&#039;");
    }

    function findingEdgeIndex() {
      const index = new Map();

      function addFinding(key, finding) {
        if (!index.has(key)) {
          index.set(key, []);
        }

        const list = index.get(key);
        const id = get(finding, "id", "id");

        for (const existing of list) {
          if (get(existing, "id", "id") === id) {
            return;
          }
        }

        list.push(finding);
      }

      for (const finding of data.findings || []) {
        const id = get(finding, "id", "id");
        const evidence = get(finding, "evidence", "evidence", []);

        for (const item of evidence || []) {
          const message = get(item, "message", "message");
          const match = /dependency\s+([A-Za-z0-9_.:-]+)\s+->\s+([A-Za-z0-9_.:-]+)/.exec(message);
          if (!match) continue;

          addFinding(edgeKey(match[1], match[2]), finding);
        }

        const idMatch = /^architecture\.([A-Za-z0-9_.:-]+)\.([A-Za-z0-9_.:-]+)$/.exec(id);
        if (idMatch) {
          addFinding(edgeKey(idMatch[1], idMatch[2]), finding);
        }
      }

      return index;
    }

    const findingByEdge = findingEdgeIndex();

    function sortedEdges() {
      const edges = get(data.layer_graph, "edges", "edges", []);
      return Array.from(edges).sort(function(a, b) {
        const aKey = edgeKey(get(a, "from", "from"), get(a, "to", "to"));
        const bKey = edgeKey(get(b, "from", "from"), get(b, "to", "to"));
        const aSuspicious = findingByEdge.has(aKey) ? 1 : 0;
        const bSuspicious = findingByEdge.has(bKey) ? 1 : 0;

        if (aSuspicious !== bSuspicious) return bSuspicious - aSuspicious;

        const aCount = get(a, "count", "count", 0);
        const bCount = get(b, "count", "count", 0);
        if (aCount !== bCount) return bCount - aCount;

        return aKey.localeCompare(bKey);
      });
    }

    function renderSummary() {
      document.getElementById("root").textContent = data.root || "";

      const summary = data.summary || {};
      const findings = (data.findings || []).length;
      const nodes = get(data.layer_graph, "nodes", "nodes", []).length;
      const edges = get(data.layer_graph, "edges", "edges", []).length;

      const metrics = [
        ["Production files", get(summary, "production_files", "production_files", 0)],
        ["Test files", get(summary, "test_files", "test_files", 0)],
        ["Dependencies", get(summary, "total_edges", "total_edges", 0)],
        ["Resolved", get(summary, "resolved", "resolved", 0)],
        ["Findings", findings],
        ["Graph edges", edges],
        ["Graph nodes", nodes],
        ["Symbols", get(summary, "symbols", "symbols", 0)]
      ];

      document.getElementById("summary").innerHTML = metrics.map(function(item) {
        return "<div class=\"metric\">" +
          "<div class=\"value\">" + escapeHTML(item[1]) + "</div>" +
          "<div class=\"label\">" + escapeHTML(item[0]) + "</div>" +
          "</div>";
      }).join("");
    }

    function renderEdges() {
      const edges = sortedEdges();
      const container = document.getElementById("edges");

      if (edges.length === 0) {
        container.innerHTML = "<div class=\"empty\">No layer edges.</div>";
        return;
      }

      container.innerHTML = edges.map(function(edge, index) {
        const from = get(edge, "from", "from");
        const to = get(edge, "to", "to");
        const count = get(edge, "count", "count", 0);
        const key = edgeKey(from, to);
        const findings = findingByEdge.get(key) || [];
        const badge = findings.length > 0
          ? "<span class=\"badge warn\">" + findings.length + " finding" + (findings.length === 1 ? "" : "s") + "</span>"
          : "<span class=\"badge\">" + count + " deps</span>";

        return "<button class=\"edge-row\" data-index=\"" + index + "\" data-from=\"" + escapeHTML(from) + "\" data-to=\"" + escapeHTML(to) + "\">" +
          "<div class=\"edge-title\">" +
          "<span>" + escapeHTML(from) + " → " + escapeHTML(to) + "</span>" +
          "<span>" + escapeHTML(count) + "</span>" +
          "</div>" +
          "<div class=\"edge-meta\">" + badge + "</div>" +
          "</button>";
      }).join("");

      for (const button of container.querySelectorAll(".edge-row")) {
        button.addEventListener("click", function() {
          selectEdge(button.dataset.from, button.dataset.to);
        });
      }
    }

    function dependencyTarget(dep) {
      return get(dep, "to_file", "toFile") || get(dep, "target", "target");
    }

    function edgeDependencies(from, to) {
      return (data.dependencies || [])
        .filter(function(dep) {
          return get(dep, "resolved", "resolved", false) === true &&
                 get(dep, "external", "external", false) !== true &&
                 get(dep, "from_layer", "fromLayer") === from &&
                 get(dep, "to_layer", "toLayer") === to;
        })
        .sort(function(a, b) {
          const af = get(a, "from_file", "fromFile");
          const bf = get(b, "from_file", "fromFile");
          if (af !== bf) return af.localeCompare(bf);
          return dependencyTarget(a).localeCompare(dependencyTarget(b));
        });
    }

    function countBy(items, fn) {
      const counts = new Map();
      for (const item of items) {
        const key = fn(item);
        if (!key) continue;
        counts.set(key, (counts.get(key) || 0) + 1);
      }

      return Array.from(counts.entries())
        .sort(function(a, b) {
          return b[1] - a[1] || a[0].localeCompare(b[0]);
        })
        .slice(0, 10);
    }

    function usageSummary(deps) {
      const usage = { used: 0, maybe: 0, unused: 0, unknown: 0 };
      for (const dep of deps) {
        const value = get(dep, "usage", "usage", "unknown") || "unknown";
        if (usage[value] === undefined) usage.unknown++;
        else usage[value]++;
      }
      return usage;
    }

    function renderCountList(rows) {
      if (rows.length === 0) return "<div class=\"empty\">none</div>";

      return "<ul class=\"small-list\">" + rows.map(function(row) {
        return "<li><span><code>" + escapeHTML(row[0]) + "</code></span><strong>" + escapeHTML(row[1]) + "</strong></li>";
      }).join("") + "</ul>";
    }

    function renderDependencies(deps) {
      if (deps.length === 0) return "No dependencies for this edge.";

      const lines = [];
      let current = "";

      for (const dep of deps) {
        const from = get(dep, "from_file", "fromFile");
        const target = dependencyTarget(dep);
        const usage = get(dep, "usage", "usage", "unknown");

        if (from !== current) {
          current = from;
          lines.push("");
          lines.push(from);
        }

        lines.push("  -> " + target + " [" + usage + "]");
      }

      return lines.join("\n").trim();
    }

    function selectEdge(from, to) {
      for (const row of document.querySelectorAll(".edge-row")) {
        row.classList.toggle("selected", row.dataset.from === from && row.dataset.to === to);
      }

      const key = edgeKey(from, to);
      const deps = edgeDependencies(from, to);
      const findings = findingByEdge.get(key) || [];
      const usage = usageSummary(deps);
      const topFrom = countBy(deps, function(dep) { return get(dep, "from_file", "fromFile"); });
      const topTo = countBy(deps, function(dep) { return dependencyTarget(dep); });

      const details = document.getElementById("edgeDetails");
      details.classList.remove("empty");
      details.innerHTML =
        "<h3>Edge</h3>" +
        "<div style=\"font-size: 22px; font-weight: 800; margin-bottom: 12px;\">" +
          escapeHTML(from) + " → " + escapeHTML(to) +
        "</div>" +
        "<div style=\"display: flex; gap: 8px; flex-wrap: wrap; margin-bottom: 16px;\">" +
          "<span class=\"badge\">" + deps.length + " dependencies</span>" +
          "<span class=\"badge\">used " + usage.used + "</span>" +
          "<span class=\"badge\">maybe " + usage.maybe + "</span>" +
          "<span class=\"badge\">unused " + usage.unused + "</span>" +
          "<span class=\"badge\">unknown " + usage.unknown + "</span>" +
          (findings.length > 0 ? "<span class=\"badge warn\">" + findings.length + " related finding" + (findings.length === 1 ? "" : "s") + "</span>" : "") +
        "</div>" +
        "<div class=\"columns\">" +
          "<div><h3>Top source files</h3>" + renderCountList(topFrom) + "</div>" +
          "<div><h3>Top target files</h3>" + renderCountList(topTo) + "</div>" +
        "</div>" +
        "<div style=\"margin-top: 16px;\">" +
          "<h3>Related findings</h3>" +
          (findings.length === 0 ? "<div class=\"empty\">No findings attached to this edge.</div>" : findings.map(renderFinding).join("")) +
        "</div>" +
        "<div style=\"margin-top: 16px;\">" +
          "<h3>Dependencies</h3>" +
          "<pre>" + escapeHTML(renderDependencies(deps)) + "</pre>" +
        "</div>";
    }

    function renderFinding(finding) {
      const id = get(finding, "id", "id");
      const severity = get(finding, "severity", "severity");
      const kind = get(finding, "kind", "kind");
      const title = get(finding, "title", "title");
      const risk = get(finding, "risk", "risk");
      const suggestion = get(finding, "suggestion", "suggestion");
      const dangerClass = severity === "high" || severity === "critical" ? "danger" : "warn";

      return "<div class=\"finding\">" +
        "<div><span class=\"badge " + dangerClass + "\">" + escapeHTML(severity) + "/" + escapeHTML(kind) + "</span></div>" +
        "<div style=\"margin-top: 6px; font-weight: 700;\">" + escapeHTML(id || title) + "</div>" +
        (title && id ? "<div class=\"muted\">" + escapeHTML(title) + "</div>" : "") +
        (risk ? "<p>" + escapeHTML(risk) + "</p>" : "") +
        (suggestion ? "<p class=\"muted\"><strong>Suggestion:</strong> " + escapeHTML(suggestion) + "</p>" : "") +
        "</div>";
    }

    function renderFindings() {
      const container = document.getElementById("findings");
      const findings = data.findings || [];

      if (findings.length === 0) {
        container.innerHTML = "<div class=\"empty\">No findings.</div>";
        return;
      }

      container.innerHTML = findings.slice(0, 20).map(renderFinding).join("") +
        (findings.length > 20 ? "<div class=\"muted\" style=\"margin-top: 12px;\">... " + (findings.length - 20) + " more findings</div>" : "");
    }

    renderSummary();
    renderEdges();
    renderFindings();

    const edges = sortedEdges();
    const firstSuspicious = edges.find(function(edge) {
      return findingByEdge.has(edgeKey(get(edge, "from", "from"), get(edge, "to", "to")));
    });
    const firstEdge = firstSuspicious || edges[0];

    if (firstEdge) {
      selectEdge(get(firstEdge, "from", "from"), get(firstEdge, "to", "to"));
    }
  </script>
</body>
</html>
`
