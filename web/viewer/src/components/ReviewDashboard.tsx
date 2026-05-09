import { useMemo, useState } from 'react'
import type {
  ContractChange,
  ContractsReport,
  DependenciesReport,
  DependencyChange,
  FindingChange,
  FindingsReport,
  ReviewGraph,
  ReviewResult,
  RuntimeReport,
  TreeReport,
} from '../types'
import { EvidenceList, Metric, severityRank } from './common'
import { ProjectTree } from './ProjectTree'

type Tab = 'overview' | 'tree' | 'runtime' | 'findings' | 'contracts' | 'dependencies'

interface Props {
  review: ReviewResult
  graph: ReviewGraph
  tree: TreeReport
  runtime: RuntimeReport
  findings: FindingsReport
  contracts: ContractsReport
  dependencies: DependenciesReport
}

export function ReviewDashboard({ review, graph, tree, runtime, findings, contracts, dependencies }: Props) {
  const [activeTab, setActiveTab] = useState<Tab>('overview')

  const tabs: Array<[Tab, string]> = [
    ['overview', 'Overview'],
    ['tree', 'Tree'],
    ['runtime', 'Runtime'],
    ['findings', 'Findings'],
    ['contracts', 'Contracts'],
    ['dependencies', 'Dependencies'],
  ]

  return (
    <>
      <nav className="tabs">
        {tabs.map(([tab, label]) => (
          <button key={tab} className={activeTab === tab ? 'active' : ''} onClick={() => setActiveTab(tab)}>
            {label}
          </button>
        ))}
      </nav>

      {activeTab === 'overview' && <Overview review={review} />}
      {activeTab === 'tree' && <ProjectTree tree={tree} graph={graph} />}
      {activeTab === 'runtime' && <RuntimeView runtime={runtime} />}
      {activeTab === 'findings' && <FindingsView findings={findings} />}
      {activeTab === 'contracts' && <ContractsView contracts={contracts} />}
      {activeTab === 'dependencies' && <DependenciesView dependencies={dependencies} />}
    </>
  )
}

function Overview({ review }: { review: ReviewResult }) {
  return (
    <section className="stack">
      <div className="grid">
        <Metric value={review.summary.contract_changes} label="Contract changes" />
        <Metric value={review.summary.dependency_changes} label="Dependency changes" />
        <Metric value={review.summary.layer_edge_changes} label="Layer edge changes" />
        <Metric value={review.summary.finding_changes} label="Finding changes" />
        <Metric value={review.summary.added_findings} label="Added findings" />
        <Metric value={review.summary.added_high_findings} label="Added high findings" />
      </div>

      <section className="card">
        <h2>Risk reasons</h2>
        {review.risk.reasons?.length ? (
          <ul className="reason-list">
            {review.risk.reasons.map((reason) => (
              <li key={`${reason.points}-${reason.message}`}>
                <strong>+{reason.points}</strong>
                <span>{reason.message}</span>
              </li>
            ))}
          </ul>
        ) : (
          <p className="muted">No risk reasons.</p>
        )}
      </section>

      <section className="card">
        <h2>Changed files</h2>
        {review.changed_files?.length ? (
          <ul className="compact-list">
            {review.changed_files.map((file) => (
              <li key={file}>
                <code>{file}</code>
              </li>
            ))}
          </ul>
        ) : (
          <p className="muted">No changed files.</p>
        )}
      </section>
    </section>
  )
}

function RuntimeView({ runtime }: { runtime: RuntimeReport }) {
  const sortedChanges = useMemo(
    () =>
      [...runtime.changes].sort(
        (a, b) => severityRank(b.after_severity || b.before_severity) - severityRank(a.after_severity || a.before_severity),
      ),
    [runtime.changes],
  )

  return (
    <section className="stack">
      <div className="grid">
        <Metric value={runtime.summary.change_count} label="Runtime changes" />
        <Metric value={runtime.summary.high_count} label="High" />
        <Metric value={runtime.summary.medium_count} label="Medium" />
        <Metric value={runtime.summary.low_count} label="Low" />
      </div>

      {sortedChanges.map((change) => (
        <article className="card" key={change.id}>
          <div className="card-title-row">
            <h2><code>{change.id}</code></h2>
            <div>
              <span className="tag">{change.kind}</span>
              {change.after_severity && <span className={`tag severity-${change.after_severity}`}>{change.after_severity}</span>}
              {change.after_confidence && <span className="tag">{change.after_confidence}</span>}
            </div>
          </div>

          {change.title && <p className="lead">{change.title}</p>}
          {change.risk && <p><strong>Risk:</strong> {change.risk}</p>}
          {change.suggestion && <p><strong>Suggestion:</strong> {change.suggestion}</p>}
          <EvidenceList evidence={change.evidence ?? []} />
        </article>
      ))}
    </section>
  )
}

function FindingsView({ findings }: { findings: FindingsReport }) {
  return (
    <section className="stack">
      <div className="grid">
        <Metric value={findings.summary.change_count ?? 0} label="Finding changes" />
        <Metric value={findings.summary.added_count ?? 0} label="Added" />
        <Metric value={findings.summary.removed_count ?? 0} label="Removed" />
        <Metric value={findings.summary.changed_count ?? 0} label="Changed" />
      </div>

      {findings.changes.map((change) => (
        <FindingChangeCard key={`${change.kind}-${change.id}`} change={change} />
      ))}
    </section>
  )
}

function FindingChangeCard({ change }: { change: FindingChange }) {
  const finding = change.after ?? change.before
  const evidence = change.added_evidence?.length
    ? change.added_evidence
    : change.removed_evidence?.length
      ? change.removed_evidence
      : finding?.evidence ?? []

  return (
    <article className="card">
      <div className="card-title-row">
        <h2><code>{change.id}</code></h2>
        <div>
          <span className="tag">{change.kind}</span>
          {finding?.kind && <span className="tag">{finding.kind}</span>}
          {finding?.severity && <span className={`tag severity-${finding.severity}`}>{finding.severity}</span>}
          {finding?.confidence && <span className="tag">{finding.confidence}</span>}
        </div>
      </div>

      {finding?.title && <p className="lead">{finding.title}</p>}
      {finding?.risk && <p><strong>Risk:</strong> {finding.risk}</p>}
      {finding?.suggestion && <p><strong>Suggestion:</strong> {finding.suggestion}</p>}
      <EvidenceList evidence={evidence} />
    </article>
  )
}

function ContractsView({ contracts }: { contracts: ContractsReport }) {
  return (
    <section className="stack">
      <div className="grid">
        <Metric value={contracts.summary.change_count ?? 0} label="Contract changes" />
        <Metric value={contracts.summary.added_count ?? 0} label="Added" />
        <Metric value={contracts.summary.removed_count ?? 0} label="Removed" />
        <Metric value={contracts.summary.changed_count ?? 0} label="Changed" />
        <Metric value={contracts.summary.impact_count ?? 0} label="Impacts" />
      </div>

      <section className="card">
        <h2>Contract changes</h2>
        <div className="table-list">
          {contracts.changes.map((change) => (
            <ContractRow key={`${change.kind}-${change.symbol_key}`} change={change} />
          ))}
        </div>
      </section>

      {contracts.impacts?.length ? (
        <section className="card">
          <h2>Contract impacts</h2>
          <ul className="compact-list">
            {contracts.impacts.map((impact) => (
              <li key={`${impact.symbol_key}-${impact.change_kind}-${impact.location}`}>
                <span className="tag">{impact.change_kind}</span>
                <code>{impact.symbol_key}</code>
                <span>{impact.impact}</span>
                {impact.delivery_impacted && <span className="tag warn">delivery/API</span>}
                {!impact.tests_changed && <span className="tag bad">no tests</span>}
              </li>
            ))}
          </ul>
        </section>
      ) : null}
    </section>
  )
}

function ContractRow({ change }: { change: ContractChange }) {
  return (
    <div className="row-card">
      <div>
        <span className="tag">{change.kind}</span>{' '}
        <code>{change.symbol_key}</code>
      </div>
      {change.before?.file && <p className="muted">before: <code>{change.before.file}</code></p>}
      {change.after?.file && <p className="muted">after: <code>{change.after.file}</code></p>}
      {change.before?.signature && <pre>{change.before.signature}</pre>}
      {change.after?.signature && <pre>{change.after.signature}</pre>}
    </div>
  )
}

function DependenciesView({ dependencies }: { dependencies: DependenciesReport }) {
  return (
    <section className="stack">
      <div className="grid">
        <Metric value={dependencies.summary.dependency_change_count ?? 0} label="Dependency changes" />
        <Metric value={dependencies.summary.layer_edge_change_count ?? 0} label="Layer edge changes" />
        <Metric value={dependencies.summary.added_dependencies ?? 0} label="Added deps" />
        <Metric value={dependencies.summary.added_layer_edges ?? 0} label="Added layer edges" />
      </div>

      <section className="card">
        <h2>Layer edge changes</h2>
        {dependencies.layer_edge_changes.length ? (
          <ul className="compact-list">
            {dependencies.layer_edge_changes.map((edge) => (
              <li key={`${edge.kind}-${edge.from_layer}-${edge.to_layer}`}>
                <span className="tag">{edge.kind}</span>
                <code>{edge.from_layer} → {edge.to_layer}</code>
                <span>{edge.before_count ?? 0} → {edge.after_count ?? 0}</span>
              </li>
            ))}
          </ul>
        ) : (
          <p className="muted">No layer edge changes.</p>
        )}
      </section>
    </section>
  )
}
