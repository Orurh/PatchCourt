package check

const checkHTMLJS = `    const data = JSON.parse(document.getElementById("patchcourt-data").textContent);

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


    function buildFileLayerMap() {
      const result = new Map();
      const files = data.files || [];

      for (const file of files) {
        const path = get(file, "path", "path", "");
        const layer = get(file, "layer", "layer", "");
        if (!path || !layer) continue;
        result.set(path, layer);
      }

      return result;
    }

    function evidenceFile(evidence) {
      return get(evidence, "file", "file", "") || get(evidence, "from_file", "fromFile", "");
    }

    function findingSeverityRank(finding) {
      const severity = get(finding, "severity", "severity", "");
      if (severity === "critical") return 4;
      if (severity === "high") return 3;
      if (severity === "medium") return 2;
      if (severity === "low") return 1;
      return 0;
    }

    function buildNodeFindingInfo() {
      const fileLayer = buildFileLayerMap();
      const result = new Map();

      for (const finding of data.findings || []) {
        const evidence = get(finding, "evidence", "evidence", []);
        const rank = findingSeverityRank(finding);

        for (const item of evidence) {
          const file = evidenceFile(item);
          const layer = fileLayer.get(file);
          if (!layer) continue;

          let info = result.get(layer);
          if (!info) {
            info = {
              findingIDs: new Set(),
              evidenceCount: 0,
              maxSeverityRank: 0,
              maxSeverity: "",
              runtimeRiskCount: 0
            };
            result.set(layer, info);
          }

          const id = get(finding, "id", "id", "");
          const kind = get(finding, "kind", "kind", "");

          if (id) info.findingIDs.add(id);
          info.evidenceCount += 1;

          if (kind === "runtime_risk") {
            info.runtimeRiskCount += 1;
          }

          if (rank > info.maxSeverityRank) {
            info.maxSeverityRank = rank;
            info.maxSeverity = get(finding, "severity", "severity", "");
          }
        }
      }

      return result;
    }

    const nodeFindingInfo = buildNodeFindingInfo();

    function nodeRiskClass(info) {
      if (!info || info.evidenceCount === 0) return "";
      if (info.maxSeverityRank >= 3) return "danger";
      if (info.maxSeverityRank >= 2) return "warn";
      return "info";
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

        const info = nodeFindingInfo.get(node);
        const riskClass = nodeRiskClass(info);

        let fill = connected ? "#1f2a44" : "#172033";
        let stroke = connected ? "#3a4a72" : "#6b7280";
        let strokeWidth = 1.2;
        const dash = connected ? "" : ' stroke-dasharray="7 5"';
        const textColor = connected ? "#e8eefc" : "#9aa7c7";
        const label = truncateText(node, 17);

        if (riskClass === "danger") {
          fill = "#3a1f2c";
          stroke = "#ef4444";
          strokeWidth = 2.4;
        } else if (riskClass === "warn") {
          fill = "#392a18";
          stroke = "#fb923c";
          strokeWidth = 2.0;
        } else if (riskClass === "info") {
          stroke = "#60a5fa";
          strokeWidth = 1.8;
        }

        const badge = info && info.evidenceCount > 0
          ? '<g>' +
              '<circle cx="' + (p.x + nodeWidth / 2 - 6) + '" cy="' + (p.y - nodeHeight / 2 + 6) + '" r="11" fill="' + stroke + '"></circle>' +
              '<text x="' + (p.x + nodeWidth / 2 - 6) + '" y="' + (p.y - nodeHeight / 2 + 10) + '" fill="#fff" font-size="11" font-weight="700" text-anchor="middle">' + escapeHTML(String(info.findingIDs.size)) + '</text>' +
            '</g>'
          : "";

        const title = info && info.evidenceCount > 0
          ? '<title>' + escapeHTML(node + ": " + info.findingIDs.size + " finding(s), " + info.evidenceCount + " evidence item(s)") + '</title>'
          : "";

        parts.push(
          '<g class="overview-node overview-node-' + escapeHTML(riskClass || "normal") + '">' +
            title +
            '<ellipse cx="' + p.x + '" cy="' + p.y + '" rx="' + (nodeWidth / 2) + '" ry="' + (nodeHeight / 2) + '" fill="' + fill + '" stroke="' + stroke + '" stroke-width="' + strokeWidth + '"' + dash + '></ellipse>' +
            '<text x="' + p.x + '" y="' + (p.y + 4) + '" fill="' + textColor + '" font-size="13" text-anchor="middle">' + escapeHTML(label) + '</text>' +
            badge +
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

    function evidenceLocation(item) {
      const file = get(item, "file", "file", "") || get(item, "from_file", "fromFile", "");
      if (!file) return "";

      const lineStart = Number(get(item, "line_start", "lineStart", 0));
      const lineEnd = Number(get(item, "line_end", "lineEnd", 0));

      if (lineStart > 0 && lineEnd > lineStart) {
        return file + ":" + lineStart + "-" + lineEnd;
      }

      if (lineStart > 0) {
        return file + ":" + lineStart;
      }

      return file;
    }

    function renderEvidenceList(evidence) {
      if (!evidence || evidence.length === 0) {
        return "";
      }

      const limit = 5;
      const items = evidence.slice(0, limit).map(function(item) {
        const location = evidenceLocation(item);
        const message = get(item, "message", "message");
        const snippet = get(item, "snippet", "snippet");
        const files = [get(item, "from_file", "fromFile"), get(item, "to_file", "toFile")].filter(Boolean).join(" -> ");
        const layers = [get(item, "from_layer", "fromLayer"), get(item, "to_layer", "toLayer")].filter(Boolean).join(" -> ");

        return "<div class=\"evidence-item\">" +
          (location ? "<div><code>" + escapeHTML(location) + "</code></div>" : "") +
          (message ? "<div class=\"evidence-message\">" + escapeHTML(message) + "</div>" : "") +
          (snippet ? "<pre class=\"evidence-snippet\">" + escapeHTML(snippet) + "</pre>" : "") +
          (files ? "<div class=\"muted\"><code>" + escapeHTML(files) + "</code></div>" : "") +
          (layers ? "<div class=\"muted\">" + escapeHTML(layers) + "</div>" : "") +
          "</div>";
      }).join("");

      return "<details class=\"evidence-details\" style=\"margin-top: 8px;\">" +
        "<summary class=\"muted\">Evidence: " + evidence.length + "</summary>" +
        "<div class=\"evidence-list\">" + items + "</div>" +
        (evidence.length > limit ? "<div class=\"muted\" style=\"margin-top: 8px;\">... " + (evidence.length - limit) + " more evidence item(s)</div>" : "") +
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
    }`
