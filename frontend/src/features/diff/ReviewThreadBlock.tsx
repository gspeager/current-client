import { useState } from 'react'
import { ChevronRight, MessageSquare } from 'lucide-react'
import type { ReviewThread } from '@current-client-bindings/app'
import HostMarkdown from '../../components/git/HostMarkdown'
import ReviewComposer from '../review/ReviewComposer'
import type { ThreadActions } from '../review/useReviewActions'
import { errorMessage } from '../../lib/errors'
import { plural } from '../../lib/plural'
import { relativeTime } from '../../lib/relativeTime'
import './ReviewThreadBlock.scss'

interface ReviewThreadBlockProps {
  thread: ReviewThread
  // Shown for outdated threads, which are listed away from the code.
  where?: string
  // With these, the thread takes a reply and, where the host's API allows,
  // resolves; without, it is read-only.
  actions?: ThreadActions
}

// One review thread: resolved ones start collapsed to a line. Note bodies
// come rendered from the Go side; nothing here fetches, and nothing is sent
// until Reply or Resolve is clicked.
function ReviewThreadBlock({ thread, where, actions }: ReviewThreadBlockProps) {
  const [open, setOpen] = useState(!thread.resolved)
  const [replying, setReplying] = useState(false)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const resolve = () => {
    if (!actions) return
    setBusy(true)
    setError(null)
    actions
      .resolve(thread.id, !thread.resolved)
      .catch((err: unknown) => setError(errorMessage(err)))
      .finally(() => setBusy(false))
  }
  const first = thread.notes?.[0]
  const title = `${thread.resolved ? 'Resolved' : 'Unresolved'} thread${first ? ` by ${first.author}` : ''}${where ? ` at ${where}` : ''}`
  return (
    <div
      className={thread.resolved ? 'review-thread review-thread-resolved' : 'review-thread'}
      role="group"
      aria-label={title}
    >
      <button type="button" className="review-thread-head" onClick={() => setOpen((o) => !o)} aria-expanded={open}>
        <ChevronRight
          size={12}
          strokeWidth={1.75}
          className={open ? 'review-thread-chevron review-thread-chevron-open' : 'review-thread-chevron'}
        />
        <MessageSquare size={12} strokeWidth={1.75} />
        <span className="review-thread-summary">
          {thread.resolved ? 'Resolved' : plural(thread.notes?.length ?? 0, 'note')}
          {first ? ` · ${first.author}` : ''}
          {where ? ` · ${where}` : ''}
        </span>
      </button>
      {open && (
        <ol className="review-thread-notes">
          {thread.notes?.map((note, i) => (
            <li key={i} className="review-note">
              <span className="review-note-meta">
                <span className="review-note-author">{note.author}</span>
                <span className="review-note-time">{relativeTime(new Date(note.createdAt))}</span>
              </span>
              <HostMarkdown html={note.bodyHtml} className="review-note-body" />
            </li>
          ))}
        </ol>
      )}
      {open && actions && !replying && (
        <div className="review-thread-actions">
          <button type="button" className="review-thread-action" onClick={() => setReplying(true)}>
            Reply
          </button>
          {actions.canResolve && (
            <button type="button" className="review-thread-action" onClick={resolve} disabled={busy}>
              {thread.resolved ? 'Reopen' : 'Resolve'}
            </button>
          )}
        </div>
      )}
      {open && actions && replying && (
        <ReviewComposer
          placeholder="Reply"
          sendLabel="Reply"
          onSend={(body) => actions.reply(thread.id, body).then(() => setReplying(false))}
          onCancel={() => setReplying(false)}
        />
      )}
      {error && (
        <p className="review-thread-error" role="alert">
          {error}
        </p>
      )}
    </div>
  )
}

export default ReviewThreadBlock
