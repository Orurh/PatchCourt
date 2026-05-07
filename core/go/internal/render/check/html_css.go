package check

const checkHTMLCSS = `
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
      --svg-bg: #0d1528;
    }

    * { box-sizing: border-box; }

    html,
    body {
      margin: 0;
      height: 100%;
      background: var(--bg);
      color: var(--text);
      font: 14px/1.5 system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
      overflow: hidden;
    }

    header {
      height: 64px;
      padding: 12px 20px;
      border-bottom: 1px solid var(--line);
      background: linear-gradient(135deg, #111a32, #0b1020);
    }

    h1, h2, h3 { margin: 0; }

    h1 {
      font-size: 23px;
      letter-spacing: -0.03em;
      line-height: 1.1;
    }

    h2 {
      font-size: 16px;
      margin-bottom: 10px;
    }

    h3 {
      font-size: 12px;
      margin-bottom: 8px;
      color: var(--muted);
      text-transform: uppercase;
      letter-spacing: 0.08em;
    }

    .subtitle {
      margin-top: 4px;
      color: var(--muted);
      font-size: 12px;
      overflow: hidden;
      text-overflow: ellipsis;
      white-space: nowrap;
    }

    .shell {
      height: calc(100vh - 64px);
      max-width: 100vw;
      padding: 12px;
      display: grid;
      grid-template-rows: 96px minmax(0, 1fr);
      gap: 12px;
      overflow: hidden;
    }

    .topbar {
      min-height: 0;
      display: grid;
      grid-template-columns: minmax(680px, 1fr) 430px;
      gap: 12px;
      overflow: hidden;
    }

    .workspace {
      min-height: 0;
      display: grid;
      grid-template-columns: 320px minmax(650px, 1fr) 420px;
      grid-template-areas: "edges details graph";
      gap: 12px;
      overflow: hidden;
    }

    .panel {
      min-width: 0;
      min-height: 0;
      background: var(--panel);
      border: 1px solid var(--line);
      border-radius: 16px;
      padding: 14px;
      overflow: hidden;
      box-shadow: 0 16px 44px rgba(0, 0, 0, 0.18);
    }

    .summary-panel,
    .filters-panel {
      padding: 12px 14px;
    }

    .summary-panel h2,
    .filters-panel h2 {
      margin-bottom: 8px;
      font-size: 14px;
    }

    .summary {
      display: grid;
      grid-template-columns: repeat(8, minmax(82px, 1fr));
      gap: 8px;
    }

    .metric {
      min-width: 0;
      background: var(--panel2);
      border: 1px solid rgba(138, 180, 255, 0.13);
      border-radius: 12px;
      padding: 8px 10px;
    }

    .metric .value {
      font-size: 20px;
      line-height: 1.1;
      font-weight: 800;
    }

    .metric .label {
      margin-top: 3px;
      color: var(--muted);
      font-size: 11px;
      overflow: hidden;
      text-overflow: ellipsis;
      white-space: nowrap;
    }

    .filters {
      display: grid;
      gap: 8px;
      margin: 0;
    }

    .filter-row {
      display: grid;
      grid-template-columns: 1fr 94px;
      gap: 8px;
    }

    input[type="search"],
    input[type="number"] {
      width: 100%;
      height: 36px;
      color: var(--text);
      background: var(--code);
      border: 1px solid var(--line);
      border-radius: 11px;
      padding: 8px 10px;
      outline: none;
    }

    input[type="search"]:focus,
    input[type="number"]:focus {
      border-color: var(--accent);
      box-shadow: 0 0 0 3px rgba(138, 180, 255, 0.12);
    }

    .check-row {
      display: flex;
      flex-wrap: wrap;
      gap: 14px;
      color: var(--muted);
      font-size: 13px;
    }

    .edges-pane {
      grid-area: edges;
      display: flex;
      flex-direction: column;
    }

    .graph-pane {
      grid-area: graph;
      display: flex;
      flex-direction: column;
    }

    .side-stack {
      grid-area: details;
      min-width: 0;
      min-height: 0;
      display: grid;
      grid-template-rows: minmax(0, 1fr) minmax(220px, 0.36fr);
      gap: 12px;
      overflow: hidden;
    }

    .details-pane,
    .findings-pane {
      display: flex;
      flex-direction: column;
      min-width: 0;
      min-height: 0;
    }

    .edge-list,
    #edgeDetails,
    #findings {
      min-height: 0;
      overflow: auto;
      scrollbar-width: thin;
      scrollbar-color: #3a4a72 transparent;
    }

    .edge-list {
      display: grid;
      align-content: start;
      gap: 8px;
      padding-right: 4px;
    }

    .edge-row {
      width: 100%;
      text-align: left;
      cursor: pointer;
      color: var(--text);
      background: rgba(255, 255, 255, 0.035);
      border: 1px solid var(--line);
      border-radius: 12px;
      padding: 10px;
    }

    .edge-row:hover {
      border-color: rgba(138, 180, 255, 0.65);
      background: rgba(138, 180, 255, 0.08);
    }

    .edge-row.selected {
      border-color: var(--accent);
      background: rgba(138, 180, 255, 0.12);
      box-shadow: inset 3px 0 0 var(--accent);
    }

    .edge-title {
      display: flex;
      justify-content: space-between;
      gap: 12px;
      font-weight: 800;
    }

    .edge-meta {
      margin-top: 6px;
      color: var(--muted);
      font-size: 12px;
    }

    .panel-heading {
      display: flex;
      justify-content: space-between;
      gap: 12px;
      align-items: flex-start;
      margin-bottom: 10px;
      flex: 0 0 auto;
    }

    .legend {
      display: flex;
      gap: 8px;
      flex-wrap: wrap;
      color: var(--muted);
      font-size: 11px;
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
      flex: 1 1 auto;
      min-height: 0;
      border: 1px solid var(--line);
      border-radius: 14px;
      overflow: auto;
      background: var(--svg-bg);
    }

    #overviewGraph {
      display: block;
      width: 100%;
      height: 100%;
      min-width: 420px;
      min-height: 330px;
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

    #edgeDetails {
      padding-right: 6px;
    }

    .badge {
      display: inline-flex;
      align-items: center;
      gap: 4px;
      border-radius: 999px;
      padding: 2px 8px;
      font-size: 12px;
      background: rgba(255, 255, 255, 0.035);
      border: 1px solid var(--line);
      color: var(--muted);
    }

    .badge.danger {
      color: #fecaca;
      border-color: rgba(255, 107, 107, 0.55);
      background: rgba(255, 107, 107, 0.08);
    }

    .badge.warn {
      color: #fed7aa;
      border-color: rgba(255, 184, 107, 0.55);
      background: rgba(255, 184, 107, 0.08);
    }

    .finding {
      border: 1px solid rgba(255,255,255,0.08);
      background: rgba(255, 255, 255, 0.025);
      border-radius: 12px;
      padding: 12px;
      margin-top: 10px;
    }

    .finding:first-child {
      margin-top: 0;
    }

    .dependency-list {
      display: grid;
      gap: 8px;
    }

    .dependency-card {
      border: 1px solid var(--line);
      background: rgba(255, 255, 255, 0.025);
      border-radius: 12px;
      padding: 10px;
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
      line-height: 1.4;
      white-space: normal;
      overflow-wrap: anywhere;
      word-break: break-word;
    }

    .dependency-card .dep-arrow {
      color: var(--accent);
      margin: 3px 0;
    }

    .small-help {
      margin-bottom: 10px;
      color: var(--muted);
      font-size: 12px;
    }

    .muted {
      color: var(--muted);
    }

    .empty {
      color: var(--muted);
      border: 1px dashed var(--line);
      border-radius: 12px;
      padding: 14px;
      background: rgba(255, 255, 255, 0.02);
    }

    code,
    pre {
      font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
    }

    code {
      background: var(--code);
      border: 1px solid var(--line);
      border-radius: 6px;
      padding: 1px 5px;
      overflow-wrap: anywhere;
      word-break: break-word;
    }

    pre {
      background: var(--code);
      border: 1px solid var(--line);
      border-radius: 12px;
      padding: 12px;
      overflow: auto;
      max-height: 420px;
      white-space: pre-wrap;
      overflow-wrap: anywhere;
      word-break: break-word;
    }

    .columns {
      display: grid;
      grid-template-columns: 1fr 1fr;
      gap: 12px;
    }

    .small-list {
      display: grid;
      gap: 4px;
      margin: 0;
      padding: 0;
      list-style: none;
    }

    .small-list li {
      min-width: 0;
      display: grid;
      grid-template-columns: minmax(0, 1fr) auto;
      gap: 10px;
      border-bottom: 1px solid rgba(255,255,255,0.06);
      padding: 5px 0;
    }

    .small-list code {
      display: inline-block;
      max-width: 100%;
    }

    .svg-wrap {
      margin-top: 12px;
      border: 1px solid var(--line);
      border-radius: 14px;
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
      min-width: 720px;
      min-height: 260px;
      background: var(--svg-bg);
    }

    .evidence-list {
      display: grid;
      gap: 8px;
      margin-top: 8px;
    }

    .evidence-item,
    .evidence-snippet {
      border: 1px solid rgba(138, 180, 255, 0.18);
      border-radius: 12px;
      background: rgba(15, 23, 42, 0.72);
      padding: 10px;
    }

    .evidence-location {
      color: var(--accent);
      font-weight: 700;
      margin-bottom: 6px;
    }

    .evidence-message {
      color: var(--text);
      margin-bottom: 8px;
    }

    .evidence-snippet {
      margin-top: 8px;
      font-size: 12px;
      line-height: 1.45;
      color: #dbeafe;
    }

    @media (max-width: 1400px) {
      .workspace {
        grid-template-columns: 300px minmax(560px, 1fr) 360px;
      }

      .topbar {
        grid-template-columns: minmax(640px, 1fr) 380px;
      }

      .summary {
        grid-template-columns: repeat(4, minmax(90px, 1fr));
      }
    }

    @media (max-width: 1100px) {
      html,
      body {
        height: auto;
        min-height: 100%;
        overflow: auto;
      }

      .shell {
        height: auto;
        overflow: visible;
      }

      .topbar {
        grid-template-columns: 1fr;
      }

      .workspace {
        grid-template-columns: 1fr;
        grid-template-areas:
          "graph"
          "edges"
          "details";
        overflow: visible;
      }

      .side-stack {
        grid-template-rows: auto auto;
        overflow: visible;
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
        max-height: 520px;
      }

      .overview-wrap {
        height: 420px;
      }

      .columns {
        grid-template-columns: 1fr;
      }
    }

/* PatchCourt check report layout override:
   left = edge list, center = readable details/findings, right = compact graph. */
.workspace {
  grid-template-columns: 320px minmax(680px, 1fr) 430px;
  grid-template-areas: "edges details graph";
}

.edges-pane {
  grid-area: edges;
}

.graph-pane {
  grid-area: graph;
}

.side-stack {
  grid-area: details;
  grid-template-rows: minmax(0, 1fr) minmax(220px, 0.38fr);
}

.overview-wrap {
  min-height: 0;
}

#overviewGraph {
  min-width: 420px;
  min-height: 330px;
}

#edgeDetails {
  font-size: 14px;
  line-height: 1.55;
}

.finding {
  border: 1px solid rgba(255,255,255,0.08);
  background: rgba(255,255,255,0.025);
  border-radius: 12px;
  padding: 12px;
  margin-top: 10px;
}

.dependency-card {
  border-radius: 12px;
  padding: 10px;
}

@media (max-width: 1400px) {
  .workspace {
    grid-template-columns: 300px minmax(580px, 1fr) 360px;
  }

  .summary {
    grid-template-columns: repeat(4, minmax(90px, 1fr));
  }
}

@media (max-width: 1100px) {
  .workspace {
    grid-template-columns: 1fr;
    grid-template-areas:
      "graph"
      "edges"
      "details";
  }
}


/* PatchCourt check report readability override.
   Make report page scrollable instead of fixed dashboard. */
html,
body {
  height: auto;
  min-height: 100%;
  overflow: auto;
}

body {
  overflow-x: hidden;
}

.shell {
  height: auto;
  min-height: calc(100vh - 96px);
  overflow: visible;
  grid-template-rows: auto auto;
  padding: 14px;
}

.topbar {
  min-height: auto;
  overflow: visible;
}

.workspace {
  min-height: auto;
  align-items: start;
  grid-template-columns: 310px minmax(700px, 1fr) 390px;
  grid-template-areas:
    "edges details graph"
    "edges findings graph";
}

.edges-pane {
  grid-area: edges;
  position: sticky;
  top: 12px;
  max-height: calc(100vh - 24px);
}

.graph-pane {
  grid-area: graph;
  position: sticky;
  top: 12px;
  max-height: calc(100vh - 24px);
}

.side-stack {
  grid-area: details;
  display: contents;
}

.details-pane {
  grid-area: details;
}

.findings-pane {
  grid-area: findings;
}

.edges-pane,
.graph-pane,
.details-pane,
.findings-pane {
  min-height: auto;
  overflow: visible;
}

.edge-list {
  max-height: calc(100vh - 120px);
  overflow: auto;
}

#edgeDetails,
#findings {
  overflow: visible;
  max-height: none;
  min-height: auto;
}

#findings {
  display: grid;
  gap: 12px;
  font-size: 14px;
  line-height: 1.55;
}

.finding {
  border: 1px solid rgba(255,255,255,0.08);
  background: rgba(255,255,255,0.025);
  border-radius: 14px;
  padding: 14px;
  margin-top: 0;
}

.finding:first-child {
  border-top: 1px solid rgba(255,255,255,0.08);
  padding-top: 14px;
}

.finding p {
  margin: 8px 0 0;
}

.finding pre {
  max-height: none;
  font-size: 12px;
  line-height: 1.45;
}

.overview-wrap {
  height: 360px;
  min-height: 320px;
  flex: 0 0 auto;
}

#overviewGraph {
  min-width: 420px;
  min-height: 330px;
}

.summary {
  grid-template-columns: repeat(8, minmax(88px, 1fr));
}

@media (max-width: 1450px) {
  .workspace {
    grid-template-columns: 290px minmax(620px, 1fr) 340px;
  }

  .summary {
    grid-template-columns: repeat(4, minmax(90px, 1fr));
  }
}

@media (max-width: 1150px) {
  html,
  body {
    overflow: auto;
  }

  .workspace {
    grid-template-columns: 1fr;
    grid-template-areas:
      "graph"
      "edges"
      "details"
      "findings";
  }

  .edges-pane,
  .graph-pane {
    position: static;
    max-height: none;
  }

  .edge-list {
    max-height: 420px;
  }

  .overview-wrap {
    height: 480px;
  }
}

`
