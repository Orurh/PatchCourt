import type { Evidence } from '../types'

export function Metric({ value, label }: { value: number; label: string }) {
  return (
    <article className="metric">
      <strong>{value}</strong>
      <span>{label}</span>
    </article>
  )
}

export function EvidenceList({ evidence }: { evidence: Evidence[] }) {
  if (!evidence.length) {
    return <p className="muted">No evidence.</p>
  }

  return (
    <div className="evidence-list">
      {evidence.map((item, index) => (
        <div className="evidence" key={`${item.file}-${item.line_start}-${index}`}>
          <div>
            <code>
              {item.file}
              {item.line_start ? `:${item.line_start}` : ''}
            </code>
          </div>
          {item.message && <p>{item.message}</p>}
          {item.snippet && <pre>{item.snippet}</pre>}
        </div>
      ))}
    </div>
  )
}

export function severityRank(severity?: string) {
  if (severity === 'critical') return 4
  if (severity === 'high') return 3
  if (severity === 'medium') return 2
  if (severity === 'low') return 1
  return 0
}
