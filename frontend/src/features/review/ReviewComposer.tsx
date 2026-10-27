import { useState, type KeyboardEvent } from 'react'
import { errorMessage } from '../../lib/errors'
import './ReviewComposer.scss'

interface ReviewComposerProps {
  placeholder: string
  // Sends to the host now; the composer closes when it resolves.
  onSend: (body: string) => Promise<void>
  sendLabel?: string
  // Keeps the comment for a review submitted later; no request.
  onAddToReview?: (body: string) => void
  onCancel: () => void
}

// One text box with the two ways a comment leaves it: to the host now, or
// into the drafted review. Cmd/Ctrl+Enter sends.
function ReviewComposer({ placeholder, onSend, sendLabel = 'Comment', onAddToReview, onCancel }: ReviewComposerProps) {
  const [body, setBody] = useState('')
  const [sending, setSending] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const ready = body.trim() !== '' && !sending
  const send = () => {
    if (!ready) return
    setSending(true)
    setError(null)
    onSend(body.trim()).catch((err: unknown) => {
      setError(errorMessage(err))
      setSending(false)
    })
  }
  const onKeyDown = (e: KeyboardEvent<HTMLTextAreaElement>) => {
    if (e.key === 'Enter' && (e.metaKey || e.ctrlKey)) {
      e.preventDefault()
      send()
    } else if (e.key === 'Escape') {
      onCancel()
    }
  }
  return (
    <div className="review-composer">
      <textarea
        className="review-composer-text"
        value={body}
        onChange={(e) => setBody(e.target.value)}
        onKeyDown={onKeyDown}
        placeholder={placeholder}
        aria-label={placeholder}
        autoFocus
        rows={3}
      />
      {error && (
        <p className="review-composer-error" role="alert">
          {error}
        </p>
      )}
      <div className="review-composer-actions">
        <button type="button" className="review-composer-button" onClick={onCancel}>
          Cancel
        </button>
        {onAddToReview && (
          <button
            type="button"
            className="review-composer-button"
            disabled={!ready}
            onClick={() => onAddToReview(body.trim())}
          >
            Add to review
          </button>
        )}
        <button type="button" className="review-composer-button review-composer-send" disabled={!ready} onClick={send}>
          {sending ? 'Sending…' : sendLabel}
        </button>
      </div>
    </div>
  )
}

export default ReviewComposer
