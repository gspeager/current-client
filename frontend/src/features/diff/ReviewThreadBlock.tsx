import { useState } from 'react'
import { ChevronRight, MessageSquare } from 'lucide-react'
import type { ReviewThread } from '@current-client-bindings/app'
import HostMarkdown from '../../components/git/HostMarkdown'
import { relativeTime } from '../../lib/relativeTime'
import './ReviewThreadBlock.scss'

interface ReviewThreadBlockProps {
  thread: ReviewThread
  // Shown for outdated threads, which are listed away from the code.
  where?: string
}

// One review thread, read-only: resolved ones start collapsed to a line.
// Note bodies come rendered from the Go side; nothing here fetches.
function ReviewThreadBlock({ thread, where }: ReviewThreadBlockProps) {
  const [open, setOpen] = useState(!thread.resolved)
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
          {thread.resolved
            ? 'Resolved'
            : `${thread.notes?.length ?? 0} ${thread.notes?.length === 1 ? 'note' : 'notes'}`}
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
    </div>
  )
}

export default ReviewThreadBlock
