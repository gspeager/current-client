import { createContext, useContext, useMemo, useState, type MouseEvent } from 'react'
import { StickyNote } from 'lucide-react'
import { NotesService, type NoteInfo } from '@current-client-bindings/app'
import { useAsyncData } from '../../lib/useAsyncData'
import './notes.scss'

export type NoteKind = 'commit' | 'branch' | 'tag' | 'file'

export interface NoteTarget {
  kind: NoteKind
  target: string
  subject?: string
}

export interface Notes {
  notes: NoteInfo[]
  noteFor: (kind: NoteKind, target: string) => NoteInfo | undefined
  edit: (target: NoteTarget) => void
  openList: () => void
}

const NO_NOTES: Notes = { notes: [], noteFor: () => undefined, edit: () => undefined, openList: () => undefined }

export const NotesContext = createContext<Notes>(NO_NOTES)

export function useNotes(): Notes {
  return useContext(NotesContext)
}

// The state behind NotesContext and NotesDialogs, for the app to hold.
export function useNotesState(repoPath: string | null, refreshKey: unknown) {
  const { data, reload } = useAsyncData(() => (repoPath ? NotesService.List(repoPath) : null), [repoPath], {
    refreshKey,
  })
  const [editing, setEditing] = useState<NoteTarget | null>(null)
  const [listOpen, setListOpen] = useState(false)

  const context = useMemo<Notes>(() => {
    const notes = repoPath ? (data ?? []) : []
    const byKey = new Map(notes.map((note) => [`${note.kind}\0${note.target}`, note]))
    return {
      notes,
      noteFor: (kind, target) => byKey.get(`${kind}\0${target}`),
      edit: setEditing,
      openList: () => setListOpen(true),
    }
  }, [data, repoPath])

  return { context, reload, editing, setEditing, listOpen, setListOpen }
}

export function noteMenuLabel(notes: Notes, kind: NoteKind, target: string): string {
  return notes.noteFor(kind, target) ? 'Edit note…' : 'Add note…'
}

interface NoteMarkProps {
  kind: NoteKind
  target: string
}

// Beside the thing the note is on; its tooltip is the note.
export function NoteMark({ kind, target }: NoteMarkProps) {
  const notes = useNotes()
  const note = notes.noteFor(kind, target)
  if (!note) return null
  const open = (e: MouseEvent) => {
    e.stopPropagation()
    notes.edit({ kind, target, subject: note.subject })
  }
  return (
    <button type="button" className="note-mark" title={note.text} aria-label={`Note: ${note.text}`} onClick={open}>
      <StickyNote size={12} strokeWidth={1.75} />
    </button>
  )
}
