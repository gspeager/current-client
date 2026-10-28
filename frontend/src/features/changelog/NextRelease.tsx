import { useEffect, useState, type FormEvent } from 'react'
import {
  ArrowRight,
  CircleCheck,
  CircleHelp,
  Info,
  PackagePlus,
  Pencil,
  RefreshCw,
  Tag,
  TriangleAlert,
  type LucideIcon,
} from 'lucide-react'
import { ChangelogService, HostService, TagService, type ReleaseCheck } from '@current-client-bindings/app'
import PublishReleaseDialog from '../host-changes/PublishReleaseDialog'
import { errorMessage } from '../../lib/errors'
import { relativeTime } from '../../lib/relativeTime'
import { useAsyncData } from '../../lib/useAsyncData'
import { useClockTick } from '../../lib/useClockTick'
import { useDialogs } from '../../lib/useDialogs'
import './NextRelease.scss'

interface NextReleaseProps {
  repoPath: string
  // The checks are read again whenever this changes, so they never go stale.
  repoVersion: number
  onTagged?: () => void
}

const LEVELS: Record<string, { icon: LucideIcon; label: string }> = {
  ok: { icon: CircleCheck, label: 'Ready' },
  warn: { icon: TriangleAlert, label: 'Needs attention' },
  info: { icon: Info, label: 'For information' },
  unknown: { icon: CircleHelp, label: 'Not known' },
}

// A version file's label is its path; the rest are words.
const isPath = (label: string) => !label.includes(' ') && /[./]/.test(label)

// The release being finished: what it would be, what still stands in its
// way, and the steps to make it, in order. Git answers all but CI and
// protection, which come from the host's last refresh. Nothing is tagged or
// published until a step's button is clicked.
function NextRelease({ repoPath, repoVersion, onTagged }: NextReleaseProps) {
  useClockTick()
  const { confirm } = useDialogs()
  const [asked, setAsked] = useState({ ref: '', version: '', target: '', run: 0 })
  const { data, error, reload } = useAsyncData(
    () => ChangelogService.ReleaseReadiness(repoPath, asked.ref, asked.version, asked.target),
    [repoPath, asked],
    { keepData: true, refreshKey: repoVersion },
  )
  const { data: canPublish } = useAsyncData(
    () => HostService.CanPublishReleases(repoPath).catch(() => false),
    [repoPath],
  )
  const [checkedAt, setCheckedAt] = useState<Date | null>(null)
  useEffect(() => setCheckedAt(data ? new Date() : null), [data])
  const [editing, setEditing] = useState(false)
  const [publishing, setPublishing] = useState(false)
  const [actionError, setActionError] = useState<string | null>(null)

  if (!data) {
    return (
      <section className="next-release" aria-label="Next release">
        <p className="next-release-hint">{error ? `Could not check the release: ${error}` : 'Checking…'}</p>
      </section>
    )
  }

  const tagged = data.taggedAs !== ''
  const tag = async () => {
    const message = `Release ${data.version}`
    const confirmed = await confirm({
      title: `Tag ${data.tag}`,
      message: `Tag ${data.sha.slice(0, 7)} as ${data.tag}, annotated "${message}". The tag stays local until Publish release pushes it.`,
      confirmLabel: 'Tag',
    })
    if (!confirmed) return
    setActionError(null)
    TagService.CreateTag(repoPath, data.tag, message, data.sha)
      .then(() => {
        reload()
        onTagged?.()
      })
      .catch((err: unknown) => setActionError(errorMessage(err)))
  }
  const publishHint = !canPublish
    ? 'Needs the host integration on, with write scope'
    : !tagged
      ? 'Tag first'
      : undefined

  return (
    <section className="next-release" aria-label="Next release">
      <header className="next-release-header">
        <h2 className="next-release-title">Next release</h2>
        <span className="next-release-version">{data.tag || 'No version'}</span>
        <span className="next-release-meta">
          {data.previous ? `from ${data.previous}` : 'first release'} · candidate {data.ref}{' '}
          <span className="next-release-sha">{data.sha.slice(0, 7)}</span> · target {data.target}
          {tagged && ` · tagged ${data.taggedAs}`}
        </span>
        <span className="next-release-spacer" />
        {checkedAt && <span className="next-release-checked">checked {relativeTime(checkedAt)}</span>}
        <button
          type="button"
          className="next-release-button"
          aria-expanded={editing}
          onClick={() => setEditing((on) => !on)}
        >
          <Pencil size={14} strokeWidth={1.75} />
          Edit
        </button>
        <button type="button" className="next-release-button" onClick={reload}>
          <RefreshCw size={14} strokeWidth={1.75} />
          Re-check
        </button>
      </header>
      {editing && (
        <div className="next-release-edit">
          <ReleaseForm
            key={`${data.sha}-${asked.run}`}
            version={data.version}
            candidate={data.ref}
            target={data.target}
            onCheck={(next) => setAsked({ ...next, run: asked.run + 1 })}
          />
          <VersionFiles repoPath={repoPath} onSaved={reload} />
        </div>
      )}
      {error && <p className="next-release-error">{error}</p>}
      <ul className="next-release-checks" aria-label="Checks">
        {data.checks.map((check) => (
          <CheckRow key={check.label} check={check} />
        ))}
      </ul>
      <footer className="next-release-steps">
        <button
          type="button"
          className={tagged ? 'next-release-step' : 'next-release-step next-release-step-primary'}
          onClick={() => void tag()}
          disabled={!data.tag || tagged}
          title={tagged ? `Already tagged ${data.taggedAs}` : undefined}
        >
          <Tag size={14} strokeWidth={1.75} />
          Tag {data.tag}…
        </button>
        <ArrowRight size={14} strokeWidth={1.75} className="next-release-arrow" aria-hidden="true" />
        <button
          type="button"
          className={tagged && canPublish ? 'next-release-step next-release-step-primary' : 'next-release-step'}
          onClick={() => setPublishing(true)}
          disabled={!tagged || !canPublish}
          title={publishHint}
        >
          <PackagePlus size={14} strokeWidth={1.75} />
          Publish release…
        </button>
        {actionError && <span className="next-release-error">{actionError}</span>}
      </footer>
      {publishing && (
        <PublishReleaseDialog repoPath={repoPath} tag={data.taggedAs} onClose={() => setPublishing(false)} />
      )}
    </section>
  )
}

// One check: an icon, what it is, what it found. What it rests on is shown
// when it needs attention, and in the tooltip otherwise.
function CheckRow({ check }: { check: ReleaseCheck }) {
  const level = LEVELS[check.level] ?? LEVELS.unknown
  const Icon = level.icon
  const evidence = check.evidence ?? []
  return (
    <li
      className={`release-check release-check-${check.level}`}
      title={check.level !== 'warn' && evidence.length > 0 ? evidence.join('\n') : undefined}
    >
      <Icon size={14} strokeWidth={1.75} aria-label={level.label} role="img" />
      <span className={isPath(check.label) ? 'release-check-label release-check-path' : 'release-check-label'}>
        {check.label}
      </span>
      <span className="release-check-value">{check.value}</span>
      {check.asOf && <span className="release-check-asof">{relativeTime(new Date(check.asOf))}</span>}
      {check.level === 'warn' && evidence.length > 0 && (
        <ul className="release-check-evidence">
          {evidence.map((line, i) => (
            <li key={i}>{line}</li>
          ))}
        </ul>
      )}
    </li>
  )
}

function ReleaseForm({
  version: initialVersion,
  candidate,
  target: initialTarget,
  onCheck,
}: {
  version: string
  candidate: string
  target: string
  onCheck: (asked: { ref: string; version: string; target: string }) => void
}) {
  const [version, setVersion] = useState(initialVersion)
  const [ref, setRef] = useState(candidate)
  const [target, setTarget] = useState(initialTarget)
  const submit = (e: FormEvent) => {
    e.preventDefault()
    onCheck({ ref: ref.trim(), version: version.trim(), target: target.trim() })
  }
  return (
    <form className="next-release-form" onSubmit={submit}>
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
        <span className="dialog-message">Candidate</span>
        <input className="dialog-input" value={ref} onChange={(e) => setRef(e.target.value)} spellCheck={false} />
      </label>
      <label className="dialog-field">
        <span className="dialog-message">Target branch</span>
        <input className="dialog-input" value={target} onChange={(e) => setTarget(e.target.value)} spellCheck={false} />
      </label>
      <button type="submit" className="dialog-button">
        Check
      </button>
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
    <div className="next-release-files">
      <label className="dialog-field">
        <span className="dialog-message">Version files, one path per line from the repository root</span>
        <textarea
          className="dialog-input next-release-files-input"
          value={shown}
          onChange={(e) => setText(e.target.value)}
          rows={3}
          spellCheck={false}
        />
      </label>
      {error && <p className="dialog-error">{error}</p>}
      <button type="button" className="dialog-button" onClick={save} disabled={text === null}>
        Save and check
      </button>
    </div>
  )
}

export default NextRelease
