import { useCallback, useEffect, useMemo, useState } from 'react'
import ELK from 'elkjs/lib/elk.bundled.js'
import {
  Background,
  BaseEdge,
  Controls,
  EdgeLabelRenderer,
  Handle,
  MarkerType,
  MiniMap,
  Position,
  ReactFlow,
  type Edge,
  type EdgeProps,
  type Node,
  type NodeProps,
} from '@xyflow/react'
import '@xyflow/react/dist/style.css'

import type { ReviewGraph, ReviewGraphEdge, ReviewGraphNode } from '../types'

type DependencyScope = 'changed' | 'all'
type GraphDetail = 'grouped' | 'packages'
type LayoutDirection = 'DOWN' | 'RIGHT'

interface Props {
  graph: ReviewGraph
  selectedEdgeKey: string | null
  selectedNodeID: string | null
  onSelectEdge: (key: string | null) => void
  onSelectNode: (id: string | null) => void
}

interface ArchitectureNodeData extends Record<string, unknown> {
  graphNode: ReviewGraphNode
  direction: LayoutDirection
}

interface ArchitectureEdgeData extends Record<string, unknown> {
  edge: ReviewGraphEdge
  lane: number
  direction: LayoutDirection
}

const elk = new ELK()

const nodeTypes = {
  architectureNode: ArchitectureNode,
}

const edgeTypes = {
  architectureEdge: ArchitectureFlowEdge,
}

export function ArchitectureFlow({ graph, selectedEdgeKey, selectedNodeID, onSelectEdge, onSelectNode }: Props) {
  const [scope, setScope] = useState<DependencyScope>('changed')
  const [detail, setDetail] = useState<GraphDetail>('grouped')
  const [direction, setDirection] = useState<LayoutDirection>('DOWN')
  const displayGraph = useMemo(() => (detail === 'grouped' ? groupGraphByParentModules(graph) : graph), [graph, detail])
  const visibleGraph = useMemo(() => filterGraph(displayGraph, scope), [displayGraph, scope])
  const denseLayout = scope === 'all' || detail === 'packages'
  const [nodes, setNodes] = useState<Node<ArchitectureNodeData>[]>([])
  const [edges, setEdges] = useState<Edge[]>([])

  useEffect(() => {
    let cancelled = false

    layoutGraph(visibleGraph, direction, denseLayout)
      .then((layouted) => {
        if (cancelled) return
        setNodes(layouted.nodes)
        setEdges(layouted.edges)
      })
      .catch((error: unknown) => {
        console.error('layout architecture graph:', error)
        setNodes([])
        setEdges([])
      })

    return () => {
      cancelled = true
    }
  }, [visibleGraph, direction, denseLayout])

  const handleEdgeClick = useCallback(
    (_event: React.MouseEvent, edge: Edge) => {
      onSelectNode(null)
      onSelectEdge(edge.id)
    },
    [onSelectEdge, onSelectNode],
  )

  const handleNodeClick = useCallback(
    (_event: React.MouseEvent, node: Node<ArchitectureNodeData>) => {
      onSelectEdge(null)
      onSelectNode(node.id)
    },
    [onSelectEdge, onSelectNode],
  )

  return (
    <section className={['architecture-flow-card', denseLayout ? 'dense' : ''].join(' ')}>
      <div className="tree-pane-header">
        <div>
          <h3>Architecture dependency map</h3>
          <p className="muted">Auto-layout dependency graph. Click an edge to inspect evidence.</p>
        </div>

        <div className="architecture-map-actions">
          {graph.source && <span className="tag">{graph.source}</span>}
          <span className="tag">{visibleGraph.nodes.length}/{displayGraph.nodes.length} nodes</span>
          <span className="tag">{visibleGraph.edges.length}/{displayGraph.edges.length} edges</span>

          <div className="tree-mode-toggle small">
            <button type="button" className={detail === 'grouped' ? 'active' : ''} onClick={() => setDetail('grouped')}>
              Grouped
            </button>
            <button type="button" className={detail === 'packages' ? 'active' : ''} onClick={() => setDetail('packages')}>
              Packages
            </button>
          </div>

          <div className="tree-mode-toggle small">
            <button type="button" className={direction === 'DOWN' ? 'active' : ''} onClick={() => setDirection('DOWN')}>
              Top-down
            </button>
            <button type="button" className={direction === 'RIGHT' ? 'active' : ''} onClick={() => setDirection('RIGHT')}>
              Left-right
            </button>
          </div>

          <div className="tree-mode-toggle small">
            <button type="button" className={scope === 'changed' ? 'active' : ''} onClick={() => setScope('changed')}>
              Changed only
            </button>
            <button type="button" className={scope === 'all' ? 'active' : ''} onClick={() => setScope('all')}>
              All deps
            </button>
          </div>
        </div>
      </div>

      <div className="architecture-flow-stage">
        {visibleGraph.edges.length === 0 ? (
          <div className="architecture-map-empty">
            <strong>No dependency edges to show.</strong>
            <span>Try switching to All deps or generate a review with dependency changes.</span>
          </div>
        ) : (
          <ReactFlow
            nodes={nodes.map((node) => ({
              ...node,
              selected: node.id === selectedNodeID,
            }))}
            edges={edges.map((edge) => ({
              ...edge,
              selected: edge.id === selectedEdgeKey,
            }))}
            nodeTypes={nodeTypes}
            edgeTypes={edgeTypes}
            fitView
            fitViewOptions={{ padding: denseLayout ? 0.08 : 0.18 }}
            minZoom={denseLayout ? 0.18 : 0.25}
            maxZoom={1.6}
            nodesDraggable={false}
            nodesConnectable={false}
            elementsSelectable
            onEdgeClick={handleEdgeClick}
            onNodeClick={handleNodeClick}
            onPaneClick={() => {
              onSelectEdge(null)
              onSelectNode(null)
            }}
          >
            <Background />
            <MiniMap pannable zoomable />
            <Controls />
          </ReactFlow>
        )}
      </div>
    </section>
  )
}

function ArchitectureFlowEdge({
  id,
  sourceX,
  sourceY,
  targetX,
  targetY,
  markerEnd,
  selected,
  data,
  style,
}: EdgeProps<Edge<ArchitectureEdgeData>>) {
  const edge = data?.edge
  const lane = data?.lane ?? 0
  const direction = data?.direction ?? 'DOWN'

  const route = direction === 'DOWN'
    ? verticalEdgeRoute(sourceX, sourceY, targetX, targetY, lane)
    : horizontalEdgeRoute(sourceX, sourceY, targetX, targetY, lane)

  return (
    <>
      <BaseEdge id={id} path={route.path} markerEnd={markerEnd} style={style} />

      {selected && edge ? (
        <EdgeLabelRenderer>
          <button
            type="button"
            className="architecture-flow-edge-label nodrag nopan"
            style={{
              transform: `translate(-50%, -50%) translate(${route.labelX}px, ${route.labelY}px)`,
            }}
          >
            <span>{edge.movement}</span>
            <strong>{edge.before_count ?? 0} → {edge.after_count ?? 0}</strong>
          </button>
        </EdgeLabelRenderer>
      ) : null}
    </>
  )
}

function horizontalEdgeRoute(sourceX: number, sourceY: number, targetX: number, targetY: number, lane: number) {
  const radius = 18
  const exitX = sourceX + 48
  const enterX = targetX - 48
  const midX = sourceX + (targetX - sourceX) * 0.5 + lane * 0.26
  const labelY = sourceY + (targetY - sourceY) * 0.5 + lane

  return {
    labelX: midX,
    labelY,
    path: roundedOrthogonalPath([
      { x: sourceX, y: sourceY },
      { x: exitX, y: sourceY },
      { x: midX, y: sourceY },
      { x: midX, y: targetY },
      { x: enterX, y: targetY },
      { x: targetX, y: targetY },
    ], radius),
  }
}

function verticalEdgeRoute(sourceX: number, sourceY: number, targetX: number, targetY: number, lane: number) {
  const radius = 18
  const exitY = sourceY + 48
  const enterY = targetY - 48
  const midY = sourceY + (targetY - sourceY) * 0.5 + lane * 0.26
  const labelX = sourceX + (targetX - sourceX) * 0.5 + lane

  return {
    labelX,
    labelY: midY,
    path: roundedOrthogonalPath([
      { x: sourceX, y: sourceY },
      { x: sourceX, y: exitY },
      { x: sourceX, y: midY },
      { x: targetX, y: midY },
      { x: targetX, y: enterY },
      { x: targetX, y: targetY },
    ], radius),
  }
}

interface RoutePoint {
  x: number
  y: number
}

function roundedOrthogonalPath(points: RoutePoint[], radius: number) {
  if (points.length < 2) {
    return ''
  }

  const commands = [`M ${points[0].x} ${points[0].y}`]

  for (let i = 1; i < points.length - 1; i++) {
    const previous = points[i - 1]
    const current = points[i]
    const next = points[i + 1]

    const before = shortenToward(current, previous, radius)
    const after = shortenToward(current, next, radius)

    commands.push(`L ${before.x} ${before.y}`)
    commands.push(`Q ${current.x} ${current.y} ${after.x} ${after.y}`)
  }

  const last = points[points.length - 1]
  commands.push(`L ${last.x} ${last.y}`)

  return commands.join(' ')
}

function shortenToward(from: RoutePoint, to: RoutePoint, radius: number): RoutePoint {
  const dx = to.x - from.x
  const dy = to.y - from.y
  const length = Math.max(Math.abs(dx), Math.abs(dy))

  if (length === 0) {
    return from
  }

  const distance = Math.min(radius, length / 2)

  return {
    x: from.x + Math.sign(dx) * distance,
    y: from.y + Math.sign(dy) * distance,
  }
}

function ArchitectureNode({ data }: NodeProps<Node<ArchitectureNodeData>>) {
  const graphNode = data.graphNode
  const risk = (graphNode.risk_points ?? 0) > 0 || (graphNode.finding_count ?? 0) > 0
  const sourcePosition = data.direction === 'DOWN' ? Position.Bottom : Position.Right
  const targetPosition = data.direction === 'DOWN' ? Position.Top : Position.Left

  return (
    <div className={['architecture-flow-node', graphNode.changed ? 'changed' : '', risk ? 'risky' : ''].join(' ')}>
      <Handle type="target" position={targetPosition} />
      <div className="architecture-flow-node-kind">{graphNode.changed ? 'changed' : 'module'}</div>
      <strong>{compactModuleLabel(graphNode.label || graphNode.id)}</strong>
      <code>{graphNode.id}</code>
      <span>
        {graphNode.before_dependency_count ?? 0} → {graphNode.after_dependency_count ?? 0}
      </span>
      <Handle type="source" position={sourcePosition} />
    </div>
  )
}

async function layoutGraph(
  graph: ReviewGraph,
  direction: LayoutDirection,
  denseLayout: boolean,
): Promise<{ nodes: Node<ArchitectureNodeData>[]; edges: Edge[] }> {
  const nodeWidth = denseLayout ? 210 : 240
  const nodeHeight = denseLayout ? 86 : 104

  const graphNodes = graph.nodes.map((node) => ({
    id: node.id,
    width: nodeWidth,
    height: nodeHeight,
  }))

  const graphEdges = graph.edges.map((edge) => ({
    id: graphEdgeKey(edge),
    sources: [edge.from],
    targets: [edge.to],
  }))

  const layout = await elk.layout({
    id: 'root',
    layoutOptions: {
      'elk.algorithm': 'layered',
      'elk.direction': direction,
      'elk.spacing.nodeNode': denseLayout ? '68' : direction === 'DOWN' ? '104' : '84',
      'elk.spacing.edgeEdge': denseLayout ? '38' : '52',
      'elk.spacing.edgeNode': denseLayout ? '34' : '44',
      'elk.layered.spacing.nodeNodeBetweenLayers': denseLayout ? '126' : direction === 'DOWN' ? '176' : '148',
      'elk.layered.spacing.edgeNodeBetweenLayers': denseLayout ? '44' : '58',
      'elk.layered.spacing.edgeEdgeBetweenLayers': denseLayout ? '38' : '48',
      'elk.layered.nodePlacement.strategy': 'NETWORK_SIMPLEX',
      'elk.layered.crossingMinimization.strategy': 'LAYER_SWEEP',
      'elk.edgeRouting': 'ORTHOGONAL',
    },
    children: graphNodes,
    edges: graphEdges,
  })

  const byID = new Map(graph.nodes.map((node) => [node.id, node]))

  const nodes: Node<ArchitectureNodeData>[] = (layout.children ?? []).map((node) => {
    const graphNode = byID.get(node.id)
    if (!graphNode) {
      throw new Error(`missing graph node ${node.id}`)
    }

    return {
      id: node.id,
      type: 'architectureNode',
      position: {
        x: node.x ?? 0,
        y: node.y ?? 0,
      },
      data: {
        graphNode,
        direction,
      },
    }
  })

  const edges: Edge<ArchitectureEdgeData>[] = graph.edges.map((edge, index) => {
    const selectedClass = edge.finding_ids?.length ? 'has-findings' : ''

    return {
      id: graphEdgeKey(edge),
      source: edge.from,
      target: edge.to,
      type: 'architectureEdge',
      markerEnd: {
        type: MarkerType.ArrowClosed,
      },
      className: ['architecture-flow-edge', edge.movement, selectedClass].filter(Boolean).join(' '),
      data: {
        edge,
        lane: edgeLane(index, denseLayout),
        direction,
      },
    }
  })

  return { nodes, edges }
}

function groupGraphByParentModules(graph: ReviewGraph): ReviewGraph {
  const parentCounts = new Map<string, number>()

  for (const node of graph.nodes) {
    const parent = parentModuleID(node.id)
    if (!parent || parent === node.id) {
      continue
    }
    parentCounts.set(parent, (parentCounts.get(parent) ?? 0) + 1)
  }

  const groupID = (id: string) => {
    const parent = parentModuleID(id)
    if (!parent || parent === id) {
      return id
    }
    return (parentCounts.get(parent) ?? 0) >= 2 ? parent : id
  }

  const nodes = new Map<string, ReviewGraphNode>()
  const ensureNode = (id: string) => {
    const existing = nodes.get(id)
    if (existing) return existing

    const node: ReviewGraphNode = {
      id,
      label: id,
    }
    nodes.set(id, node)
    return node
  }

  for (const node of graph.nodes) {
    const groupedID = groupID(node.id)
    const grouped = ensureNode(groupedID)

    grouped.before_dependency_count = (grouped.before_dependency_count ?? 0) + (node.before_dependency_count ?? 0)
    grouped.after_dependency_count = (grouped.after_dependency_count ?? 0) + (node.after_dependency_count ?? 0)
    grouped.finding_count = (grouped.finding_count ?? 0) + (node.finding_count ?? 0)
    grouped.risk_points = (grouped.risk_points ?? 0) + (node.risk_points ?? 0)
    grouped.changed = Boolean(grouped.changed || node.changed)
  }

  const edgeCounts = new Map<string, ReviewGraphEdge>()

  for (const edge of graph.edges) {
    const from = groupID(edge.from)
    const to = groupID(edge.to)
    if (!from || !to || from === to) {
      continue
    }

    ensureNode(from)
    ensureNode(to)

    const key = `${from}\u0000${to}`
    const existing = edgeCounts.get(key)

    if (!existing) {
      edgeCounts.set(key, {
        from,
        to,
        before_count: edge.before_count ?? 0,
        after_count: edge.after_count ?? 0,
        movement: edge.movement,
        finding_ids: [...(edge.finding_ids ?? [])],
      })
      continue
    }

    existing.before_count = (existing.before_count ?? 0) + (edge.before_count ?? 0)
    existing.after_count = (existing.after_count ?? 0) + (edge.after_count ?? 0)
    existing.finding_ids = uniqueStrings([...(existing.finding_ids ?? []), ...(edge.finding_ids ?? [])])
    existing.movement = movementFromCounts(existing.before_count ?? 0, existing.after_count ?? 0)
  }

  return {
    ...graph,
    source: graph.source ? `${graph.source}:grouped` : 'grouped',
    nodes: [...nodes.values()].sort((a, b) => a.id.localeCompare(b.id)),
    edges: [...edgeCounts.values()].sort((a, b) => `${a.from}->${a.to}`.localeCompare(`${b.from}->${b.to}`)),
  }
}

function parentModuleID(id: string) {
  const parts = id.split('/').filter(Boolean)
  if (parts.length <= 1) {
    return id
  }

  return parts.slice(0, -1).join('/')
}

function movementFromCounts(before: number, after: number) {
  if (before === 0 && after > 0) return 'added'
  if (before > 0 && after === 0) return 'removed'
  if (before !== after) return 'changed'
  return 'unchanged'
}

function uniqueStrings(values: string[]) {
  return [...new Set(values)].sort()
}

function filterGraph(graph: ReviewGraph, scope: DependencyScope): ReviewGraph {
  if (scope === 'all') {
    return graph
  }

  const changedEdges = graph.edges.filter((edge) => edge.movement !== 'unchanged' || (edge.finding_ids?.length ?? 0) > 0)

  if (changedEdges.length === 0) {
    return {
      ...graph,
      nodes: graph.nodes.slice(0, 18),
      edges: graph.edges.slice(0, 32),
    }
  }

  const nodeIDs = new Set<string>()
  for (const edge of changedEdges) {
    nodeIDs.add(edge.from)
    nodeIDs.add(edge.to)
  }

  return {
    ...graph,
    nodes: graph.nodes.filter((node) => nodeIDs.has(node.id)),
    edges: changedEdges,
  }
}

function edgeLane(index: number, denseLayout: boolean) {
  const lanes = denseLayout
    ? [-108, -78, -52, -26, 0, 26, 52, 78, 108]
    : [-156, -120, -84, -48, -16, 16, 48, 84, 120, 156]

  return lanes[index % lanes.length]
}

function graphEdgeKey(edge: ReviewGraphEdge) {
  return `${edge.from}->${edge.to}:${edge.before_count ?? 0}:${edge.after_count ?? 0}:${edge.movement}`
}

function compactModuleLabel(value: string) {
  const parts = value.split('/').filter(Boolean)
  if (parts.length <= 2) {
    return value
  }

  return parts.slice(-2).join('/')
}
