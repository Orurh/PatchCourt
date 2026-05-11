import { useEffect, useState } from 'react'
import {
  fetchContracts,
  fetchDependencies,
  fetchFindings,
  fetchGitBranches,
  fetchGitGraph,
  fetchGitStatus,
  fetchReview,
  fetchReviewGraph,
  fetchRuntime,
  fetchTree,
} from './api'
import { RepositoryPanel } from './components/RepositoryPanel'
import { ReviewDashboard } from './components/ReviewDashboard'
import type {
  ContractsReport,
  DependenciesReport,
  FindingsReport,
  GitBranch,
  GitGraphCommit,
  GitGraphLayout,
  GitStatus,
  ReviewGraph,
  ReviewResult,
  RuntimeReport,
  TreeReport,
} from './types'

interface BundleState {
  review: ReviewResult | null
  graph: ReviewGraph | null
  tree: TreeReport | null
  runtime: RuntimeReport | null
  findings: FindingsReport | null
  contracts: ContractsReport | null
  dependencies: DependenciesReport | null
}

interface ReadyBundleState {
  review: ReviewResult
  graph: ReviewGraph
  tree: TreeReport
  runtime: RuntimeReport
  findings: FindingsReport
  contracts: ContractsReport
  dependencies: DependenciesReport
}

const emptyBundle: BundleState = {
  review: null,
  graph: null,
  tree: null,
  runtime: null,
  findings: null,
  contracts: null,
  dependencies: null,
}

function readyBundleOrNull(bundle: BundleState): ReadyBundleState | null {
  if (
    bundle.review === null ||
    bundle.graph === null ||
    bundle.tree === null ||
    bundle.runtime === null ||
    bundle.findings === null ||
    bundle.contracts === null ||
    bundle.dependencies === null
  ) {
    return null
  }

  return {
    review: bundle.review,
    graph: bundle.graph,
    tree: bundle.tree,
    runtime: bundle.runtime,
    findings: bundle.findings,
    contracts: bundle.contracts,
    dependencies: bundle.dependencies,
  }
}

export function App() {
  const [bundle, setBundle] = useState<BundleState>(emptyBundle)
  const [gitStatus, setGitStatus] = useState<GitStatus | null>(null)
  const [branches, setBranches] = useState<GitBranch[]>([])
  const [selectedRef, setSelectedRef] = useState('')
  const [showAllBranches, setShowAllBranches] = useState(false)
  const [commits, setCommits] = useState<GitGraphCommit[]>([])
  const [graphLayout, setGraphLayout] = useState<GitGraphLayout | null>(null)
  const [loadingGit, setLoadingGit] = useState(true)
  const [loadingBundle, setLoadingBundle] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [lastReviewID, setLastReviewID] = useState<string | null>(null)

  useEffect(() => {
    void loadGitContext()
    void loadLatestBundle()
  }, [])

  useEffect(() => {
    if (selectedRef !== '') {
      void loadCommitsForRef(selectedRef, showAllBranches)
    }
  }, [selectedRef, showAllBranches])

  async function loadGitContext() {
    setLoadingGit(true)

    try {
      const [status, branchResponse] = await Promise.all([fetchGitStatus(), fetchGitBranches()])
      const initialRef = branchResponse.current || status.branch

      setGitStatus(status)
      setBranches(branchResponse.branches)
      setSelectedRef(initialRef)

      const graphResponse = await fetchGitGraph(120, initialRef, showAllBranches)
      setCommits(graphResponse.commits)
      setGraphLayout(graphResponse.layout ?? null)
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err))
    } finally {
      setLoadingGit(false)
    }
  }

  async function loadCommitsForRef(ref: string, allBranches = false) {
    try {
      const graphResponse = await fetchGitGraph(120, ref, allBranches)
      setCommits(graphResponse.commits)
      setGraphLayout(graphResponse.layout ?? null)
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err))
    }
  }

  async function loadLatestBundle() {
    setLoadingBundle(true)

    try {
      const [review, graph, tree, runtime, findings, contracts, dependencies] = await Promise.all([
        fetchReview(),
        fetchReviewGraph(),
        fetchTree(),
        fetchRuntime(),
        fetchFindings(),
        fetchContracts(),
        fetchDependencies(),
      ])

      setBundle({ review, graph, tree, runtime, findings, contracts, dependencies })
    } catch {
      setBundle(emptyBundle)
    } finally {
      setLoadingBundle(false)
    }
  }

  const readyBundle = readyBundleOrNull(bundle)

  return (
    <main className="page">
      <header className="hero">
        <div>
          <p className="eyebrow">PatchCourt</p>
          <h1>Architecture Review</h1>
          <p className="muted">Select git points, generate a review, and inspect architecture impact.</p>
        </div>

        {bundle.review ? (
          <div className={`risk-badge risk-${bundle.review.risk.level}`}>
            <span>{bundle.review.risk.level}</span>
            <strong>{bundle.review.risk.points}</strong>
            <small>points</small>
          </div>
        ) : (
          <div className="risk-badge">
            <span>No review</span>
            <strong>—</strong>
            <small>yet</small>
          </div>
        )}
      </header>

      {error && (
        <section className="card">
          <p className="error">{error}</p>
        </section>
      )}

      <RepositoryPanel
        status={gitStatus}
        branches={branches}
        selectedRef={selectedRef}
        onSelectedRefChange={setSelectedRef}
        showAllBranches={showAllBranches}
        onShowAllBranchesChange={setShowAllBranches}
        commits={commits}
        graphLayout={graphLayout}
        loading={loadingGit}
        onError={setError}
        onReviewGenerated={(response) => {
          setLastReviewID(response.id)
          setError(null)
          void loadLatestBundle()
          void loadGitContext()
        }}
      />

      {lastReviewID && (
        <section className="card">
          <p className="muted">
            Latest generated review: <code>{lastReviewID}</code>
          </p>
        </section>
      )}

      {loadingBundle && (
        <section className="card">
          <p className="muted">Loading latest review bundle…</p>
        </section>
      )}

      {!loadingBundle && readyBundle === null && (
        <section className="card">
          <h2>No latest review bundle</h2>
          <p className="muted">Choose base/head or base/worktree and click Generate review.</p>
        </section>
      )}

      {readyBundle && (
        <ReviewDashboard
          review={readyBundle.review}
          graph={readyBundle.graph}
          tree={readyBundle.tree}
          runtime={readyBundle.runtime}
          findings={readyBundle.findings}
          contracts={readyBundle.contracts}
          dependencies={readyBundle.dependencies}
        />
      )}
    </main>
  )
}
