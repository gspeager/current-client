import { useState, type FormEvent } from 'react'
import { HostService } from '@current-client-bindings/app'
import type { Issue } from '@current-client-bindings/core/hostapi'
import Modal from '../../components/chrome/Modal'
import { errorMessage } from '../../lib/errors'
import type { IssueDraft } from '../../lib/issuePanel'
import './CreateChangeDialog.scss'

interface NewIssueDialogProps {
  repoPath: string
  draft?: IssueDraft
  onCreated: (issue: Issue) => void
  onClose: () => void
}

function splitLabels(text: string): string[] {
  return text
    .split(',')
    .map((label) => label.trim())
    .filter(Boolean)
}

// Nothing is sent until Create issue; a token without write scope gets the
// message naming the scope.
function NewIssueDialog({ repoPath, draft, onCreated, onClose }: NewIssueDialogProps) {
  const [title, setTitle] = useState(draft?.title ?? '')
  const [description, setDescription] = useState(draft?.description ?? '')
  const [labels, setLabels] = useState('')
  const [running, setRunning] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const canSubmit = title.trim() !== '' && !running

  const submit = (e: FormEvent) => {
    e.preventDefault()
    if (!canSubmit) return
    setRunning(true)
    setError(null)
    HostService.CreateIssue(repoPath, { title: title.trim(), description, labels: splitLabels(labels) })
      .then((issue) => onCreated(issue))
      .catch((err: unknown) => setError(errorMessage(err)))
      .finally(() => setRunning(false))
  }

  return (
    <Modal title="New issue" onClose={onClose} className="dialog-panel create-change-panel">
      <form className="dialog-body" onSubmit={submit}>
        <label className="dialog-field">
          <span className="dialog-message">Title</span>
          <input className="dialog-input" value={title} onChange={(e) => setTitle(e.target.value)} autoFocus />
        </label>
        <label className="dialog-field">
          <span className="dialog-message">Description</span>
          <textarea
            className="dialog-input create-change-description"
            value={description}
            onChange={(e) => setDescription(e.target.value)}
            rows={8}
          />
        </label>
        <label className="dialog-field">
          <span className="dialog-message">Labels, separated by commas</span>
          <input
            className="dialog-input"
            value={labels}
            onChange={(e) => setLabels(e.target.value)}
            spellCheck={false}
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
          <button type="submit" className="dialog-button dialog-button-primary" disabled={!canSubmit}>
            {running ? 'Creating…' : 'Create issue'}
          </button>
        </div>
      </form>
    </Modal>
  )
}

export default NewIssueDialog
