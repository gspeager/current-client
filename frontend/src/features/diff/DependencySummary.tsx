import type { DependencyChangeInfo } from '@current-client-bindings/app'
import './DependencySummary.scss'

interface DependencySummaryProps {
  changes: DependencyChangeInfo[]
}

const NO_VERSION = 'no version'

function countsOf(changes: DependencyChangeInfo[]): string {
  const count = (kind: string) => changes.filter((change) => change.kind === kind).length
  return ['bumped', 'added', 'removed']
    .map((kind) => [count(kind), kind] as const)
    .filter(([n]) => n > 0)
    .map(([n, kind]) => `${n} ${kind}`)
    .join(', ')
}

function DependencySummary({ changes }: DependencySummaryProps) {
  return (
    <div className="dependency-summary">
      <p className="dependency-summary-counts">{countsOf(changes)}</p>
      <ul className="dependency-summary-list">
        {changes.map((change) => (
          <li key={`${change.name}-${change.from}-${change.to}`} className="dependency-summary-row">
            <span className="dependency-summary-name" title={change.name}>
              {change.name}
            </span>
            <span className="dependency-summary-kind">{change.kind}</span>
            <span className="dependency-summary-versions">
              {change.kind === 'bumped'
                ? `${change.from || NO_VERSION} → ${change.to || NO_VERSION}`
                : change.from || change.to || NO_VERSION}
            </span>
          </li>
        ))}
      </ul>
      <p className="dependency-summary-note">From the file's two versions in Git. Not a vulnerability check.</p>
    </div>
  )
}

export default DependencySummary
