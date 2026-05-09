import { useMemo, useState } from 'react'
import type { ReviewGraph, ReviewGraphEdge, TreeNode, TreeReport } from '../types'

type TreeMode = 'single' | 'compare'

interface Props {
  tree: TreeReport
  graph: ReviewGraph
}

interface FlatTreeNode {
  node: TreeNode
  depth: number
  key: string
}

export function ProjectTree({ tree, graph }: Props) {
  const [mode, setMode] = useState<TreeMode>('single')
  const [selectedNodeKey, setSelectedNodeKey] = useState(nodeKey(tree.root))
  const [selectedEdgeKey, setSelectedEdgeKey] = useState<string | null>(null)
  const [collapsed, setCollapsed] = useState<Set<string>>(() => new Set())

  const selectedNode = useMemo(() => findNodeByKey(tree.root, selectedNodeKey), [tree.root, selectedNodeKey])
  const selectedEdge = useMemo(
    () => graph.edges.find((edge) => graphEdgeKey(edge) === selectedEdgeKey) ?? null,
    [graph.edges, selectedEdgeKey],
  )

  function toggleNode(key: string) {
    setCollapsed((current) => {
      const next = new Set(current)
      if (next.has(key)) {
        next.delete(key)
      } else {
        next.add(key)
      }
      return next
    })
  }

  return (
    <section className="card project-tree-card">
      <div className="card-title-row">
        <div>
          <p className="eyebrow">Architecture map</p>
          <h2>Dependency Tree</h2>
          <p className="muted">Layers/modules as 2.5D nodes, dependency movement as arrows, files as drill-down.</p>
        </div>

        <div className="tree-mode-toggle">
          <button type="button" className={mode === 'single' ? 'active' : ''} onClick={() => setMode('single')}>
            Single 2.5D
          </button>
          <button type="button" className={mode === 'compare' ? 'active' : ''} onClick={() => setMode('compare')}>
            Compare
          </button>
        </div>
      </div>

      <MapLegend />

      <div className="architecture-map-layout">
        <ArchitectureMap
          graph={graph}
          mode={mode}
          selectedEdgeKey={selectedEdgeKey}
          onSelectEdge={setSelectedEdgeKey}
        />

        <TreeDetailsPanel node={selectedNode} edge={selectedEdge} />
      </div>

      <details className="file-tree-drilldown">
        <summary>Files and directories drill-down</summary>

        <TreePane
          title="Files touched by review"
          root={tree.root}
          collapsed={collapsed}
          selectedNodeKey={selectedNodeKey}
          onToggleNode={toggleNode}
          onSelectNode={(key) => {
            setSelectedNodeKey(key)
            setSelectedEdgeKey(null)
          }}
        />
      </details>
    </section>
  )
}

function TreeLegend() {
  return (
    <div className="tree-impact-legend">
      <div>
        <strong>How to read this tree</strong>
        <span>Focus on colored nodes first. Neutral nodes are project context.</span>
      </div>

      <div className="tree-legend-items">
        <span className="tree-legend-item changed">Patch changed</span>
        <span className="tree-legend-item risky">Finding / risk</span>
        <span className="tree-legend-item neutral">Context</span>
      </div>
    </div>
  )
}

function MapLegend() {
  return (
    <div className="tree-impact-legend architecture-map-legend">
      <div>
        <strong>How to read this map</strong>
        <span>Nodes are layers/modules. Arrows are dependency movement between them.</span>
      </div>

      <div className="tree-legend-items">
        <span className="tree-legend-item changed">Changed edge</span>
        <span className="tree-legend-item risky">Finding / risk</span>
        <span className="tree-legend-item neutral">Unchanged context</span>
      </div>
    </div>
  )
}

function ArchitectureMap({
  graph,
  mode,
  selectedEdgeKey,
  onSelectEdge,
}: {
  graph: ReviewGraph
  mode: TreeMode
  selectedEdgeKey: string | null
  onSelectEdge: (key: string) => void
}) {
  const layout = buildArchitectureMapLayout(graph)
  const width = 1180
  const height = Math.max(520, layout.rows * 150 + 150)

  return (
    <section className={['architecture-map-card', mode === 'compare' ? 'compare' : ''].join(' ')}>
      <div className="tree-pane-header">
        <div>
          <h3>{mode === 'compare' ? 'Architecture compare map' : 'Architecture dependency map'}</h3>
          <p className="muted">Click an arrow to inspect dependency evidence. Changed/risky areas are visually raised.</p>
        </div>
        <span className="tag">{graph.nodes.length} nodes / {graph.edges.length} edges</span>
      </div>

      <div className="architecture-map-stage">
        <svg
          className="architecture-map-svg"
          viewBox={`0 0 ${width} ${height}`}
          preserveAspectRatio="xMidYMid meet"
          aria-hidden="true"
        >
          <defs>
            <marker
              id="architecture-arrow-head"
              markerWidth="12"
              markerHeight="12"
              refX="10"
              refY="6"
              orient="auto"
              markerUnits="strokeWidth"
            >
              <path d="M 0 0 L 12 6 L 0 12 z" />
            </marker>
          </defs>

          {layout.edges.map((edgeLayout, index) => {
            const key = graphEdgeKey(edgeLayout.edge)
            const selected = selectedEdgeKey === key
            const bend = edgeBend(index)

            return (
              <path
                key={`arch-path-${key}`}
                className={[
                  'architecture-edge',
                  edgeLayout.edge.movement,
                  selected ? 'selected' : '',
                  edgeLayout.edge.finding_ids?.length ? 'has-findings' : '',
                ].join(' ')}
                d={architectureEdgePath(edgeLayout.from, edgeLayout.to, bend)}
                markerEnd="url(#architecture-arrow-head)"
              />
            )
          })}
        </svg>

        {layout.nodes.map((nodeLayout) => (
          <button
            type="button"
            key={nodeLayout.node.id}
            className={[
              'architecture-node',
              nodeLayout.node.changed ? 'changed' : '',
              (nodeLayout.node.risk_points ?? 0) > 0 || (nodeLayout.node.finding_count ?? 0) > 0 ? 'risky' : '',
              mode === 'compare' ? 'compare' : '',
            ].join(' ')}
            style={{
              left: `${nodeLayout.x}px`,
              top: `${nodeLayout.y}px`,
            }}
            title={nodeLayout.node.id}
          >
            <span className="architecture-node-type">{nodeLayout.column}</span>
            <strong>{nodeLayout.node.label || nodeLayout.node.id}</strong>
            <span className="architecture-node-meta">
              <span>{nodeLayout.node.before_dependency_count ?? 0} → {nodeLayout.node.after_dependency_count ?? 0}</span>
              {(nodeLayout.node.finding_count ?? 0) > 0 && <span className="tag bad">findings {nodeLayout.node.finding_count}</span>}
              {(nodeLayout.node.risk_points ?? 0) > 0 && <span className="tag risk">risk {nodeLayout.node.risk_points}</span>}
            </span>
          </button>
        ))}

        {layout.edges.map((edgeLayout, index) => {
          const key = graphEdgeKey(edgeLayout.edge)
          const selected = selectedEdgeKey === key
          const label = architectureEdgeLabelPoint(edgeLayout.from, edgeLayout.to, edgeBend(index))

          return (
            <button
              type="button"
              key={`arch-label-${key}`}
              className={[
                'architecture-edge-label',
                edgeLayout.edge.movement,
                selected ? 'selected' : '',
                edgeLayout.edge.finding_ids?.length ? 'has-findings' : '',
              ].join(' ')}
              style={{
                left: `${label.x}px`,
                top: `${label.y}px`,
              }}
              onClick={() => onSelectEdge(key)}
            >
              <span>{edgeLayout.edge.movement}</span>
              <strong>{edgeLayout.edge.before_count ?? 0} → {edgeLayout.edge.after_count ?? 0}</strong>
              {(edgeLayout.edge.finding_ids?.length ?? 0) > 0 && <em>{edgeLayout.edge.finding_ids?.length} findings</em>}
            </button>
          )
        })}
      </div>
    </section>
  )
}

interface ArchitectureNodeLayout {
  node: ReviewGraph['nodes'][number]
  x: number
  y: number
  column: 'source' | 'target' | 'both'
}

interface ArchitectureEdgeLayout {
  edge: ReviewGraphEdge
  from: ArchitectureNodeLayout
  to: ArchitectureNodeLayout
}

function buildArchitectureMapLayout(graph: ReviewGraph) {
  const sources = new Set(graph.edges.map((edge) => edge.from))
  const targets = new Set(graph.edges.map((edge) => edge.to))

  const nodes = [...graph.nodes].sort((a, b) => architectureNodeWeight(b) - architectureNodeWeight(a) || a.id.localeCompare(b.id))

  const sourceNodes = nodes.filter((node) => sources.has(node.id) && !targets.has(node.id))
  const targetNodes = nodes.filter((node) => targets.has(node.id) && !sources.has(node.id))
  const bothNodes = nodes.filter((node) => sources.has(node.id) && targets.has(node.id))
  const isolatedNodes = nodes.filter((node) => !sources.has(node.id) && !targets.has(node.id))

  const layouts: ArchitectureNodeLayout[] = []
  const addColumn = (items: typeof nodes, x: number, column: ArchitectureNodeLayout['column']) => {
    items.forEach((node, index) => {
      layouts.push({
        node,
        x,
        y: 95 + index * 135,
        column,
      })
    })
  }

  addColumn(sourceNodes, 70, 'source')
  addColumn(bothNodes.concat(isolatedNodes), 430, 'both')
  addColumn(targetNodes, 790, 'target')

  const byID = new Map(layouts.map((item) => [item.node.id, item]))
  const edgeLayouts: ArchitectureEdgeLayout[] = graph.edges
    .map((edge) => {
      const from = byID.get(edge.from)
      const to = byID.get(edge.to)
      if (!from || !to) return null
      return { edge, from, to }
    })
    .filter((edge): edge is ArchitectureEdgeLayout => edge !== null)

  const maxRows = Math.max(sourceNodes.length, targetNodes.length, bothNodes.length + isolatedNodes.length, 1)

  return {
    nodes: layouts,
    edges: edgeLayouts,
    rows: maxRows,
  }
}

function architectureNodeWeight(node: ReviewGraph['nodes'][number]) {
  return (node.changed ? 1000 : 0) + (node.risk_points ?? 0) * 20 + (node.finding_count ?? 0) * 10 + (node.after_dependency_count ?? 0)
}

function architectureEdgePath(from: ArchitectureNodeLayout, to: ArchitectureNodeLayout, bend: number) {
  const startX = from.x + 270
  const startY = from.y + 54
  const endX = to.x
  const endY = to.y + 54
  const c1 = startX + Math.max(90, Math.abs(endX - startX) * 0.32)
  const c2 = endX - Math.max(90, Math.abs(endX - startX) * 0.32)

  return `M ${startX} ${startY} C ${c1} ${startY + bend}, ${c2} ${endY - bend}, ${endX} ${endY}`
}

function architectureEdgeLabelPoint(from: ArchitectureNodeLayout, to: ArchitectureNodeLayout, bend: number) {
  return {
    x: (from.x + to.x) / 2 + 170,
    y: (from.y + to.y) / 2 + 34 + bend * 0.16,
  }
}

function TreePane({
  title,
  root,
  collapsed,
  selectedNodeKey,
  onToggleNode,
  onSelectNode,
  muted,
}: {
  title: string
  root: TreeNode
  collapsed: Set<string>
  selectedNodeKey: string
  onToggleNode: (key: string) => void
  onSelectNode: (key: string) => void
  muted?: boolean
}) {
  const rows = flattenVisibleTree(root, collapsed)

  return (
    <div className={['tree-pane', muted ? 'tree-pane-muted' : ''].join(' ')}>
      <div className="tree-pane-header">
        <h3>{title}</h3>
        <span className="tag">{rows.length} nodes</span>
      </div>

      <div className="tree-rows">
        {rows.map(({ node, depth, key }) => (
          <TreeRow
            key={key}
            node={node}
            depth={depth}
            nodeKeyValue={key}
            selected={key === selectedNodeKey}
            collapsed={collapsed.has(key)}
            onToggleNode={onToggleNode}
            onSelectNode={onSelectNode}
          />
        ))}
      </div>
    </div>
  )
}

function TreeRow({
  node,
  depth,
  nodeKeyValue,
  selected,
  collapsed,
  onToggleNode,
  onSelectNode,
}: {
  node: TreeNode
  depth: number
  nodeKeyValue: string
  selected: boolean
  collapsed: boolean
  onToggleNode: (key: string) => void
  onSelectNode: (key: string) => void
}) {
  const children = node.children ?? []
  const hasChildren = children.length > 0
  const changed = hasNodeSignal(node)
  const riskPoints = node.risk_points ?? 0

  return (
    <div
      className={[
        'tree-row-25d',
        selected ? 'selected' : '',
        changed ? 'changed' : '',
        riskPoints > 0 ? 'risky' : '',
      ].join(' ')}
      style={{ '--tree-depth': String(depth) } as React.CSSProperties}
    >
      <button
        type="button"
        className="tree-disclosure"
        disabled={!hasChildren}
        onClick={() => onToggleNode(nodeKeyValue)}
        aria-label={collapsed ? 'Expand node' : 'Collapse node'}
      >
        {hasChildren ? (collapsed ? '▸' : '▾') : '•'}
      </button>

      <button type="button" className="tree-node-main" onClick={() => onSelectNode(nodeKeyValue)}>
        <span className={`node-kind ${node.kind}`}>{node.kind}</span>

        <span className="tree-node-text">
          <strong>{node.name || node.path || '.'}</strong>
          {node.path && node.path !== node.name ? <code>{node.path}</code> : null}
        </span>

        <span className="tree-node-badges">
          {node.change_kind && <span className="tag">{node.change_kind}</span>}
          {node.layer && <span className="tag">{node.layer}</span>}
          {node.role && <span className="tag">{node.role}</span>}
          {(node.changed_files_count ?? 0) > 0 && <span className="tag">changed {node.changed_files_count}</span>}
          {(node.finding_count ?? 0) > 0 && <span className="tag bad">findings {node.finding_count}</span>}
          {(node.runtime_finding_count ?? 0) > 0 && <span className="tag warn">runtime {node.runtime_finding_count}</span>}
          {riskPoints > 0 && <span className="tag risk">risk {riskPoints}</span>}
        </span>
      </button>
    </div>
  )
}

function DependencyRail({
  graph,
  selectedEdgeKey,
  onSelectEdge,
  compact,
}: {
  graph: ReviewGraph
  selectedEdgeKey: string | null
  onSelectEdge: (key: string) => void
  compact?: boolean
}) {
  const edges = graph.edges.slice(0, compact ? 24 : 48)
  const fromLayers = uniqueSorted(edges.map((edge) => edge.from))
  const toLayers = uniqueSorted(edges.map((edge) => edge.to))

  const rowHeight = compact ? 48 : 56
  const top = 42
  const width = 440
  const leftX = 96
  const rightX = 344
  const height = Math.max(fromLayers.length, toLayers.length, 1) * rowHeight + top + 38

  const fromY = layerYIndex(fromLayers, rowHeight, top)
  const toY = layerYIndex(toLayers, rowHeight, top)

  return (
    <div className={['dependency-rail', 'dependency-map', compact ? 'compact' : ''].join(' ')}>
      <div className="tree-pane-header">
        <h3>Dependency map</h3>
        <span className="tag">{graph.edges.length} edges</span>
      </div>

      {edges.length === 0 ? (
        <p className="muted">No dependency edges.</p>
      ) : (
        <div className="dependency-map-stage" style={{ minHeight: `${height}px` }}>
          <svg
            className="dependency-map-svg"
            viewBox={`0 0 ${width} ${height}`}
            preserveAspectRatio="none"
            aria-hidden="true"
          >
            <defs>
              <marker
                id="dependency-arrow-head"
                markerWidth="10"
                markerHeight="10"
                refX="8"
                refY="5"
                orient="auto"
                markerUnits="strokeWidth"
              >
                <path d="M 0 0 L 10 5 L 0 10 z" />
              </marker>
            </defs>

            {edges.map((edge, index) => {
              const key = graphEdgeKey(edge)
              const y1 = fromY(edge.from)
              const y2 = toY(edge.to)
              const bend = edgeBend(index)
              const selected = selectedEdgeKey === key

              return (
                <path
                  key={`path-${key}`}
                  className={[
                    'dependency-map-edge',
                    edge.movement,
                    selected ? 'selected' : '',
                    edge.finding_ids?.length ? 'has-findings' : '',
                  ].join(' ')}
                  d={dependencyPath(leftX, rightX, y1, y2, bend)}
                  markerEnd="url(#dependency-arrow-head)"
                />
              )
            })}
          </svg>

          {fromLayers.map((layer) => (
            <button
              type="button"
              key={`from-${layer}`}
              className="dependency-layer-chip dependency-layer-from"
              style={{ top: `${fromY(layer) - 18}px`, left: '8px' }}
              title={`from: ${layer}`}
            >
              <span>from</span>
              <strong>{layer}</strong>
            </button>
          ))}

          {toLayers.map((layer) => (
            <button
              type="button"
              key={`to-${layer}`}
              className="dependency-layer-chip dependency-layer-to"
              style={{ top: `${toY(layer) - 18}px`, right: '8px' }}
              title={`to: ${layer}`}
            >
              <span>to</span>
              <strong>{layer}</strong>
            </button>
          ))}

          {edges.map((edge, index) => {
            const key = graphEdgeKey(edge)
            const y = (fromY(edge.from) + toY(edge.to)) / 2 + edgeBend(index) * 0.18
            const selected = selectedEdgeKey === key

            return (
              <button
                type="button"
                key={`label-${key}`}
                className={[
                  'dependency-map-label',
                  edge.movement,
                  selected ? 'selected' : '',
                  edge.finding_ids?.length ? 'has-findings' : '',
                ].join(' ')}
                style={{ top: `${y - 17}px` }}
                onClick={() => onSelectEdge(key)}
                title={`${edge.from} -> ${edge.to}`}
              >
                <span className="dependency-map-label-main">
                  <strong>{edge.from}</strong>
                  <span>→</span>
                  <strong>{edge.to}</strong>
                </span>
                <span className="dependency-map-label-meta">
                  <span className="tag">{edge.movement}</span>
                  <span>{edge.before_count ?? 0} → {edge.after_count ?? 0}</span>
                  {(edge.finding_ids?.length ?? 0) > 0 && <span className="tag bad">{edge.finding_ids?.length} findings</span>}
                </span>
              </button>
            )
          })}
        </div>
      )}
    </div>
  )
}

function uniqueSorted(values: string[]) {
  return [...new Set(values.filter(Boolean))].sort()
}

function layerYIndex(layers: string[], rowHeight: number, top: number) {
  const index = new Map<string, number>()
  layers.forEach((layer, i) => index.set(layer, i))

  return (layer: string) => top + (index.get(layer) ?? 0) * rowHeight
}

function dependencyPath(leftX: number, rightX: number, y1: number, y2: number, bend: number) {
  const c1 = leftX + 92
  const c2 = rightX - 92
  const midLift = bend

  return `M ${leftX} ${y1} C ${c1} ${y1 + midLift}, ${c2} ${y2 - midLift}, ${rightX} ${y2}`
}

function edgeBend(index: number) {
  const pattern = [-18, -8, 0, 8, 18]
  return pattern[index % pattern.length]
}

function TreeDetailsPanel({ node, edge }: { node: TreeNode | null; edge: ReviewGraphEdge | null }) {
  return (
    <aside className="tree-details-panel">
      <div>
        <p className="eyebrow">Details</p>
        <h3>{edge ? 'Dependency edge' : 'Tree node'}</h3>
      </div>

      {edge ? (
        <div className="details-stack">
          <DetailRow label="From" value={edge.from} />
          <DetailRow label="To" value={edge.to} />
          <DetailRow label="Movement" value={edge.movement} />
          <DetailRow label="Before count" value={String(edge.before_count ?? 0)} />
          <DetailRow label="After count" value={String(edge.after_count ?? 0)} />
          <DetailRow label="Evidence count" value={String(edge.finding_ids?.length ?? 0)} />
          {edge.finding_ids?.length ? (
            <div className="detail-row">
              <span className="muted">Finding IDs</span>
              <div className="detail-chip-list">
                {edge.finding_ids.map((id) => (
                  <code key={id}>{id}</code>
                ))}
              </div>
            </div>
          ) : null}
        </div>
      ) : node ? (
        <div className="details-stack">
          <DetailRow label="Path" value={node.path || node.name} code />
          <DetailRow label="Kind" value={node.kind} />
          <DetailRow label="Layer" value={node.layer || '—'} />
          <DetailRow label="Role" value={node.role || '—'} />
          <DetailRow label="Change" value={node.change_kind || '—'} />
          <DetailRow label="Changed files" value={String(node.changed_files_count ?? 0)} />
          <DetailRow label="Findings" value={String(node.finding_count ?? 0)} />
          <DetailRow label="Runtime findings" value={String(node.runtime_finding_count ?? 0)} />
          <DetailRow label="Risk points" value={String(node.risk_points ?? 0)} />
          <DetailRow label="Children" value={String(node.children?.length ?? 0)} />
        </div>
      ) : (
        <p className="muted">Select a node or dependency edge.</p>
      )}
    </aside>
  )
}

function DetailRow({ label, value, code }: { label: string; value: string; code?: boolean }) {
  return (
    <div className="detail-row">
      <span className="muted">{label}</span>
      {code ? <code>{value}</code> : <strong>{value}</strong>}
    </div>
  )
}

function flattenVisibleTree(root: TreeNode, collapsed: Set<string>): FlatTreeNode[] {
  const rows: FlatTreeNode[] = []

  function walk(node: TreeNode, depth: number) {
    const key = nodeKey(node)
    rows.push({ node, depth, key })

    if (collapsed.has(key)) {
      return
    }

    for (const child of node.children ?? []) {
      walk(child, depth + 1)
    }
  }

  walk(root, 0)
  return rows
}

function findNodeByKey(root: TreeNode, key: string): TreeNode | null {
  if (nodeKey(root) === key) {
    return root
  }

  for (const child of root.children ?? []) {
    const found = findNodeByKey(child, key)
    if (found) {
      return found
    }
  }

  return null
}

function nodeKey(node: TreeNode) {
  return node.path || node.name || 'root'
}

function graphEdgeKey(edge: ReviewGraphEdge) {
  return `${edge.from}->${edge.to}:${edge.before_count ?? 0}:${edge.after_count ?? 0}:${edge.movement}`
}

function hasNodeSignal(node: TreeNode) {
  return (
    Boolean(node.change_kind) ||
    (node.changed_files_count ?? 0) > 0 ||
    (node.finding_count ?? 0) > 0 ||
    (node.runtime_finding_count ?? 0) > 0 ||
    (node.risk_points ?? 0) > 0
  )
}
