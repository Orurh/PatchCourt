import { useEffect, useMemo, useState } from 'react'
import { createReview } from '../api'
import type { CreateReviewResponse, GitBranch, GitCommit, GitStatus } from '../types'

interface Props {
  status: GitStatus | null
  branches: GitBranch[]
  selectedRef: string
  onSelectedRefChange: (ref: string) => void
  commits: GitCommit[]
  loading: boolean
  onReviewGenerated: (response: CreateReviewResponse) => void
  onError: (message: string) => void
}

type SelectionTarget = 'base' | 'head'

const laneColors = ['blue', 'pink', 'green', 'yellow', 'purple', 'cyan']

export function RepositoryPanel({
  status,
  branches,
  selectedRef,
  onSelectedRefChange,
  commits,
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
      if (seen.has(branch.name)) return false
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

  function selectCommit(commit: GitCommit) {
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
    if (head === 'HEAD' && currentHead) {
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

            <label className="graph-ref-selector">
              <span>Branch</span>
              <select value={selectedRef} onChange={(event) => onSelectedRefChange(event.target.value)}>
                {branchOptions.map((branch) => (
                  <option key={`${branch.kind}-${branch.name}`} value={branch.name}>
                    {branch.kind === 'remote' ? 'remote: ' : 'local: '}
                    {branch.name}
                    {branch.current ? ' ← current' : ''}
                  </option>
                ))}
              </select>
            </label>
          </div>

          <div className="commit-timeline git-graph-timeline">
            {status?.has_worktree_changes && (
              <button
                type="button"
                className={['commit-row graph-row working-tree-row', worktree ? 'selected-head' : ''].join(' ')}
                onClick={selectLocalChanges}
              >
                <GraphCell lane={0} pseudo="local" />
                <span className="commit-main graph-commit-main">
                  <span className="commit-title">
                    <span className="tag warn">local changes</span>
                    <strong>Uncommitted changes</strong>
                  </span>
                  <span className="commit-meta">
                    <span>working tree</span>
                    {worktree && <span className="tag warn">head</span>}
                  </span>
                </span>
                <span className="commit-date">*</span>
                <span className="commit-short">*</span>
              </button>
            )}

            {commits.slice(0, 100).map((commit, index) => {
              const lane = laneForCommit(commit, index)
              const refs = normalizedRefs(commit.refs)
              const isBase = commit.hash === base
              const isHead = commit.hash === head && !worktree
              const isMerge = (commit.parents?.length ?? 0) > 1

              return (
                <button
                  type="button"
                  key={commit.hash}
                  className={[
                    'commit-row',
                    'graph-row',
                    isBase ? 'selected-base' : '',
                    isHead ? 'selected-head' : '',
                    isMerge ? 'merge-row' : '',
                  ].join(' ')}
                  onClick={() => selectCommit(commit)}
                >
                  <GraphCell lane={lane} merge={isMerge} />

                  <span className="commit-main graph-commit-main">
                    <span className="commit-title">
                      {refs.map((ref) => (
                        <span className={refClassName(ref)} key={`${commit.hash}-${ref}`}>
                          {ref}
                        </span>
                      ))}
                      <code>{commit.short_hash}</code>
                      <span>{commit.message}</span>
                    </span>
                    <span className="commit-meta">
                      <span>{commit.author}</span>
                      {isMerge && <span className="tag">merge</span>}
                      {isBase && <span className="tag">base</span>}
                      {isHead && <span className="tag warn">head</span>}
                    </span>
                  </span>

                  <span className="commit-date">{formatDate(commit.date)}</span>
                  <span className="commit-short">{commit.short_hash}</span>
                </button>
              )
            })}
          </div>
        </div>
      </div>
    </section>
  )
}

function GraphCell({ lane, merge = false, pseudo }: { lane: number; merge?: boolean; pseudo?: 'local' }) {
  const lanes = [0, 1, 2, 3, 4]
  const color = laneColors[lane % laneColors.length]

  return (
    <span className="graph-cell" aria-hidden="true">
      {lanes.map((item) => (
        <span
          key={item}
          className={[
            'graph-lane',
            `lane-${laneColors[item % laneColors.length]}`,
            item === lane ? 'active' : '',
            merge && (item === lane || item === lane + 1) ? 'merge-active' : '',
          ].join(' ')}
          style={{ left: `${item * 14 + 10}px` }}
        />
      ))}

      {merge && <span className={`graph-merge-arc lane-${color}`} />}
      <span className={['graph-dot', `lane-${color}`, pseudo === 'local' ? 'local-dot' : ''].join(' ')} style={{ left: `${lane * 14 + 5}px` }} />
    </span>
  )
}

function laneForCommit(commit: GitCommit, index: number) {
  const refs = normalizedRefs(commit.refs)
  if (refs.some((ref) => ref.includes('origin/') || ref.includes('main'))) return 1
  if ((commit.parents?.length ?? 0) > 1) return 2
  if (index % 11 === 0) return 3
  if (index % 7 === 0) return 2
  return 0
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

function shortRef(value: string) {
  if (value.length > 12 && /^[0-9a-f]{12,40}$/i.test(value)) {
    return value.slice(0, 7)
  }

  return value
}
