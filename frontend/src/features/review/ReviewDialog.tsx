import { useEffect, useState, type FormEvent } from 'react'
import { HostService } from '@current-client-bindings/app'
import Modal from '../../components/chrome/Modal'
import { useAsyncData } from '../../lib/useAsyncData'
import { reviewTarget, useReviewDraft } from '../../lib/reviewDraft'
import DialogError from '../../components/chrome/DialogError'
import { useAction } from '../../lib/useAction'
import DialogActions from '../../components/chrome/DialogActions'
import './ReviewDialog.scss'

interface ReviewDialogProps {
  repoPath: string
  number: number
  prefix: string
  onSubmitted: () => void
  onClose: () => void
}

type Verdict = 'approve' | 'request-changes' | 'comment'

const VERDICTS: { id: Verdict; label: string; hint: string }[] = [
  { id: 'approve', label: 'Approve', hint: 'Clears the change to merge.' },
  { id: 'request-changes', label: 'Request changes', hint: 'Something must change before it merges.' },
  { id: 'comment', label: 'Comment', hint: 'Notes without a verdict.' },
]

// A review submitted together: the verdict the host allows, an optional
// summary, and the comments drafted in the diff. One request on Submit
// (several on GitLab, which has no single review call), nothing before.
function ReviewDialog({ repoPath, number, prefix, onSubmitted, onClose }: ReviewDialogProps) {
  const target = reviewTarget(repoPath, number)
  const draft = useReviewDraft(target)
  const { data: capabilities, error: capabilitiesError } = useAsyncData(
    () => HostService.ReviewCapabilities(repoPath),
    [repoPath],
  )
  const allowed = VERDICTS.filter(
    (v) =>
      v.id === 'comment' ||
      (v.id === 'approve' && capabilities?.approve) ||
      (v.id === 'request-changes' && capabilities?.requestChanges),
  )
  const [verdict, setVerdict] = useState<Verdict | null>(null)
  const [summary, setSummary] = useState('')
  const { run, running, error } = useAction()
  useEffect(() => {
    if (capabilities) setVerdict((v) => v ?? (capabilities.approve ? 'approve' : 'comment'))
  }, [capabilities])
  const chosen = verdict ?? 'comment'
  const canSubmit =
    !!capabilities && !running && (chosen !== 'comment' || summary.trim() !== '' || draft.comments.length > 0)

  const submit = (e: FormEvent) => {
    e.preventDefault()
    if (!canSubmit) return
    const comments = draft.comments.map(({ path, line, oldLine, body }) => ({ path, line, oldLine, body }))
    void run(() =>
      HostService.SubmitReview(repoPath, number, { verdict: chosen, summary: summary.trim(), comments }).then(() => {
        draft.clear(target)
        draft.wrote()
        onSubmitted()
      }),
    )
  }

  return (
    <Modal title={`Review ${prefix}${number}`} onClose={onClose} className="dialog-panel review-dialog-panel">
      <form className="dialog-body" onSubmit={submit}>
        <DialogError>{capabilitiesError}</DialogError>
        {capabilities && (
          <fieldset className="review-dialog-verdicts">
            <legend className="dialog-message">Verdict</legend>
            {allowed.map((v) => (
              <label key={v.id} className="review-dialog-verdict">
                <input
                  type="radio"
                  name="verdict"
                  value={v.id}
                  checked={chosen === v.id}
                  onChange={() => setVerdict(v.id)}
                />
                <span>
                  {v.label}
                  <span className="review-dialog-hint">{v.hint}</span>
                </span>
              </label>
            ))}
            {!capabilities.requestChanges && (
              <p className="dialog-note">The host's API has no request-changes verdict; a comment says it instead.</p>
            )}
          </fieldset>
        )}
        <label className="dialog-field">
          <span className="dialog-message">Summary</span>
          <textarea
            className="dialog-input review-dialog-summary"
            value={summary}
            onChange={(e) => setSummary(e.target.value)}
            placeholder="Optional"
          />
        </label>
        <p className="dialog-note">
          {draft.comments.length === 0
            ? 'No drafted comments. Comments drafted in the diff are sent with the review.'
            : `${draft.comments.length} drafted ${draft.comments.length === 1 ? 'comment is' : 'comments are'} sent with the review.`}
        </p>
        <DialogError>{error}</DialogError>
        <DialogActions onCancel={onClose} disabled={!canSubmit}>
          {running ? 'Submitting…' : 'Submit review'}
        </DialogActions>
      </form>
    </Modal>
  )
}

export default ReviewDialog
