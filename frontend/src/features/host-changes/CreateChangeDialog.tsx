import { useState, type FormEvent } from 'react'
import { CodeownersService, HostService, RemoteService, type ChangeDraft } from '@current-client-bindings/app'
import Modal from '../../components/chrome/Modal'
import Switch from '../../components/forms/Switch'
import { errorMessage } from '../../lib/errors'
import { withSignIn } from '../../lib/signIn'
import { useAsyncData } from '../../lib/useAsyncData'
import { useDialogs } from '../../lib/useDialogs'
import './CreateChangeDialog.scss'

interface CreateChangeDialogProps {
  repoPath: string
  branch: string
  // The host's word, for the title before the draft arrives.
  kind: string
  onCreated: (number: number) => void
  onClose: () => void
}

// The in-app form for a merge request / pull request from a branch,
// prefilled from its commits and the repository's template. The one write
// to the host; a token without write scope gets the message naming the
// scope.
function CreateChangeDialog({ repoPath, branch, kind, onCreated, onClose }: CreateChangeDialogProps) {
  const { data: draft, error: draftError } = useAsyncData(
    () => HostService.ChangeDraft(repoPath, branch),
    [repoPath, branch],
  )
  if (draftError) {
    return (
      <Modal title={`Create ${kind}`} onClose={onClose} className="dialog-panel">
        <div className="dialog-body">
          <p className="dialog-error" role="alert">
            {draftError}
          </p>
        </div>
      </Modal>
    )
  }
  if (!draft) return null
  return <CreateChangeForm repoPath={repoPath} draft={draft} onCreated={onCreated} onClose={onClose} />
}

interface CreateChangeFormProps {
  repoPath: string
  draft: ChangeDraft
  onCreated: (number: number) => void
  onClose: () => void
}

function CreateChangeForm({ repoPath, draft, onCreated, onClose }: CreateChangeFormProps) {
  const { signIn } = useDialogs()
  const [title, setTitle] = useState(draft.title)
  const [target, setTarget] = useState(draft.target)
  const [description, setDescription] = useState(draft.description)
  const [isDraft, setIsDraft] = useState(false)
  const [running, setRunning] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const kind = draft.kind
  const needsPush = !draft.hasUpstream
  // CODEOWNERS owners of what the branch changes against the target; local only.
  const { data: reviewers } = useAsyncData(
    () => CodeownersService.SuggestedReviewers(repoPath, draft.branch, target.trim()).catch(() => []),
    [repoPath, draft.branch, target],
    { keepData: true },
  )
  const canSubmit = title.trim() !== '' && target.trim() !== '' && !running

  // A branch without an upstream is pushed first, with Git's own sign-in prompt if the remote asks.
  const submit = (e: FormEvent) => {
    e.preventDefault()
    if (!canSubmit) return
    setRunning(true)
    setError(null)
    withSignIn(async (auth) => {
      if (needsPush) await RemoteService.PushSetUpstream(repoPath, draft.remote, draft.branch, auth)
      return HostService.CreateChange(repoPath, draft.branch, {
        target: target.trim(),
        title: title.trim(),
        description,
        draft: isDraft,
      })
    }, signIn)
      .then((change) => onCreated(change.number))
      .catch((err: unknown) => setError(errorMessage(err)))
      .finally(() => setRunning(false))
  }

  return (
    <Modal title={`Create ${kind}`} onClose={onClose} className="dialog-panel create-change-panel">
      <form className="dialog-body" onSubmit={submit}>
        <p className="dialog-message">
          From <span className="dialog-result">{draft.branch}</span>
          {needsPush ? ', pushed first' : ''}
        </p>
        <label className="dialog-field">
          <span className="dialog-message">Title</span>
          <input className="dialog-input" value={title} onChange={(e) => setTitle(e.target.value)} autoFocus />
        </label>
        <label className="dialog-field">
          <span className="dialog-message">Target branch</span>
          <input
            className="dialog-input create-change-target"
            value={target}
            onChange={(e) => setTarget(e.target.value)}
            spellCheck={false}
          />
        </label>
        {reviewers && reviewers.length > 0 && (
          <p className="dialog-message create-change-reviewers">
            Suggested reviewers <span className="dialog-result">{reviewers.join(' ')}</span>
            <span className="create-change-reviewers-hint">Code owners of the changed files</span>
          </p>
        )}
        <label className="dialog-field">
          <span className="dialog-message">Description</span>
          <textarea
            className="dialog-input create-change-description"
            value={description}
            onChange={(e) => setDescription(e.target.value)}
            rows={10}
          />
        </label>
        <Switch checked={isDraft} onChange={setIsDraft} label="Draft" />
        {error && (
          <p className="dialog-error" role="alert">
            {error}
          </p>
        )}
        <div className="dialog-actions">
          <button type="button" className="dialog-button" onClick={onClose}>
            Cancel
          </button>
          <button type="submit" className="dialog-button dialog-button-primary" disabled={!canSubmit}>
            {running ? 'Creating…' : needsPush ? `Push and create ${kind}` : `Create ${kind}`}
          </button>
        </div>
      </form>
    </Modal>
  )
}

export default CreateChangeDialog
