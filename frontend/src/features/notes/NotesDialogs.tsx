import { useState } from 'react'
import { NotesService, type NoteInfo } from '@current-client-bindings/app'
import Modal from '../../components/chrome/Modal'
import { errorMessage } from '../../lib/errors'
import type { NoteKind, NoteTarget, useNotesState } from './notes'
import { shortSha } from '../../lib/text'
import DialogError from '../../components/chrome/DialogError'

export function noteLabel(note: { kind: string; target: string; subject?: string }): string {
  return note.kind === 'commit' ? `${shortSha(note.target)} ${note.subject ?? ''}`.trim() : note.target
}

const KIND_LABEL: Record<NoteKind, string> = { commit: 'Commit', branch: 'Branch', tag: 'Tag', file: 'File' }

interface NoteEditorProps {
  repoPath: string
  target: NoteTarget
  existing?: NoteInfo
  onSaved: () => void
  onClose: () => void
}

function NoteEditor({ repoPath, target, existing, onSaved, onClose }: NoteEditorProps) {
  const [text, setText] = useState(existing?.text ?? '')
  const [error, setError] = useState<string | null>(null)

  const save = (value: string) => {
    setError(null)
    NotesService.Set(repoPath, target.kind, target.target, target.subject ?? '', value)
      .then(() => {
        onSaved()
        onClose()
      })
      .catch((err: unknown) => setError(errorMessage(err)))
  }

  return (
    <Modal title={`Note on ${noteLabel(target)}`} onClose={onClose} className="dialog-panel notes-panel">
      <div className="dialog-body">
        <textarea
          className="note-editor"
          aria-label="Note"
          autoFocus
          value={text}
          onChange={(e) => setText(e.target.value)}
        />
        <p className="dialog-note">Plain text, kept on this computer. Never pushed or shared.</p>
        <DialogError>{error}</DialogError>
        <div className="dialog-actions">
          {existing && (
            <button type="button" className="dialog-button dialog-button-destructive" onClick={() => save('')}>
              Remove
            </button>
          )}
          <button type="button" className="dialog-button" onClick={onClose}>
            Cancel
          </button>
          <button
            type="button"
            className="dialog-button dialog-button-primary"
            disabled={text.trim() === ''}
            onClick={() => save(text)}
          >
            Save
          </button>
        </div>
      </div>
    </Modal>
  )
}

interface NotesListProps {
  notes: NoteInfo[]
  onEdit: (note: NoteInfo) => void
  onOpen: (note: NoteInfo) => void
  onClose: () => void
}

function NotesList({ notes, onEdit, onOpen, onClose }: NotesListProps) {
  return (
    <Modal title="Notes" onClose={onClose} className="dialog-panel notes-panel">
      <div className="dialog-body">
        {notes.length === 0 ? (
          <p className="dialog-message">No notes in this repository.</p>
        ) : (
          <ul className="notes-list">
            {notes.map((note) => (
              <li key={`${note.kind}-${note.target}`} className="notes-row">
                <div className="notes-row-main">
                  <div className="notes-row-where">
                    <span>{KIND_LABEL[note.kind as NoteKind]}</span>
                    <span className="notes-row-target" title={note.target}>
                      {noteLabel(note)}
                    </span>
                    {note.stale && <span className="notes-row-stale">Gone</span>}
                  </div>
                  <p className="notes-row-text">{note.text}</p>
                </div>
                <div className="notes-row-actions">
                  {!note.stale && (
                    <button type="button" className="dialog-button" onClick={() => onOpen(note)}>
                      Open
                    </button>
                  )}
                  <button type="button" className="dialog-button" onClick={() => onEdit(note)}>
                    Edit
                  </button>
                </div>
              </li>
            ))}
          </ul>
        )}
        <div className="dialog-actions">
          <button type="button" className="dialog-button" onClick={onClose}>
            Close
          </button>
        </div>
      </div>
    </Modal>
  )
}

interface NotesDialogsProps {
  repoPath: string
  state: ReturnType<typeof useNotesState>
  onOpen: (note: NoteInfo) => void
}

function NotesDialogs({ repoPath, state, onOpen }: NotesDialogsProps) {
  const { context, editing, setEditing, listOpen, setListOpen, reload } = state
  return (
    <>
      {listOpen && !editing && (
        <NotesList
          notes={context.notes}
          onEdit={(note) => setEditing({ kind: note.kind as NoteKind, target: note.target, subject: note.subject })}
          onOpen={(note) => {
            setListOpen(false)
            onOpen(note)
          }}
          onClose={() => setListOpen(false)}
        />
      )}
      {editing && (
        <NoteEditor
          key={`${editing.kind}-${editing.target}`}
          repoPath={repoPath}
          target={editing}
          existing={context.noteFor(editing.kind, editing.target)}
          onSaved={reload}
          onClose={() => setEditing(null)}
        />
      )}
    </>
  )
}

export default NotesDialogs
