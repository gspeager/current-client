import { useState, type FormEvent } from 'react'
import { HostService, PlatformService, TagService, type ReleaseDraft } from '@current-client-bindings/app'
import type { PublishedRelease } from '@current-client-bindings/core/hostapi'
import Modal from '../../components/chrome/Modal'
import { errorMessage, ignoreDecorativeFailure } from '../../lib/errors'
import { withSignIn } from '../../lib/signIn'
import { useAsyncData } from '../../lib/useAsyncData'
import { useDialogs } from '../../lib/useDialogs'
import './CreateChangeDialog.scss'
import { useHostLogo } from '../../components/git/hostLogos'

interface PublishReleaseDialogProps {
  repoPath: string
  tag: string
  onClose: () => void
}

// A release on the host for a tag, its notes prefilled from the changelog at
// the tag or its commits. The tag is pushed first; nothing reaches the host
// before Publish.
function PublishReleaseDialog({ repoPath, tag, onClose }: PublishReleaseDialogProps) {
  const { data: draft, error } = useAsyncData(() => HostService.ReleaseDraft(repoPath, tag), [repoPath, tag])
  if (error) {
    return (
      <Modal title="Publish release" onClose={onClose} className="dialog-panel">
        <div className="dialog-body">
          <p className="dialog-error" role="alert">
            {error}
          </p>
        </div>
      </Modal>
    )
  }
  if (!draft) return null
  return <PublishReleaseForm repoPath={repoPath} draft={draft} onClose={onClose} />
}

function notesSource(draft: ReleaseDraft): string {
  if (draft.notesFrom === 'changelog') return `From the changelog at ${draft.tag}`
  return draft.since ? `From the commits since ${draft.since}` : 'From every commit up to the tag'
}

function PublishReleaseForm({
  repoPath,
  draft,
  onClose,
}: {
  repoPath: string
  draft: ReleaseDraft
  onClose: () => void
}) {
  const { signIn } = useDialogs()
  const [name, setName] = useState(draft.name)
  const [notes, setNotes] = useState(draft.notes)
  const [running, setRunning] = useState(false)
  const HostIcon = useHostLogo(repoPath)
  const [error, setError] = useState<string | null>(null)
  const [published, setPublished] = useState<PublishedRelease | null>(null)

  const submit = (e: FormEvent) => {
    e.preventDefault()
    if (running) return
    setRunning(true)
    setError(null)
    withSignIn(async (auth) => {
      await TagService.PushTag(repoPath, draft.remote, draft.tag, auth)
      return HostService.PublishRelease(repoPath, { tag: draft.tag, name, notes })
    }, signIn)
      .then(setPublished)
      .catch((err: unknown) => setError(errorMessage(err)))
      .finally(() => setRunning(false))
  }

  if (published) {
    return (
      <Modal title="Publish release" onClose={onClose} className="dialog-panel">
        <div className="dialog-body">
          <p className="dialog-message" role="status">
            Published <span className="dialog-result">{published.name || published.tag}</span>.
          </p>
          <div className="dialog-actions">
            <button
              type="button"
              className="dialog-button"
              onClick={() => PlatformService.OpenWebPage(published.webUrl).catch(ignoreDecorativeFailure)}
              disabled={!published.webUrl}
            >
              <HostIcon size={14} strokeWidth={1.75} />
              Open on host
            </button>
            <button type="button" className="dialog-button dialog-button-primary" onClick={onClose}>
              Close
            </button>
          </div>
        </div>
      </Modal>
    )
  }

  return (
    <Modal title="Publish release" onClose={onClose} className="dialog-panel create-change-panel">
      <form className="dialog-body" onSubmit={submit}>
        <p className="dialog-message">
          For <span className="dialog-result">{draft.tag}</span>, pushed to {draft.remote} first
        </p>
        <label className="dialog-field">
          <span className="dialog-message">Name</span>
          <input className="dialog-input" value={name} onChange={(e) => setName(e.target.value)} autoFocus />
        </label>
        <label className="dialog-field">
          <span className="dialog-message">Notes · {notesSource(draft)}</span>
          <textarea
            className="dialog-input create-change-description"
            value={notes}
            onChange={(e) => setNotes(e.target.value)}
            rows={12}
          />
        </label>
        {error && (
          <p className="dialog-error" role="alert">
            {error}
          </p>
        )}
        <div className="dialog-actions">
          <button type="button" className="dialog-button" onClick={onClose}>
            Cancel
          </button>
          <button type="submit" className="dialog-button dialog-button-primary" disabled={running}>
            {running ? 'Publishing…' : 'Publish release'}
          </button>
        </div>
      </form>
    </Modal>
  )
}

export default PublishReleaseDialog
