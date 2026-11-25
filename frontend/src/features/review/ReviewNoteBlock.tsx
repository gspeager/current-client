import { useState } from 'react'
import { Pencil, StickyNote, X } from 'lucide-react'
import type { ReadThroughNote } from '@current-client-bindings/app'
import { errorMessage } from '../../lib/errors'
import ReviewComposer from './ReviewComposer'
import './PendingCommentBlock.scss'

// Notes on one file, kept on this computer.
export interface LineNotes {
  items: ReadThroughNote[]
  add: (line: number, oldLine: number, text: string) => Promise<void>
  // The old side isn't the base, so its lines take none.
  newOnly?: boolean
  // Empty text removes it.
  edit: (id: string, text: string) => Promise<void>
}

export function noteWhere({ line, oldLine }: { line: number; oldLine: number }): string {
  return line ? `line ${line}` : oldLine ? `old line ${oldLine}` : 'whole file'
}

interface ReviewNoteBlockProps {
  note: ReadThroughNote
  onEdit: LineNotes['edit']
}

function ReviewNoteBlock({ note, onEdit }: ReviewNoteBlockProps) {
  const [editing, setEditing] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const where = noteWhere(note)
  return (
    <div className="pending-comment review-note" role="group" aria-label={`Note on ${where}`}>
      <span className="pending-comment-head">
        <StickyNote size={12} strokeWidth={1.75} />
        Note · {where}
        {note.outdated && <span className="review-note-outdated">Outdated</span>}
        <button
          type="button"
          className="pending-comment-remove"
          onClick={() => setEditing(true)}
          aria-label="Edit note"
          title="Edit note"
        >
          <Pencil size={12} strokeWidth={1.75} />
        </button>
        <button
          type="button"
          className="pending-comment-remove review-note-remove"
          onClick={() => onEdit(note.id, '').catch((err: unknown) => setError(errorMessage(err)))}
          aria-label="Remove note"
          title="Remove note"
        >
          <X size={12} strokeWidth={1.75} />
        </button>
      </span>
      {note.outdated && <p className="review-note-line">{note.lineText}</p>}
      {editing ? (
        <ReviewComposer
          placeholder={`Note on ${where}`}
          initial={note.text}
          sendLabel="Save"
          onSend={(text) => onEdit(note.id, text).then(() => setEditing(false))}
          onCancel={() => setEditing(false)}
        />
      ) : (
        <p className="pending-comment-body">{note.text}</p>
      )}
      {error && <p className="review-composer-error">{error}</p>}
    </div>
  )
}

export default ReviewNoteBlock
