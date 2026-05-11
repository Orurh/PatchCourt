import type { GitGraphCommit, GitGraphLayout, GitStatus } from '../../types'

interface Props {
  status: GitStatus | null
  commits: GitGraphCommit[]
  layout: GitGraphLayout | null
  base: string
  head: string
  worktree: boolean
  onSelectCommit: (commit: GitGraphCommit) => void
  onSelectLocalChanges: () => void
}

type GraphRow =
  | {
      kind: 'local'
      lane: number
    }
  | {
      kind: 'commit'
      commit: GitGraphCommit
      lane: number
      isMerge: boolean
    }

const laneNames = ['blue', 'pink', 'green', 'yellow', 'purple', 'cyan'] as const
const defaultRowHeight = 54
const defaultLaneWidth = 12
const graphLeftPadding = 28
const graphWidth = 92

export function CommitGraph({
  status,
  commits,
  layout,
  base,
  head,
  worktree,
  onSelectCommit,
  onSelectLocalChanges,
}: Props) {
  const visibleCommits = commits.slice(0, 120)
  const rows = buildRows(status, visibleCommits)
  const rowHeight = layout?.row_height ?? defaultRowHeight
  const laneWidth = layout?.lane_width ?? defaultLaneWidth
  const graphHeight = Math.max(rows.length * rowHeight, rowHeight)

  return (
    <div className="pc-graph-list">
      <svg
        className="pc-graph-canvas"
        width={graphWidth}
        height={graphHeight}
        viewBox={`0 0 ${graphWidth} ${graphHeight}`}
        aria-hidden="true"
      >
        <GraphLines layout={layout} rowOffset={status?.has_worktree_changes ? 1 : 0} rowHeight={rowHeight} laneWidth={laneWidth} />
      </svg>

      {rows.map((row) => {
        if (row.kind === 'local') {
          return (
            <button
              type="button"
              key="local"
              className={['pc-graph-row', 'pc-graph-local-row', worktree ? 'pc-graph-selected-head' : ''].join(' ')}
              onClick={onSelectLocalChanges}
            >
              <GraphDot lane={row.lane} laneWidth={laneWidth} pseudo="local" />

              <span className="pc-graph-main">
                <span className="pc-graph-title">
                  <span className="tag warn">local changes</span>
                  <strong>Uncommitted changes</strong>
                </span>
                <span className="pc-graph-meta">
                  <span>working tree</span>
                  {worktree && <span className="tag warn">head</span>}
                </span>
              </span>

              <span className="pc-graph-date">*</span>
              <span className="pc-graph-short">*</span>
            </button>
          )
        }

        const commit = row.commit
        const refs = normalizedRefs(commit.refs)
        const isBase = commit.hash === base
        const isHead = commit.hash === head && !worktree

        return (
          <button
            type="button"
            key={commit.hash}
            className={[
              'pc-graph-row',
              isBase ? 'pc-graph-selected-base' : '',
              isHead ? 'pc-graph-selected-head' : '',
              row.isMerge ? 'pc-graph-merge-row' : '',
            ].join(' ')}
            onClick={() => onSelectCommit(commit)}
          >
            <GraphDot lane={row.lane} laneWidth={laneWidth} />

            <span className="pc-graph-main">
              <span className="pc-graph-title">
                {refs.map((ref) => (
                  <span className={refClassName(ref)} key={`${commit.hash}-${ref}`}>
                    {ref}
                  </span>
                ))}
                <code>{commit.short_hash}</code>
                <span>{commit.message}</span>
              </span>

              <span className="pc-graph-meta">
                <span>{commit.author}</span>
                {row.isMerge && <span className="tag">merge</span>}
                {commit.is_branch_point && <span className="tag">branch point</span>}
                {isBase && <span className="tag">base</span>}
                {isHead && <span className="tag warn">head</span>}
              </span>
            </span>

            <span className="pc-graph-date">{formatDate(commit.date)}</span>
            <span className="pc-graph-short">{commit.short_hash}</span>
          </button>
        )
      })}
    </div>
  )
}

function GraphLines({
  layout,
  rowOffset,
  rowHeight,
  laneWidth,
}: {
  layout: GitGraphLayout | null
  rowOffset: number
  rowHeight: number
  laneWidth: number
}) {
  if (!layout) {
    return null
  }

  return (
    <>
      {(layout.segments ?? []).map((segment, index) => {
        const x = laneX(segment.lane, laneWidth)
        const y1 = rowCenterY(segment.from_row + rowOffset, rowHeight)
        const y2 = rowCenterY(segment.to_row + rowOffset, rowHeight)

        return (
          <line
            key={`segment-${index}-${segment.lane}-${segment.from_row}-${segment.to_row}`}
            className="pc-graph-lane"
            stroke={laneColorValue(segment.lane)}
            x1={x}
            x2={x}
            y1={y1}
            y2={y2}
          />
        )
      })}

      {(layout.edges ?? []).map((edge, index) => {
        const fromY = rowCenterY(edge.from_row + rowOffset, rowHeight)
        const toY = rowCenterY(edge.to_row + rowOffset, rowHeight)

        return (
          <path
            key={`edge-${index}-${edge.from_lane}-${edge.to_lane}-${edge.from_row}-${edge.to_row}`}
            className={`pc-graph-link pc-graph-link-${edge.kind}`}
            stroke={laneColorValue(edge.from_lane)}
            d={edgePath(edge.from_lane, edge.to_lane, fromY, toY, laneWidth)}
          />
        )
      })}
    </>
  )
}

function GraphDot({ lane, laneWidth, pseudo }: { lane: number; laneWidth: number; pseudo?: 'local' }) {
  const currentLane = clampLane(lane)

  return (
    <span className="pc-graph-cell">
      <span
        className={['pc-graph-dot', pseudo === 'local' ? 'pc-graph-local-dot' : ''].join(' ')}
        style={{
          left: `${laneX(currentLane, laneWidth) - 9}px`,
          backgroundColor: laneColorValue(currentLane),
        }}
      />
    </span>
  )
}

function buildRows(status: GitStatus | null, commits: GitGraphCommit[]): GraphRow[] {
  const rows: GraphRow[] = []

  if (status?.has_worktree_changes) {
    rows.push({
      kind: 'local',
      lane: 0,
    })
  }

  for (const commit of commits) {
    rows.push({
      kind: 'commit',
      commit,
      lane: clampLane(commit.lane ?? 0),
      isMerge: Boolean(commit.is_merge) || (commit.parents?.length ?? 0) > 1,
    })
  }

  return rows
}

function edgePath(fromLane: number, toLane: number, fromY: number, toY: number, laneWidth: number) {
  const fromX = laneX(fromLane, laneWidth)
  const toX = laneX(toLane, laneWidth)
  const midY = fromY + (toY - fromY) * 0.55

  return `M ${fromX} ${fromY} C ${fromX} ${midY}, ${toX} ${midY}, ${toX} ${toY}`
}

function rowCenterY(index: number, rowHeight: number) {
  return index * rowHeight + rowHeight / 2
}

function laneX(lane: number, laneWidth: number) {
  return clampLane(lane) * laneWidth + graphLeftPadding
}

function clampLane(lane: number) {
  if (!Number.isFinite(lane)) return 0
  if (lane < 0) return 0
  if (lane > 5) return 5
  return lane
}

function laneColorValue(lane: number) {
  switch (laneNames[lane % laneNames.length]) {
    case 'blue':
      return '#3b82f6'
    case 'pink':
      return '#db2777'
    case 'green':
      return '#22c55e'
    case 'yellow':
      return '#f59e0b'
    case 'purple':
      return '#8b5cf6'
    case 'cyan':
      return '#06b6d4'
    default:
      return '#64748b'
  }
}

function normalizedRefs(refs?: string[] | null) {
  return (refs ?? [])
    .map((ref) => ref.replace(/^tag:\s*/, 'tag: '))
    .filter(Boolean)
}

function refClassName(ref: string) {
  if (ref.startsWith('tag:')) return 'ref-pill ref-tag'
  if (ref.startsWith('origin/')) return 'ref-pill ref-remote'
  return 'ref-pill ref-local'
}

function formatDate(value: string) {
  if (!value) return ''

  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return ''

  return new Intl.DateTimeFormat(undefined, {
    month: 'short',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  }).format(date)
}
