import { PencilLine, X } from 'lucide-react'
import type { PendingComment } from '../../lib/reviewDraft'
import './PendingCommentBlock.scss'

interface PendingCommentBlockProps {
  comment: PendingComment
  onRemove: () => void
}

// A comment drafted for the review, shown where it will land; nothing has
// reached the host yet.
function PendingCommentBlock({ comment, onRemove }: PendingCommentBlockProps) {
  const where = comment.line ? `line ${comment.line}` : `old line ${comment.oldLine}`
  return (
    <div className="pending-comment" role="group" aria-label={`Drafted comment at ${where}`}>
      <span className="pending-comment-head">
        <PencilLine size={12} strokeWidth={1.75} />
        Drafted for the review · {where}
        <button type="button" className="pending-comment-remove" onClick={onRemove} aria-label="Remove drafted comment">
          <X size={12} strokeWidth={1.75} />
        </button>
      </span>
      <p className="pending-comment-body">{comment.body}</p>
    </div>
  )
}

export default PendingCommentBlock
