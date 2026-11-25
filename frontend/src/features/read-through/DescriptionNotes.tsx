import { useState } from 'react'
import { ReadThroughInfo, ReadThroughRequest, ReadThroughService } from '@current-client-bindings/app'
import Checkbox from '../../components/forms/Checkbox'
import { useAsyncData } from '../../lib/useAsyncData'
import { toggled } from '../../lib/sets'
import { useNotes } from '../notes/notes'
import { noteWhere } from '../review/ReviewNoteBlock'
import { notesText } from './notesText'
import './DescriptionNotes.scss'

interface DescriptionNotesProps {
  repoPath: string
  branch: string
  onAdd: (text: string) => void
}

// A branch's notes, to choose from for a description. They're private, so none
// is chosen and none is added until the button is clicked.
function DescriptionNotes({ repoPath, branch, onAdd }: DescriptionNotesProps) {
  const ref = `refs/heads/${branch}`
  const hasNotes = useNotes().notes.some((note) => note.kind === 'review' && note.target === ref)
  const { data: notes } = useAsyncData(
    () =>
      hasNotes
        ? ReadThroughService.Open(repoPath, new ReadThroughRequest({ ref })).then((info) =>
            ReadThroughService.Notes(new ReadThroughInfo({ ...info, files: [] })),
          )
        : null,
    [repoPath, ref, hasNotes],
  )
  const [chosen, setChosen] = useState<Set<string>>(new Set())
  const [added, setAdded] = useState<Set<string>>(new Set())
  const left = (notes ?? []).filter((note) => !added.has(note.id))
  if (left.length === 0) return null

  const add = () => {
    onAdd(notesText(left.filter((note) => chosen.has(note.id))))
    setAdded(new Set([...added, ...chosen]))
    setChosen(new Set())
  }

  return (
    <fieldset className="description-notes">
      <legend className="dialog-message">Review notes, only on this computer</legend>
      <ul className="description-notes-list">
        {left.map((note) => (
          <li key={note.id}>
            <Checkbox
              checked={chosen.has(note.id)}
              onChange={(checked) => setChosen((prev) => toggled(prev, note.id, checked))}
              ariaLabel={`${note.path}, ${noteWhere(note)}: ${note.text}`}
            />
            <span className="description-notes-where">
              {note.path} · {noteWhere(note)}
            </span>
            <span className="description-notes-text">{note.text}</span>
          </li>
        ))}
      </ul>
      <button type="button" className="dialog-button" disabled={chosen.size === 0} onClick={add}>
        Add to description
      </button>
    </fieldset>
  )
}

export default DescriptionNotes
