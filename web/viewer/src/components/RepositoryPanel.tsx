import { useEffect, useMemo, useState } from 'react'
import { createReview } from '../api'
import { CommitGraph } from './git/CommitGraph'
import type { CreateReviewResponse, GitBranch, GitGraphCommit, GitGraphLayout, GitStatus } from '../types'

interface Props {
  status: GitStatus | null
  branches: GitBranch[]
  selectedRef: string
  onSelectedRefChange: (ref: string) => void
  showAllBranches: boolean
  onShowAllBranchesChange: (show: boolean) => void
  commits: GitGraphCommit[]
  graphLayout: GitGraphLayout | null
  loading: boolean
  onReviewGenerated: (response: CreateReviewResponse) => void
  onError: (message: string) => void
}

type SelectionTarget = 'base' | 'head'

export function RepositoryPanel({
  status,
  branches,
  selectedRef,
  onSelectedRefChange,
  showAllBranches,
  onShowAllBranchesChange,
  commits,
  graphLayout,
  loading,
  onReviewGenerated,
  onError,
}: Props) {
  const [base, setBase] = useState('HEAD')
  const [head, setHead] = useState('HEAD')
  const [worktree, setWorktree] = useState(true)
  const [selectionTarget, setSelectionTarget] = useState<SelectionTarget>('base')
  const [generating, setGenerating] = useState(false)

  const currentHead = commits[0]

  useEffect(() => {
    if (currentHead && (base === 'HEAD' || base === '')) {
      setBase(currentHead.hash)
    }
  }, [currentHead, base])

  const branchOptions = useMemo(() => {
    const seen = new Set<string>()

    return branches.filter((branch) => {
      if (seen.has(branch.name)) {
        return false
      }

      seen.add(branch.name)
      return true
    })
  }, [branches])

  async function handleGenerate() {
    setGenerating(true)

    try {
      const response = await createReview({
        base,
        head: worktree ? undefined : head,
        worktree,
      })

      onReviewGenerated(response)
    } catch (err) {
      onError(err instanceof Error ? err.message : String(err))
    } finally {
      setGenerating(false)
    }
  }

  function selectCommit(commit: GitGraphCommit) {
    if (selectionTarget === 'base' || worktree) {
      setBase(commit.hash)
      return
    }

    setHead(commit.hash)
  }

  function selectLocalChanges() {
    setWorktree(true)
    setSelectionTarget('base')
  }

  function enableHeadSelection() {
    setWorktree(false)
    setSelectionTarget('head')

    if ((head === 'HEAD' || head === '') && currentHead) {
      setHead(currentHead.hash)
    }
  }

  return (
    <section className="card repository-card">
      <div className="card-title-row">
        <div>
          <p className="eyebrow">Repository</p>
          <h2>Compare changes</h2>
        </div>

        {status && (
          <div className="repo-status">
            <span className="tag">{status.branch}</span>
            <code>{status.short_head}</code>
            {status.has_worktree_changes && <span className="tag warn">dirty worktree</span>}
            {status.upstream && <span className="tag">{status.upstream}</span>}
            {typeof status.ahead === 'number' && status.ahead > 0 && <span className="tag">ahead {status.ahead}</span>}
            {typeof status.behind === 'number' && status.behind > 0 && <span className="tag warn">behind {status.behind}</span>}
          </div>
        )}
      </div>

      {loading && <p className="muted">Loading git status…</p>}

      <div className="compare-layout graph-first-layout">
        <aside className="compare-control-panel">
          <div className="selector-row">
            <button
              type="button"
              className={selectionTarget === 'base' ? 'active' : ''}
              onClick={() => setSelectionTarget('base')}
            >
              Pick base
            </button>
            <button
              type="button"
              className={selectionTarget === 'head' && !worktree ? 'active' : ''}
              onClick={enableHeadSelection}
            >
              Pick head
            </button>
          </div>

          <button
            type="button"
            className={worktree ? 'local-toggle active' : 'local-toggle'}
            onClick={selectLocalChanges}
          >
            Use local changes
          </button>

          <div className="compare-summary compare-summary-large">
            <div>
              <span className="muted">Base</span>
              <code>{shortRef(base)}</code>
            </div>
            <div>
              <span className="muted">Head</span>
              <code>{worktree ? 'working tree' : shortRef(head)}</code>
            </div>
          </div>

          <p className="muted compare-help">
            Select comparison points by clicking commits in the graph. By default PatchCourt compares the latest commit with local changes.
          </p>

          <button className="primary-button" disabled={generating || !base.trim()} onClick={handleGenerate}>
            {generating ? 'Generating…' : 'Generate review'}
          </button>
        </aside>

        <div className="commit-panel git-graph-panel">
          <div className="commit-panel-header">
            <div>
              <h3>Commit graph</h3>
              <p className="muted">Click commits to select base/head. Refs are shown as pills.</p>
            </div>

            <div className="graph-ref-controls">
              <label className="graph-ref-selector">
                <span>Branch</span>
                <select
                  value={selectedRef}
                  disabled={showAllBranches}
                  onChange={(event) => onSelectedRefChange(event.target.value)}
                >
                  {branchOptions.map((branch) => (
                    <option key={`${branch.kind}-${branch.name}`} value={branch.name}>
                      {branch.kind === 'remote' ? 'remote: ' : 'local: '}
                      {branch.name}
                      {branch.current ? ' ← current' : ''}
                    </option>
                  ))}
                </select>
              </label>

              <label className="graph-all-toggle">
                <input
                  type="checkbox"
                  checked={showAllBranches}
                  onChange={(event) => onShowAllBranchesChange(event.target.checked)}
                />
                <span>Show all branches</span>
              </label>
            </div>
          </div>

          <CommitGraph
            status={status}
            commits={commits}
            layout={graphLayout}
            base={base}
            head={head}
            worktree={worktree}
            onSelectCommit={selectCommit}
            onSelectLocalChanges={selectLocalChanges}
          />
        </div>
      </div>
    </section>
  )
}

function shortRef(value: string) {
  if (value.length > 12 && /^[0-9a-f]{12,40}$/i.test(value)) {
    return value.slice(0, 7)
  }

  return value
}
