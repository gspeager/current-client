import { useState, type FormEvent } from 'react'
import { HostService, TagService, type ReleaseDraft } from '@current-client-bindings/app'
import type { PublishedRelease } from '@current-client-bindings/core/hostapi'
import Modal from '../../components/chrome/Modal'
import { openWebPage } from '../../lib/hostPages'
import { withSignIn } from '../../lib/signIn'
import { useAsyncData } from '../../lib/useAsyncData'
import { useDialogs } from '../../lib/useDialogs'
import { useHostLogo } from '../../components/git/hostLogos'
import { DraftErrorModal } from './CreateChangeDialog'
import DialogError from '../../components/chrome/DialogError'
import { useAction } from '../../lib/useAction'
import DialogActions from '../../components/chrome/DialogActions'
import '../../components/chrome/dialog.scss'
import './CreateChangeDialog.scss'

interface PublishReleaseDialogProps {
  repoPath: string
  tag: string
  onClose: () => void
}

// The tag is pushed first; nothing reaches the host before Publish.
function PublishReleaseDialog({ repoPath, tag, onClose }: PublishReleaseDialogProps) {
  const { data: draft, error } = useAsyncData(() => HostService.ReleaseDraft(repoPath, tag), [repoPath, tag])
  if (error) return <DraftErrorModal title="Publish release" error={error} onClose={onClose} />
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
  const HostIcon = useHostLogo(repoPath)
  const { run, running, error } = useAction()
  const [published, setPublished] = useState<PublishedRelease | null>(null)

  const submit = (e: FormEvent) => {
    e.preventDefault()
    if (running) return
    void run(() =>
      withSignIn(async (auth) => {
        await TagService.PushTag(repoPath, draft.remote, draft.tag, auth)
        return HostService.PublishRelease(repoPath, { tag: draft.tag, name, notes })
      }, signIn).then(setPublished),
    )
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
              onClick={() => openWebPage(published.webUrl)}
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
        <DialogError>{error}</DialogError>
        <DialogActions onCancel={onClose} disabled={running}>
          {running ? 'Publishing…' : 'Publish release'}
        </DialogActions>
      </form>
    </Modal>
  )
}

export default PublishReleaseDialog
