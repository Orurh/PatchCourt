package report

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/orurh/patchcourt/internal/model"
	"github.com/orurh/patchcourt/internal/reportmodel"
)

type CheckHTMLInput struct {
	Report     reportmodel.CheckReport
	Project    *model.ProjectModel
	LayerGraph any
}

type checkHTMLPayload struct {
	Report       reportmodel.CheckReport `json:"report"`
	Root         string                  `json:"root"`
	ConfigPath   string                  `json:"config_path,omitempty"`
	OutDir       string                  `json:"out_dir"`
	Summary      model.ScanSummary       `json:"summary"`
	LayerGraph   any                     `json:"layer_graph"`
	Findings     []model.Finding         `json:"findings"`
	Dependencies []model.DependencyEdge  `json:"dependencies"`
}

func WriteCheckHTML(w io.Writer, input CheckHTMLInput) error {
	payload := checkHTMLPayload{
		Report:     input.Report,
		Root:       input.Report.Root,
		ConfigPath: input.Report.ConfigPath,
		OutDir:     input.Report.OutDir,
		Summary:    input.Report.Summary,
		LayerGraph: input.LayerGraph,
	}

	if input.Project != nil {
		payload.Findings = input.Project.Findings
		payload.Dependencies = input.Project.Dependencies
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
      --edge: #94a3b8;
      --edge-hot: #f97316;
      --edge-danger: #dc2626;
      --node: #1f2a44;
      --node-border: #3a4a72;
      --svg-bg: #0d1528;
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
      grid-template-columns: 380px 1fr;
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

    .filters {
      display: grid;
      gap: 8px;
      margin-bottom: 12px;
    }

    .filter-row {
      display: grid;
      grid-template-columns: 1fr 96px;
      gap: 8px;
    }

    input[type="search"],
    input[type="number"] {
      width: 100%;
      color: var(--text);
      background: var(--code);
      border: 1px solid var(--line);
      border-radius: 10px;
      padding: 9px 10px;
      outline: none;
    }

    input[type="search"]:focus,
    input[type="number"]:focus {
      border-color: var(--accent);
    }

    .check-row {
      display: flex;
      flex-wrap: wrap;
      gap: 12px;
      color: var(--muted);
      font-size: 13px;
    }

    .edge-list {
      display: grid;
      gap: 8px;
      max-height: 560px;
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

    .shell {
      max-width: 100vw;
      height: calc(100vh - 56px);
      padding: 10px;
      display: grid;
      grid-template-rows: 120px minmax(0, 1fr);
      gap: 10px;
      overflow: hidden;
    }

    .topbar {
      display: grid;
      grid-template-columns: minmax(620px, 1.2fr) minmax(460px, 0.8fr);
      gap: 10px;
      min-height: 0;
      overflow: hidden;
    }

    .summary-panel,
    .filters-panel {
      padding: 10px 12px;
      overflow: hidden;
    }

    .summary-panel h2,
    .filters-panel h2 {
      margin-bottom: 8px;
      font-size: 15px;
    }

    .summary {
      grid-template-columns: repeat(8, minmax(88px, 1fr));
      gap: 8px;
    }

    .stat {
      padding: 8px 10px;
      min-height: 54px;
    }

    .stat strong {
      font-size: 20px;
      line-height: 1.1;
    }

    .stat span {
      font-size: 11px;
    }

    .compact-filters {
      margin-top: 0;
    }

    .compact-filters .filter-row {
      margin-bottom: 8px;
    }

    .compact-filters input[type="search"] {
      height: 36px;
    }

    .compact-filters input[type="number"] {
      height: 36px;
      width: 86px;
    }

    .workspace {
      min-height: 0;
      display: grid;
      grid-template-columns: 340px minmax(560px, 1fr) 420px;
      gap: 12px;
      align-items: stretch;
    }

    .edges-pane,
    .graph-pane,
    .details-pane,
    .findings-pane {
      min-height: 0;
      overflow: hidden;
      display: flex;
      flex-direction: column;
    }

    .side-stack {
      min-height: 0;
      display: grid;
      grid-template-rows: minmax(280px, 1.05fr) minmax(180px, 0.95fr);
      gap: 12px;
      overflow: hidden;
    }

    .edge-list,
    #edgeDetails,
    #findings {
      overflow: auto;
      min-height: 0;
    }

    .edge-list {
      padding-right: 4px;
    }

    .small-help {
      margin-bottom: 10px;
      font-size: 12px;
    }

    .dependency-list {
      display: grid;
      gap: 8px;
    }

    .dependency-card {
      border: 1px solid var(--line);
      background: rgba(255, 255, 255, 0.025);
      border-radius: 10px;
      padding: 9px 10px;
      overflow-wrap: anywhere;
    }

    .dependency-card .dep-kind {
      color: var(--muted);
      font-size: 11px;
      margin-bottom: 4px;
    }

    .dependency-card .dep-path {
      font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
      font-size: 12px;
      line-height: 1.35;
      white-space: normal;
      overflow-wrap: anywhere;
      word-break: break-word;
    }

    .details-pane,
    #edgeDetails,
    .dependency-list,
    .dependency-card {
      min-width: 0;
      max-width: 100%;
      overflow-x: hidden;
    }

    .dependency-card .dep-arrow {
      color: var(--accent);
      margin: 2px 0;
    }

    .panel-heading {
      display: flex;
      justify-content: space-between;
      gap: 12px;
      align-items: start;
      margin-bottom: 10px;
      flex: 0 0 auto;
    }

    .legend {
      display: flex;
      gap: 10px;
      flex-wrap: wrap;
      color: var(--muted);
      font-size: 12px;
      white-space: nowrap;
    }

    .dot {
      display: inline-block;
      width: 9px;
      height: 9px;
      border-radius: 999px;
      margin-right: 5px;
      vertical-align: middle;
    }

    .dot.normal { background: #94a3b8; }
    .dot.warn { background: #f97316; }
    .dot.danger { background: #dc2626; }

    .overview-wrap {
      border: 1px solid var(--line);
      border-radius: 14px;
      overflow: auto;
      background: var(--svg-bg);
      min-height: 0;
      flex: 1 1 auto;
    }

    #overviewGraph {
      display: block;
      width: 100%;
      height: 100%;
      min-width: 760px;
      min-height: 480px;
      background: var(--svg-bg);
    }

    .overview-edge {
      cursor: pointer;
    }

    .overview-edge path {
      stroke-linecap: round;
    }

    .overview-edge:hover path,
    .overview-edge.selected path {
      filter: drop-shadow(0 0 4px rgba(138, 180, 255, 0.65));
    }

    .overview-node {
      cursor: default;
    }

    .svg-wrap {
      margin-top: 16px;
      border: 1px solid var(--line);
      border-radius: 12px;
      overflow: auto;
      background: var(--svg-bg);
    }

    .svg-caption {
      margin-top: 8px;
      color: var(--muted);
      font-size: 12px;
    }

    #edgeDiagram {
      display: block;
      width: 100%;
      min-height: 280px;
      background: var(--svg-bg);
    }

    @media (max-width: 1200px) {
      .shell {
        height: auto;
        overflow: visible;
      }

      .topbar,
      .workspace {
        grid-template-columns: 1fr;
      }

      .side-stack {
        grid-template-rows: auto auto;
      }

      .edges-pane,
      .graph-pane,
      .details-pane,
      .findings-pane {
        max-height: none;
        overflow: visible;
      }

      .edge-list,
      #edgeDetails,
      #findings {
        max-height: 420px;
      }

      .overview-wrap {
        height: 520px;
      }

      .columns {
        grid-template-columns: 1fr;
      }
    }

      .columns {
        grid-template-columns: 1fr;
      }

      .overview-wrap {
        max-height: none;
      }
    }
  </style>
</head>
<body>
  <header>
    <h1>PatchCourt Report</h1>
    <div class="subtitle" id="root"></div>
  </header>

  <main class="shell">
    <section class="topbar">
      <div class="panel summary-panel">
        <h2>Summary</h2>
        <div class="summary" id="summary"></div>
      </div>

      <div class="panel filters-panel">
        <h2>Filters</h2>
        <div class="filters compact-filters">
          <div class="filter-row">
            <input id="edgeSearch" type="search" placeholder="Search layer, file, finding...">
            <input id="minEdgeCount" type="number" min="0" value="0" title="Minimum edge count">
          </div>

          <div class="check-row">
            <label>
              <input id="onlyFindings" type="checkbox">
              only edges with findings
            </label>
            <label>
              <input id="onlyPolicy" type="checkbox">
              only policy violations
            </label>
          </div>
        </div>
      </div>
    </section>

    <section class="workspace">
      <aside class="panel edges-pane">
        <h2>Layer edges</h2>
        <div class="muted small-help">Click an edge. The graph and details update together.</div>
        <div class="edge-list" id="edges"></div>
      </aside>

      <section class="panel graph-pane">
        <div class="panel-heading">
          <div>
            <h2>Layer graph</h2>
            <div class="muted small-help">Click an arrow to inspect exact include/import dependencies.</div>
          </div>
          <div class="legend">
            <span><b class="dot normal"></b> normal</span>
            <span><b class="dot warn"></b> finding</span>
            <span><b class="dot danger"></b> policy</span>
          </div>
        </div>
        <div class="overview-wrap" id="overviewGraphWrap"></div>
      </section>

      <aside class="side-stack">
        <section class="panel details-pane">
          <h2>Selected edge</h2>
          <div id="edgeDetails" class="empty">Select an edge from the graph or from the list.</div>
        </section>

        <section class="panel findings-pane">
          <h2>All findings</h2>
          <div id="findings"></div>
        </section>
      </aside>
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

    function truncateText(text, max) {
      text = String(text || "");
      if (text.length <= max) return text;
      return text.slice(0, Math.max(0, max - 1)) + "…";
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
          const fromLayer = get(item, "from_layer", "fromLayer");
          const toLayer = get(item, "to_layer", "toLayer");

          if (fromLayer && toLayer) {
            addFinding(edgeKey(fromLayer, toLayer), finding);
            continue;
          }

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
    let selectedEdge = null;

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

    function edgeFilterState() {
      const search = document.getElementById("edgeSearch");
      const minCount = document.getElementById("minEdgeCount");
      const onlyFindings = document.getElementById("onlyFindings");
      const onlyPolicy = document.getElementById("onlyPolicy");

      return {
        search: search ? search.value.trim().toLowerCase() : "",
        minCount: minCount ? Number(minCount.value || 0) : 0,
        onlyFindings: onlyFindings ? onlyFindings.checked : false,
        onlyPolicy: onlyPolicy ? onlyPolicy.checked : false
      };
    }

    function findingIsPolicyViolation(finding) {
      const kind = get(finding, "kind", "kind");
      const id = get(finding, "id", "id");
      return kind === "policy_violation" || String(id).startsWith("architecture.");
    }

    function edgeHasPolicyViolation(from, to) {
      const findings = findingByEdge.get(edgeKey(from, to)) || [];
      return findings.some(findingIsPolicyViolation);
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

    function edgeSearchText(edge, deps, findings) {
      const parts = [
        get(edge, "from", "from"),
        get(edge, "to", "to")
      ];

      for (const dep of deps) {
        parts.push(get(dep, "from_file", "fromFile"));
        parts.push(dependencyTarget(dep));
      }

      for (const finding of findings) {
        parts.push(get(finding, "id", "id"));
        parts.push(get(finding, "title", "title"));
        parts.push(get(finding, "kind", "kind"));
        parts.push(get(finding, "severity", "severity"));
      }

      return parts.join(" ").toLowerCase();
    }

    function edgeMatchesFilters(edge) {
      const from = get(edge, "from", "from");
      const to = get(edge, "to", "to");
      const count = get(edge, "count", "count", 0);
      const key = edgeKey(from, to);
      const findings = findingByEdge.get(key) || [];
      const deps = edgeDependencies(from, to);
      const filters = edgeFilterState();

      if (count < filters.minCount) {
        return false;
      }

      if (filters.onlyFindings && findings.length === 0) {
        return false;
      }

      if (filters.onlyPolicy && !edgeHasPolicyViolation(from, to)) {
        return false;
      }

      if (filters.search !== "" && !edgeSearchText(edge, deps, findings).includes(filters.search)) {
        return false;
      }

      return true;
    }

    function overviewNodes() {
      const nodeSet = new Set();

      for (const node of get(data.layer_graph, "nodes", "nodes", [])) {
        if (node) nodeSet.add(String(node));
      }

      for (const edge of get(data.layer_graph, "edges", "edges", [])) {
        const from = get(edge, "from", "from");
        const to = get(edge, "to", "to");
        if (from) nodeSet.add(String(from));
        if (to) nodeSet.add(String(to));
      }

      return Array.from(nodeSet);
    }

    function orderedOverviewNodes(nodes) {
      const preferred = [
        "coroutines",
        "main_cc",
        "entrypoint",
        "application",
        "controllers",
        "server",
        "cameras",
        "configs",
        "utils",
        "domain",
        "session"
      ];

      const index = new Map();
      preferred.forEach(function(name, i) {
        index.set(name, i);
      });

      return Array.from(nodes).sort(function(a, b) {
        const ai = index.has(a) ? index.get(a) : 1000;
        const bi = index.has(b) ? index.get(b) : 1000;

        if (ai !== bi) return ai - bi;
        return a.localeCompare(b);
      });
    }

    function overviewNodePositions(nodes, width, height) {
      const positions = new Map();
      const fixed = {
        "coroutines": [120, 110],
        "main_cc": [120, 280],
        "entrypoint": [120, 280],
        "application": [310, 280],
        "controllers": [500, 260],
        "server": [680, 420],
        "cameras": [690, 140],
        "configs": [900, 110],
        "utils": [860, 400],
        "domain": [1010, 320],
        "session": [1080, 455]
      };

      const used = new Set();
      for (const node of nodes) {
        if (fixed[node]) {
          positions.set(node, { x: fixed[node][0], y: fixed[node][1] });
          used.add(node);
        }
      }

      const unknown = nodes.filter(function(node) {
        return !used.has(node);
      });

      const cx = width / 2;
      const cy = height / 2 + 8;
      const rx = Math.max(250, width / 2 - 120);
      const ry = Math.max(110, height / 2 - 90);

      unknown.forEach(function(node, i) {
        const angle = -Math.PI / 2 + (2 * Math.PI * i / Math.max(unknown.length, 1));
        positions.set(node, {
          x: cx + Math.cos(angle) * rx,
          y: cy + Math.sin(angle) * ry
        });
      });

      return positions;
    }

    function overviewEdgeColor(from, to) {
      if (edgeHasPolicyViolation(from, to)) {
        return "#dc2626";
      }

      if ((findingByEdge.get(edgeKey(from, to)) || []).length > 0) {
        return "#f97316";
      }

      return "#94a3b8";
    }

    function overviewEdgeLabel(edge) {
      const from = get(edge, "from", "from");
      const to = get(edge, "to", "to");
      const count = get(edge, "count", "count", 0);
      const hasFinding = (findingByEdge.get(edgeKey(from, to)) || []).length > 0;

      if (hasFinding) {
        return String(count) + " ⚠";
      }

      return String(count);
    }

    function overviewEdgeIsSelected(from, to) {
      return selectedEdge &&
        selectedEdge.from === from &&
        selectedEdge.to === to;
    }

    function renderOverviewGraph() {
      const wrap = document.getElementById("overviewGraphWrap");
      if (!wrap) return;

      const rawNodes = orderedOverviewNodes(overviewNodes());
      const edges = get(data.layer_graph, "edges", "edges", []);

      if (rawNodes.length === 0) {
        wrap.innerHTML = "<div class=\"empty\">No layer graph.</div>";
        return;
      }

      const width = 980;
      const height = 430;
      const nodeWidth = 150;
      const nodeHeight = 46;
      const pos = overviewNodePositions(rawNodes, width, height);
      const maxCount = edges.reduce(function(max, edge) {
        return Math.max(max, Number(get(edge, "count", "count", 0)));
      }, 1);

      const parts = [];
      parts.push('<svg id="overviewGraph" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 ' + width + ' ' + height + '">');
      parts.push('<defs>');
      parts.push('<marker id="overviewArrow" markerWidth="10" markerHeight="10" refX="8" refY="3" orient="auto" markerUnits="strokeWidth"><path d="M0,0 L0,6 L9,3 z" fill="#94a3b8"></path></marker>');
      parts.push('<marker id="overviewArrowWarn" markerWidth="10" markerHeight="10" refX="8" refY="3" orient="auto" markerUnits="strokeWidth"><path d="M0,0 L0,6 L9,3 z" fill="#f97316"></path></marker>');
      parts.push('<marker id="overviewArrowDanger" markerWidth="10" markerHeight="10" refX="8" refY="3" orient="auto" markerUnits="strokeWidth"><path d="M0,0 L0,6 L9,3 z" fill="#dc2626"></path></marker>');
      parts.push('</defs>');
      parts.push('<rect x="0" y="0" width="' + width + '" height="' + height + '" fill="#0d1528"></rect>');

      edges.forEach(function(edge, i) {
        const from = get(edge, "from", "from");
        const to = get(edge, "to", "to");
        const src = pos.get(from);
        const dst = pos.get(to);

        if (!src || !dst) return;

        const count = Number(get(edge, "count", "count", 0));
        const findings = findingByEdge.get(edgeKey(from, to)) || [];
        const policy = edgeHasPolicyViolation(from, to);
        const color = overviewEdgeColor(from, to);
        const marker = policy ? "overviewArrowDanger" : (findings.length > 0 ? "overviewArrowWarn" : "overviewArrow");
        const selected = overviewEdgeIsSelected(from, to);
        const strokeWidth = selected
          ? 4.5 + Math.min(4.5, Math.log2(count + 1))
          : 1.4 + Math.min(5.5, Math.log2(count + 1));

        const x1 = src.x + (dst.x >= src.x ? nodeWidth / 2 : -nodeWidth / 2);
        const y1 = src.y;
        const x2 = dst.x + (dst.x >= src.x ? -nodeWidth / 2 : nodeWidth / 2);
        const y2 = dst.y;

        const midX = (x1 + x2) / 2;
        const midY = (y1 + y2) / 2;
        const spread = ((i % 9) - 4) * 11;
        const verticalBias = Math.abs(x2 - x1) < 70 ? 80 : 0;
        const cx = midX;
        const cy = midY + spread - verticalBias;

        const labelX = midX;
        const labelY = midY + spread - 7;
        const className = selected ? "overview-edge selected" : "overview-edge";
        const opacity = findings.length > 0 || selected ? "0.96" : "0.58";
        const label = overviewEdgeLabel(edge);

        parts.push(
          '<g class="' + className + '" data-from="' + escapeHTML(from) + '" data-to="' + escapeHTML(to) + '">' +
            '<title>' + escapeHTML(from + " -> " + to + ": " + count + " dependencies") + '</title>' +
            '<path d="M ' + x1.toFixed(1) + ' ' + y1.toFixed(1) +
              ' Q ' + cx.toFixed(1) + ' ' + cy.toFixed(1) + ', ' + x2.toFixed(1) + ' ' + y2.toFixed(1) + '"' +
              ' fill="none" stroke="transparent" stroke-width="22"></path>' +
            '<path d="M ' + x1.toFixed(1) + ' ' + y1.toFixed(1) +
              ' Q ' + cx.toFixed(1) + ' ' + cy.toFixed(1) + ', ' + x2.toFixed(1) + ' ' + y2.toFixed(1) + '"' +
              ' fill="none" stroke="' + color + '" stroke-width="' + strokeWidth.toFixed(2) + '"' +
              ' opacity="' + opacity + '" marker-end="url(#' + marker + ')"></path>' +
            '<rect x="' + (labelX - 18).toFixed(1) + '" y="' + (labelY - 15).toFixed(1) + '" width="36" height="19" rx="9" fill="#0d1528" opacity="0.86"></rect>' +
            '<text x="' + labelX.toFixed(1) + '" y="' + labelY.toFixed(1) + '" fill="' + color + '" font-size="13" text-anchor="middle">' + escapeHTML(label) + '</text>' +
          '</g>'
        );
      });

      rawNodes.forEach(function(node) {
        const p = pos.get(node);
        if (!p) return;

        const connected = edges.some(function(edge) {
          return get(edge, "from", "from") === node || get(edge, "to", "to") === node;
        });

        const fill = connected ? "#1f2a44" : "#172033";
        const stroke = connected ? "#3a4a72" : "#6b7280";
        const dash = connected ? "" : ' stroke-dasharray="7 5"';
        const textColor = connected ? "#e8eefc" : "#9aa7c7";
        const label = truncateText(node, 17);

        parts.push(
          '<g class="overview-node">' +
            '<ellipse cx="' + p.x + '" cy="' + p.y + '" rx="' + (nodeWidth / 2) + '" ry="' + (nodeHeight / 2) + '" fill="' + fill + '" stroke="' + stroke + '"' + dash + '></ellipse>' +
            '<text x="' + p.x + '" y="' + (p.y + 4) + '" fill="' + textColor + '" font-size="13" text-anchor="middle">' + escapeHTML(label) + '</text>' +
          '</g>'
        );
      });

      parts.push('</svg>');
      wrap.innerHTML = parts.join("");

      for (const edge of wrap.querySelectorAll(".overview-edge")) {
        edge.addEventListener("click", function() {
          selectEdge(edge.dataset.from, edge.dataset.to);
        });
      }
    }

    function renderSummary() {
      document.getElementById("root").textContent = data.root || "";

      const summary = data.summary || {};
      const findings = (data.findings || []).length;
      const nodes = get(data.layer_graph, "nodes", "nodes", []).length;
      const edges = get(data.layer_graph, "edges", "edges", []).length;

      const metrics = [
        ["Production files", get(summary, "production_files", "productionFiles", 0)],
        ["Test files", get(summary, "test_files", "testFiles", 0)],
        ["Dependencies", get(summary, "total_edges", "totalEdges", 0)],
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
      const allEdges = sortedEdges();
      const edges = allEdges.filter(edgeMatchesFilters);
      const container = document.getElementById("edges");

      if (edges.length === 0) {
        container.innerHTML = "<div class=\"empty\">No layer edges match current filters.</div>";
        document.getElementById("edgeDetails").className = "empty";
        document.getElementById("edgeDetails").textContent = "No edge selected.";
        return;
      }

      container.innerHTML = edges.map(function(edge, index) {
        const from = get(edge, "from", "from");
        const to = get(edge, "to", "to");
        const count = get(edge, "count", "count", 0);
        const key = edgeKey(from, to);
        const findings = findingByEdge.get(key) || [];
        const hasPolicy = edgeHasPolicyViolation(from, to);
        const badge = findings.length > 0
          ? "<span class=\"badge " + (hasPolicy ? "danger" : "warn") + "\">" + findings.length + " finding" + (findings.length === 1 ? "" : "s") + "</span>"
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

      const first = edges[0];
      selectEdge(get(first, "from", "from"), get(first, "to", "to"));
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

    function fullCountBy(items, fn) {
      const counts = new Map();
      for (const item of items) {
        const key = fn(item);
        if (!key) continue;
        counts.set(key, (counts.get(key) || 0) + 1);
      }

      return Array.from(counts.entries())
        .sort(function(a, b) {
          return b[1] - a[1] || a[0].localeCompare(b[0]);
        });
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

    function buildEdgeDiagramData(deps) {
      const maxSide = 8;
      const sourceCounts = fullCountBy(deps, function(dep) { return get(dep, "from_file", "fromFile"); });
      const targetCounts = fullCountBy(deps, function(dep) { return dependencyTarget(dep); });

      const sourceTop = sourceCounts.slice(0, maxSide);
      const targetTop = targetCounts.slice(0, maxSide);

      const sourceKeep = new Set(sourceTop.map(function(row) { return row[0]; }));
      const targetKeep = new Set(targetTop.map(function(row) { return row[0]; }));

      const otherSourceCount = sourceCounts.slice(maxSide).reduce(function(sum, row) { return sum + row[1]; }, 0);
      const otherTargetCount = targetCounts.slice(maxSide).reduce(function(sum, row) { return sum + row[1]; }, 0);

      const sourceNodes = sourceTop.map(function(row) {
        return { id: row[0], label: row[0], count: row[1], side: "left", other: false };
      });

      const targetNodes = targetTop.map(function(row) {
        return { id: row[0], label: row[0], count: row[1], side: "right", other: false };
      });

      if (otherSourceCount > 0) {
        sourceNodes.push({
          id: "__other_sources__",
          label: "other sources",
          count: otherSourceCount,
          side: "left",
          other: true
        });
      }

      if (otherTargetCount > 0) {
        targetNodes.push({
          id: "__other_targets__",
          label: "other targets",
          count: otherTargetCount,
          side: "right",
          other: true
        });
      }

      const edgeMap = new Map();

      for (const dep of deps) {
        let from = get(dep, "from_file", "fromFile");
        let to = dependencyTarget(dep);

        if (!sourceKeep.has(from)) {
          from = "__other_sources__";
        }

        if (!targetKeep.has(to)) {
          to = "__other_targets__";
        }

        const key = from + "=>" + to;
        const current = edgeMap.get(key) || { from: from, to: to, count: 0 };
        current.count += 1;
        edgeMap.set(key, current);
      }

      return {
        sources: sourceNodes,
        targets: targetNodes,
        links: Array.from(edgeMap.values()).sort(function(a, b) {
          return b.count - a.count || a.from.localeCompare(b.from) || a.to.localeCompare(b.to);
        })
      };
    }

    function edgeDiagramSVG(fromLayer, toLayer, deps, findings) {
      const svgWidth = 900;
      const data = buildEdgeDiagramData(deps);
      const rows = Math.max(data.sources.length, data.targets.length, 1);
      const rowHeight = 58;
      const topPad = 86;
      const bottomPad = 40;
      const svgHeight = Math.max(260, topPad + rows * rowHeight + bottomPad);

      const sourceX = 36;
      const targetX = 594;
      const nodeWidth = 270;
      const nodeHeight = 36;
      const sourceTextX = sourceX + nodeWidth / 2;
      const targetTextX = targetX + nodeWidth / 2;
      const hasPolicy = findings.some(findingIsPolicyViolation);
      const edgeColor = hasPolicy ? "#dc2626" : (findings.length > 0 ? "#f97316" : "#94a3b8");
      const nodeFill = "#1f2a44";
      const nodeStroke = "#3a4a72";
      const textColor = "#e8eefc";
      const mutedColor = "#9aa7c7";
      const title = fromLayer + " → " + toLayer;
      const subtitle = deps.length + " dependencies · " + data.sources.length + " source nodes · " + data.targets.length + " target nodes";

      function nodeY(index) {
        return topPad + index * rowHeight;
      }

      const sourcePos = new Map();
      const targetPos = new Map();

      data.sources.forEach(function(node, i) {
        sourcePos.set(node.id, { x: sourceX, y: nodeY(i) });
      });

      data.targets.forEach(function(node, i) {
        targetPos.set(node.id, { x: targetX, y: nodeY(i) });
      });

      const maxLinkCount = data.links.reduce(function(max, link) {
        return Math.max(max, link.count);
      }, 1);

      const parts = [];
      parts.push('<svg id="edgeDiagram" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 ' + svgWidth + ' ' + svgHeight + '">');
      parts.push('<rect x="0" y="0" width="' + svgWidth + '" height="' + svgHeight + '" fill="#0d1528"></rect>');

      parts.push('<text x="24" y="30" fill="' + textColor + '" font-size="20" font-weight="700">' + escapeHTML(title) + '</text>');
      parts.push('<text x="24" y="52" fill="' + mutedColor + '" font-size="12">' + escapeHTML(subtitle) + '</text>');
      parts.push('<text x="' + sourceX + '" y="74" fill="' + mutedColor + '" font-size="12" font-weight="700">Source files</text>');
      parts.push('<text x="' + targetX + '" y="74" fill="' + mutedColor + '" font-size="12" font-weight="700">Target files</text>');

      for (const link of data.links) {
        const src = sourcePos.get(link.from);
        const dst = targetPos.get(link.to);
        if (!src || !dst) continue;

        const x1 = src.x + nodeWidth;
        const y1 = src.y + nodeHeight / 2;
        const x2 = dst.x;
        const y2 = dst.y + nodeHeight / 2;
        const dx = (x2 - x1) * 0.45;
        const width = Math.max(1.5, 1.5 + 5.5 * (link.count / maxLinkCount));
        const labelX = (x1 + x2) / 2;
        const labelY = (y1 + y2) / 2 - 4;

        parts.push(
          '<path d="M ' + x1 + ' ' + y1 +
          ' C ' + (x1 + dx) + ' ' + y1 + ', ' + (x2 - dx) + ' ' + y2 + ', ' + x2 + ' ' + y2 + '"' +
          ' fill="none" stroke="' + edgeColor + '" stroke-width="' + width.toFixed(2) + '" opacity="0.82"></path>'
        );

        parts.push('<text x="' + labelX + '" y="' + labelY + '" fill="' + edgeColor + '" font-size="12" text-anchor="middle">' + link.count + '</text>');
      }

      data.sources.forEach(function(node) {
        const pos = sourcePos.get(node.id);
        const label = truncateText(node.other ? (node.label + " (+" + node.count + ")") : node.label, 34);
        parts.push('<rect x="' + pos.x + '" y="' + pos.y + '" rx="10" ry="10" width="' + nodeWidth + '" height="' + nodeHeight + '" fill="' + nodeFill + '" stroke="' + nodeStroke + '"></rect>');
        parts.push('<text x="' + (pos.x + 10) + '" y="' + (pos.y + 22) + '" fill="' + textColor + '" font-size="12">' + escapeHTML(label) + '</text>');
        parts.push('<text x="' + (pos.x + nodeWidth - 10) + '" y="' + (pos.y + 22) + '" fill="' + mutedColor + '" font-size="12" text-anchor="end">' + node.count + '</text>');
      });

      data.targets.forEach(function(node) {
        const pos = targetPos.get(node.id);
        const label = truncateText(node.other ? (node.label + " (+" + node.count + ")") : node.label, 34);
        parts.push('<rect x="' + pos.x + '" y="' + pos.y + '" rx="10" ry="10" width="' + nodeWidth + '" height="' + nodeHeight + '" fill="' + nodeFill + '" stroke="' + nodeStroke + '"></rect>');
        parts.push('<text x="' + (pos.x + 10) + '" y="' + (pos.y + 22) + '" fill="' + textColor + '" font-size="12">' + escapeHTML(label) + '</text>');
        parts.push('<text x="' + (pos.x + nodeWidth - 10) + '" y="' + (pos.y + 22) + '" fill="' + mutedColor + '" font-size="12" text-anchor="end">' + node.count + '</text>');
      });

      parts.push('</svg>');
      return parts.join("");
    }

    function renderEdgeDiagram(from, to, deps, findings) {
      if (deps.length === 0) {
        return '<div class="empty">No resolved file dependencies for this edge.</div>';
      }

      return '<div class="svg-wrap">' +
        edgeDiagramSVG(from, to, deps, findings) +
        '</div>' +
        '<div class="svg-caption">Detailed file-level picture for the selected layer edge.</div>';
    }

    function selectEdge(from, to) {
      selectedEdge = { from: from, to: to };

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
        renderEdgeDiagram(from, to, deps, findings) +
        "<div class=\"columns\" style=\"margin-top: 16px;\">" +
          "<div><h3>Top source files</h3>" + renderCountList(topFrom) + "</div>" +
          "<div><h3>Top target files</h3>" + renderCountList(topTo) + "</div>" +
        "</div>" +
        "<div style=\"margin-top: 16px;\">" +
          "<h3>Related findings</h3>" +
          (findings.length === 0 ? "<div class=\"empty\">No findings attached to this edge.</div>" : findings.map(renderFinding).join("")) +
        "</div>" +
        "<div style=\"margin-top: 16px;\">" +
          "<h3>Dependencies</h3>" +
          renderDependencyCards(deps) +
        "</div>";

      renderOverviewGraph();
    }

    function renderDependencyCards(deps) {
      if (!deps || deps.length === 0) {
        return "<div class=\"empty\">No dependencies.</div>";
      }

      const limit = 80;
      const items = deps.slice(0, limit).map(function(dep) {
        const from = get(dep, "from_file", "fromFile") || "[unknown]";
        const to = dependencyTarget(dep) || "[unknown]";
        const kind = get(dep, "kind", "kind", "dependency");
        const usage = get(dep, "usage", "usage", "unknown");

        return "<div class=\"dependency-card\">" +
          "<div class=\"dep-kind\">" + escapeHTML(kind) + " / " + escapeHTML(usage) + "</div>" +
          "<div class=\"dep-path\">" + escapeHTML(from) + "</div>" +
          "<div class=\"dep-arrow\">→</div>" +
          "<div class=\"dep-path\">" + escapeHTML(to) + "</div>" +
        "</div>";
      }).join("");

      return "<div class=\"dependency-list\">" + items + "</div>" +
        (deps.length > limit ? "<div class=\"muted\" style=\"margin-top: 8px;\">... " + (deps.length - limit) + " more dependencies</div>" : "");
    }

    function renderEvidenceList(evidence) {
      if (!evidence || evidence.length === 0) {
        return "";
      }

      return "<details style=\"margin-top: 8px;\">" +
        "<summary class=\"muted\">Evidence: " + evidence.length + "</summary>" +
        "<pre>" + escapeHTML(evidence.slice(0, 5).map(function(item) {
          return [
            get(item, "message", "message"),
            [get(item, "from_file", "fromFile"), get(item, "to_file", "toFile")].filter(Boolean).join(" -> ")
          ].filter(Boolean).join("\n");
        }).join("\n\n")) + "</pre>" +
        "</details>";
    }

    function renderFinding(finding) {
      const id = get(finding, "id", "id");
      const severity = get(finding, "severity", "severity");
      const kind = get(finding, "kind", "kind");
      const title = get(finding, "title", "title");
      const risk = get(finding, "risk", "risk");
      const suggestion = get(finding, "suggestion", "suggestion");
      const evidence = get(finding, "evidence", "evidence", []);
      const dangerClass = severity === "high" || severity === "critical" ? "danger" : "warn";

      return "<div class=\"finding\">" +
        "<div><span class=\"badge " + dangerClass + "\">" + escapeHTML(severity) + "/" + escapeHTML(kind) + "</span></div>" +
        "<div style=\"margin-top: 6px; font-weight: 700;\">" + escapeHTML(id || title) + "</div>" +
        (title && id ? "<div class=\"muted\">" + escapeHTML(title) + "</div>" : "") +
        (risk ? "<p>" + escapeHTML(risk) + "</p>" : "") +
        (suggestion ? "<p class=\"muted\"><strong>Suggestion:</strong> " + escapeHTML(suggestion) + "</p>" : "") +
        renderEvidenceList(evidence) +
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
    renderOverviewGraph();
    renderEdges();
    renderFindings();

    for (const id of ["edgeSearch", "minEdgeCount", "onlyFindings", "onlyPolicy"]) {
      const el = document.getElementById(id);
      if (!el) continue;

      el.addEventListener("input", renderEdges);
      el.addEventListener("change", renderEdges);
    }
  </script>
</body>
</html>
`
