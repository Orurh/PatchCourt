package check

const checkHTMLDocumentStart = `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <title>PatchCourt Report</title>
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <style>
`

const checkHTMLAfterCSSBeforeData = `  </style>
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

  <script type="application/json" id="patchcourt-data">`

const checkHTMLAfterDataBeforeJS = `</script>

  <script>
`

const checkHTMLDocumentEnd = `
  </script>
</body>
</html>
`
