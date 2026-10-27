import { useState, type FormEvent } from 'react'
import { CircleCheck, CircleHelp, Info, TriangleAlert, type LucideIcon } from 'lucide-react'
import { ChangelogService, type ReleaseReadiness } from '@current-client-bindings/app'
import Modal from '../../components/chrome/Modal'
import { errorMessage } from '../../lib/errors'
import { useAsyncData } from '../../lib/useAsyncData'
import './ReleaseReadinessDialog.scss'

interface ReleaseReadinessDialogProps {
  repoPath: string
  onClose: () => void
}

const LEVELS: Record<string, { icon: LucideIcon; label: string }> = {
  ok: { icon: CircleCheck, label: 'Ready' },
  warn: { icon: TriangleAlert, label: 'Needs attention' },
  info: { icon: Info, label: 'For information' },
  unknown: { icon: CircleHelp, label: 'Not known' },
}

// A checklist for a candidate against the release before it, each item
// with what it rests on. Git answers all but CI and protection, which come
// from the host's last refresh. It publishes nothing.
function ReleaseReadinessDialog({ repoPath, onClose }: ReleaseReadinessDialogProps) {
  const [asked, setAsked] = useState({ ref: '', version: '', target: '', run: 0 })
  const { data, error } = useAsyncData(
    () => ChangelogService.ReleaseReadiness(repoPath, asked.ref, asked.version, asked.target),
    [repoPath, asked],
    { keepData: true },
  )
  return (
    <Modal title="Release readiness" onClose={onClose} className="dialog-panel release-readiness-panel">
      <div className="dialog-body">
        {data && (
          <ReadinessForm
            key={`${data.sha}-${asked.run}`}
            result={data}
            onCheck={(next) => setAsked({ ...next, run: asked.run + 1 })}
          />
        )}
        {error && (
          <p className="dialog-error" role="alert">
            {error}
          </p>
        )}
        {!data && !error && <p className="release-readiness-hint">Checking…</p>}
        {data && (
          <ul className="release-readiness-list" aria-label="Checks">
            {data.checks.map((check) => {
              const level = LEVELS[check.level] ?? LEVELS.unknown
              const Icon = level.icon
              return (
                <li key={check.label} className={`release-check release-check-${check.level}`}>
                  <Icon size={14} strokeWidth={1.75} aria-label={level.label} role="img" />
                  <span className="release-check-label">{check.label}</span>
                  <span className="release-check-value">{check.value}</span>
                  {check.evidence?.length > 0 && (
                    <ul className="release-check-evidence">
                      {check.evidence.map((line, i) => (
                        <li key={i}>{line}</li>
                      ))}
                    </ul>
                  )}
                </li>
              )
            })}
          </ul>
        )}
        <VersionFiles repoPath={repoPath} onSaved={() => setAsked({ ...asked, run: asked.run + 1 })} />
        <div className="dialog-actions">
          <button type="button" className="dialog-button" onClick={onClose}>
            Close
          </button>
        </div>
      </div>
    </Modal>
  )
}

function ReadinessForm({
  result,
  onCheck,
}: {
  result: ReleaseReadiness
  onCheck: (asked: { ref: string; version: string; target: string }) => void
}) {
  const [ref, setRef] = useState(result.ref)
  const [version, setVersion] = useState(result.version)
  const [target, setTarget] = useState(result.target)
  const submit = (e: FormEvent) => {
    e.preventDefault()
    onCheck({ ref: ref.trim(), version: version.trim(), target: target.trim() })
  }
  return (
    <form className="release-readiness-form" onSubmit={submit}>
      <label className="dialog-field">
        <span className="dialog-message">Candidate</span>
        <input className="dialog-input" value={ref} onChange={(e) => setRef(e.target.value)} spellCheck={false} />
      </label>
      <label className="dialog-field">
        <span className="dialog-message">Version</span>
        <input
          className="dialog-input"
          value={version}
          onChange={(e) => setVersion(e.target.value)}
          spellCheck={false}
        />
      </label>
      <label className="dialog-field">
        <span className="dialog-message">Target branch</span>
        <input className="dialog-input" value={target} onChange={(e) => setTarget(e.target.value)} spellCheck={false} />
      </label>
      <button type="submit" className="dialog-button dialog-button-primary">
        Check
      </button>
      <p className="release-readiness-hint">
        {result.previous ? `Against ${result.previous}` : 'No release before it'} · {result.sha.slice(0, 7)}
      </p>
    </form>
  )
}

// The files the version must appear in, kept for this repository.
function VersionFiles({ repoPath, onSaved }: { repoPath: string; onSaved: () => void }) {
  const { data: files } = useAsyncData(() => ChangelogService.VersionFiles(repoPath), [repoPath])
  const [text, setText] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(null)
  const shown = text ?? (files ?? []).join('\n')
  const save = () => {
    setError(null)
    ChangelogService.SetVersionFiles(repoPath, shown.split('\n'))
      .then(() => {
        setText(null)
        onSaved()
      })
      .catch((err: unknown) => setError(errorMessage(err)))
  }
  return (
    <details className="release-readiness-files">
      <summary>Version files</summary>
      <label className="dialog-field">
        <span className="dialog-message">One path per line, from the repository root</span>
        <textarea
          className="dialog-input release-readiness-files-input"
          value={shown}
          onChange={(e) => setText(e.target.value)}
          rows={5}
          spellCheck={false}
        />
      </label>
      {error && <p className="dialog-error">{error}</p>}
      <button type="button" className="dialog-button" onClick={save} disabled={text === null}>
        Save and check
      </button>
    </details>
  )
}

export default ReleaseReadinessDialog
