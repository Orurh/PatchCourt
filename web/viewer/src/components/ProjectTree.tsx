import { useMemo, useState } from 'react'
import type { ReviewGraph, ReviewGraphEdge, TreeNode, TreeReport } from '../types'
import { ArchitectureFlow } from './ArchitectureFlow'

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
  const [selectedNodeKey, setSelectedNodeKey] = useState(nodeKey(tree.root))
  const [selectedEdgeKey, setSelectedEdgeKey] = useState<string | null>(null)
  const [selectedGraphNodeID, setSelectedGraphNodeID] = useState<string | null>(null)
  const [collapsed, setCollapsed] = useState<Set<string>>(() => new Set())

  const selectedNode = useMemo(() => findNodeByKey(tree.root, selectedNodeKey), [tree.root, selectedNodeKey])
  const selectedEdge = useMemo(
    () => graph.edges.find((edge) => graphEdgeKey(edge) === selectedEdgeKey) ?? null,
    [graph.edges, selectedEdgeKey],
  )
  const selectedGraphNode = useMemo(
    () => graph.nodes.find((node) => node.id === selectedGraphNodeID) ?? null,
    [graph.nodes, selectedGraphNodeID],
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

        <span className="tag">interactive map</span>
      </div>

      <MapLegend />

      <div className="architecture-map-layout">
        <ArchitectureFlow
          graph={graph}
          selectedEdgeKey={selectedEdgeKey}
          selectedNodeID={selectedGraphNodeID}
          onSelectEdge={setSelectedEdgeKey}
          onSelectNode={setSelectedGraphNodeID}
        />

        <TreeDetailsPanel
          node={selectedNode}
          graphNode={selectedGraphNode}
          graph={graph}
          edge={selectedEdge}
        />
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
            setSelectedGraphNodeID(null)
          }}
        />
      </details>
    </section>
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
        <span className="tree-legend-item edge-added">Added dependency</span>
        <span className="tree-legend-item edge-changed">Changed dependency</span>
        <span className="tree-legend-item edge-removed">Removed dependency</span>
        <span className="tree-legend-item node-changed">Touched module</span>
        <span className="tree-legend-item node-risk">Risk / finding</span>
      </div>
    </div>
  )
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

function TreeDetailsPanel({
  node,
  graphNode,
  graph,
  edge,
}: {
  node: TreeNode | null
  graphNode: ReviewGraph['nodes'][number] | null
  graph: ReviewGraph
  edge: ReviewGraphEdge | null
}) {
  const incoming = graphNode ? graph.edges.filter((item) => item.to === graphNode.id) : []
  const outgoing = graphNode ? graph.edges.filter((item) => item.from === graphNode.id) : []

  return (
    <aside className="tree-details-panel">
      <div>
        <p className="eyebrow">Details</p>
        <h3>{edge ? 'Dependency edge' : graphNode ? 'Architecture module' : 'Tree node'}</h3>
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
      ) : graphNode ? (
        <div className="details-stack">
          <DetailRow label="Module" value={graphNode.id} code />
          <DetailRow label="Changed" value={graphNode.changed ? 'yes' : 'no'} />
          <DetailRow label="Before deps" value={String(graphNode.before_dependency_count ?? 0)} />
          <DetailRow label="After deps" value={String(graphNode.after_dependency_count ?? 0)} />
          <DetailRow label="Findings" value={String(graphNode.finding_count ?? 0)} />
          <DetailRow label="Risk points" value={String(graphNode.risk_points ?? 0)} />
          <DetailRow label="Incoming edges" value={String(incoming.length)} />
          <DetailRow label="Outgoing edges" value={String(outgoing.length)} />

          {outgoing.length ? (
            <div className="detail-row">
              <span className="muted">Outgoing</span>
              <div className="detail-chip-list">
                {outgoing.slice(0, 8).map((item) => (
                  <code key={`out-${graphEdgeKey(item)}`}>{item.to} · {item.movement}</code>
                ))}
              </div>
            </div>
          ) : null}

          {incoming.length ? (
            <div className="detail-row">
              <span className="muted">Incoming</span>
              <div className="detail-chip-list">
                {incoming.slice(0, 8).map((item) => (
                  <code key={`in-${graphEdgeKey(item)}`}>{item.from} · {item.movement}</code>
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
