import type { Report } from '@current-client-bindings/core/rework'
import PathText from '../../components/git/PathText'
import { plural } from '../../lib/plural'
import './Rework.scss'

const MOST_FILES = 20
const MOST_COMMIT_SQUARES = 8

interface ReworkProps {
  report: Report | null
  error: string | null
  uncommitted: boolean
  listed: (path: string) => boolean
  onOpen: (path: string) => void
}

// Lines the range wrote and then changed again before its end: where it most
// needs a careful read.
function Rework({ report, error, uncommitted, listed, onOpen }: ReworkProps) {
  if (error) {
    return (
      <section className="read-through-notes" aria-label="Rewritten along the way">
        <h3 className="changed-files-browser-label">Rewritten along the way</h3>
        <p className="changed-files-browser-error">{error}</p>
      </section>
    )
  }
  if (!report || report.rewritten === 0) return null
  const files = report.files.slice(0, MOST_FILES)
  const widest = Math.max(...files.map((f) => Math.max(f.written, f.rewritten)))
  const most = Math.max(...report.commits.map((c) => c.rewritten))
  return (
    <section className="read-through-notes" aria-label="Rewritten along the way">
      <h3 className="changed-files-browser-label">Rewritten along the way</h3>
      <p className="rework-about">
        Lines one commit wrote and a later commit in this range changed or removed. Moved lines don't count.
      </p>
      <h4 className="rework-label">By commit</h4>
      <ol
        className="rework-commits"
        aria-label={`Lines rewritten by each of ${plural(report.commits.length, 'commit')}`}
      >
        {report.commits.map((c) => (
          <li
            key={c.sha}
            className="rework-commit"
            title={`${c.subject}\n${c.sha.slice(0, 7)} · ${plural(c.rewritten, 'line')} rewritten`}
          >
            <span className="rework-commit-bar" style={{ height: `${(c.rewritten / most) * 100}%` }} />
          </li>
        ))}
      </ol>
      <h4 className="rework-label">By file</h4>
      <ul className="read-through-note-list">
        {files.map((f) => {
          const kept = Math.max(f.written - f.rewritten, 0)
          return (
            <li key={f.path}>
              <button
                type="button"
                className="rework-file"
                disabled={!listed(f.path)}
                onClick={() => onOpen(f.path)}
                aria-label={`${f.path}: ${f.rewritten} of ${plural(f.written, 'line')} written rewritten, across ${plural(f.commits, 'commit')}`}
              >
                <PathText path={f.path} className="rework-path" />
                <span className="rework-bar" aria-hidden="true">
                  <span className="rework-kept" style={{ width: `${(kept / widest) * 100}%` }} />
                  <span className="rework-rewritten" style={{ width: `${(f.rewritten / widest) * 100}%` }} />
                </span>
                <span className="rework-count">{f.rewritten}</span>
                <span className="rework-touches" aria-hidden="true">
                  {Array.from({ length: Math.min(f.commits, MOST_COMMIT_SQUARES) }, (_, i) => (
                    <span key={i} className="rework-touch" />
                  ))}
                </span>
              </button>
            </li>
          )
        })}
        {report.files.length > files.length && (
          <li className="rework-left-out">{plural(report.files.length - files.length, 'more file')}</li>
        )}
      </ul>
      {report.capped && (
        <p className="changed-files-browser-hint">Only the newest {report.commits.length} commits were read.</p>
      )}
      {uncommitted && <p className="changed-files-browser-hint">Uncommitted changes aren't counted.</p>}
    </section>
  )
}

export default Rework
