import { useEffect, useMemo, useRef, useState } from 'react'
import { RefreshCw } from 'lucide-react'
import {
  NotesService,
  ReadThroughInfo,
  ReadThroughNote,
  ReadThroughRequest,
  ReadThroughService,
  type ReadThroughFile,
} from '@current-client-bindings/app'
import type { DiffFocus } from '../diff/DiffViewer'
import DiffSheet from '../../components/chrome/DiffSheet'
import { Fact, FactList } from '../../components/chrome/FactList'
import Checkbox from '../../components/forms/Checkbox'
import ChangedFileDiff from '../../components/git/ChangedFileDiff'
import PathText from '../../components/git/PathText'
import RefSelect, { useRefGroups } from '../../components/git/RefSelect'
import ChangedFileRow from '../history/ChangedFileRow'
import ReviewComposer from '../review/ReviewComposer'
import ReviewNoteBlock, { noteWhere } from '../review/ReviewNoteBlock'
import { plural } from '../../lib/plural'
import { useCopyToClipboard } from '../../lib/useCopyToClipboard'
import { useActionError } from '../../lib/useActionError'
import { useAsyncData } from '../../lib/useAsyncData'
import { useRovingRows } from '../../lib/useRovingRows'
import SecondLook from './SecondLook'
import { groupFiles, totalsByKind, totalsByStatus, totalsByType, type Total } from './groupFiles'
import { notesText } from './notesText'
import type { ReadThroughOpening } from './readThrough'
import './ReadThroughSheet.scss'

const versionOf = (file: ReadThroughFile) => `${file.newBlob}:${file.path}`

interface ReadThroughSheetProps {
  repoPath: string
  opening: ReadThroughOpening
  refreshKey: unknown
  onNotesChanged: () => void
  onClose: () => void
}

function ReadThroughSheet({ repoPath, opening, refreshKey, onNotesChanged, onClose }: ReadThroughSheetProps) {
  const [target, setTarget] = useState(opening.target ?? '')
  const [uncommitted, setUncommitted] = useState(opening.uncommitted ?? true)
  const {
    data: info,
    error,
    reload,
  } = useAsyncData(
    () =>
      ReadThroughService.Open(
        repoPath,
        new ReadThroughRequest({ ref: opening.ref, path: opening.path, target, uncommitted }),
      ),
    [repoPath, opening, target, uncommitted],
    { refreshKey, keepData: true },
  )
  const refGroups = useRefGroups(repoPath)
  const { error: actionError, setError: setActionError, fail } = useActionError()
  // Null is the overview.
  const [path, setPath] = useState<string | null>(opening.at?.path ?? null)
  const [focusLine, setFocusLine] = useState<DiffFocus | null>(opening.at?.line ? { line: opening.at.line } : null)
  const [noting, setNoting] = useState(false)
  // Ticks shown before they're saved.
  const [ticking, setTicking] = useState<ReadonlyMap<string, boolean>>(new Map())
  const rows = useRovingRows('.read-through-row')

  // Another worktree isn't watched, so what changed there shows on coming back.
  const reloadRef = useRef(reload)
  reloadRef.current = reload
  useEffect(() => {
    const onFocus = () => reloadRef.current()
    window.addEventListener('focus', onFocus)
    return () => window.removeEventListener('focus', onFocus)
  }, [])

  const files = useMemo(
    () => (info?.files ?? []).map((f) => ({ ...f, read: ticking.get(versionOf(f)) ?? f.read })),
    [info, ticking],
  )
  const groups = useMemo(() => groupFiles(files), [files])
  const ordered = useMemo(() => groups.flatMap((group) => group.files), [groups])
  const selected = files.find((f) => f.path === path) ?? null
  const readCount = files.filter((f) => f.read).length

  // Without its files, which every call would send back.
  const range = info && new ReadThroughInfo({ ...info, files: [] })
  const { data: notes, reload: reloadNotes } = useAsyncData(
    () => (range ? ReadThroughService.Notes(range) : null),
    // Where it reads from comes with info.
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [info?.key],
    { refreshKey: info, keepData: true },
  )
  const versions = useMemo(() => info?.files.map((f) => f.newBlob + f.oldBlob).join(), [info])
  // The overview lists them, so they're only read again for it.
  const findingsOf = useRef('')
  if (path === null) findingsOf.current = `${info?.head}:${versions}`
  const { data: report, error: reportError } = useAsyncData(
    () => (range && range.base ? ReadThroughService.Findings(range) : null),
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [info?.key, info?.base],
    { refreshKey: findingsOf.current, keepData: true },
  )
  const noted = () => {
    reloadNotes()
    onNotesChanged()
  }
  const addNote = (file: ReadThroughFile, line: number, oldLine: number, text: string) =>
    ReadThroughService.AddNote(
      range!,
      new ReadThroughNote({ path: file.path, oldPath: file.origPath, line, oldLine, text }),
    ).then(noted)
  const editNote = (id: string, text: string) => NotesService.EditReview(repoPath, id, text).then(noted)
  const openFile = (next: string | null, line = 0, oldLine = 0) => {
    setPath(next)
    setFocusLine(line || oldLine ? { line, oldLine } : null)
    setNoting(false)
  }
  const fileNotes = (notes ?? []).filter((note) => note.path === path)
  const setRead = (file: ReadThroughFile, read: boolean) => {
    if (!range) return
    setActionError(null)
    const tick = (value: boolean) => setTicking((prev) => new Map(prev).set(versionOf(file), value))
    tick(read)
    ReadThroughService.SetRead(range, [file], read)
      .catch((err: unknown) => {
        tick(!read)
        fail(err)
      })
      .finally(reload)
  }
  const nextUnread = () => {
    const at = ordered.findIndex((f) => f.path === path)
    const after = [...ordered.slice(at + 1), ...ordered.slice(0, Math.max(0, at))]
    openFile(after.find((f) => !f.read)?.path ?? null)
  }
  const unreadElsewhere = files.some((f) => !f.read && f.path !== path)

  return (
    <DiffSheet
      title="Read through"
      onClose={onClose}
      headerContent={
        <div className="read-through-header">
          <span className="read-through-name" title={opening.name ?? info?.name}>
            {opening.name ?? info?.name}
          </span>
          <span className="read-through-against">against</span>
          <RefSelect
            groups={refGroups}
            value={info?.target ?? target}
            onChange={setTarget}
            label="Compare with"
            extraOptions={info?.target ? [] : [{ value: '', label: 'Choose a branch' }]}
          />
          {(info?.checkedOut || error) && (
            <Checkbox checked={uncommitted} onChange={setUncommitted} label="Uncommitted changes" />
          )}
          <span className="read-through-progress" role="status">
            {info && files.length > 0 && `${readCount} of ${files.length} read`}
          </span>
          <button type="button" className="read-through-refresh" onClick={reload} aria-label="Refresh" title="Refresh">
            <RefreshCw size={14} strokeWidth={1.75} />
          </button>
        </div>
      }
    >
      <div className="changed-files-browser">
        <div className="changed-files-browser-files read-through-files">
          {error ? (
            <p className="changed-files-browser-error">{error}</p>
          ) : info === null ? (
            <p className="changed-files-browser-hint">Loading changed files…</p>
          ) : info.target === '' ? (
            <p className="changed-files-browser-hint">This repository has no default branch to compare with.</p>
          ) : files.length === 0 ? (
            <p className="changed-files-browser-hint">
              No changes against {info.target}
              {info.uncommitted ? ', committed or not' : ''}.
            </p>
          ) : (
            <div ref={rows}>
              <div className="read-through-row">
                <button
                  type="button"
                  className={path === null ? 'changed-file changed-file-selected' : 'changed-file'}
                  aria-pressed={path === null}
                  onClick={() => openFile(null)}
                >
                  Overview
                </button>
              </div>
              {groups.map((group) => (
                <section key={group.folder} aria-label={group.folder || 'Top folder'}>
                  <h3 className="read-through-folder">
                    <PathText path={group.folder || 'Top folder'} className="read-through-folder-path" />
                    <span className="changed-files-browser-count">{group.files.length}</span>
                  </h3>
                  <ul className="changed-file-list">
                    {group.files.map((file) => (
                      <ChangedFileRow
                        key={file.path}
                        file={file}
                        selected={file.path === path}
                        onSelect={() => openFile(file.path)}
                        relativeTo={group.folder}
                        read={file.read}
                        onRead={(read) => setRead(file, read)}
                      />
                    ))}
                  </ul>
                </section>
              ))}
            </div>
          )}
        </div>
        <div className="changed-files-browser-diff">
          {actionError && <p className="changed-files-browser-error">{actionError}</p>}
          {range && selected ? (
            <>
              <div className="read-through-file">
                <PathText
                  path={selected.origPath ? `${selected.origPath} → ${selected.path}` : selected.path}
                  className="read-through-file-path"
                />
                <button type="button" className="read-through-button" onClick={() => setNoting(true)}>
                  Add note
                </button>
                <Checkbox checked={selected.read} onChange={(read) => setRead(selected, read)} label="Read" />
                <button type="button" className="read-through-button" disabled={!unreadElsewhere} onClick={nextUnread}>
                  Next unread
                </button>
              </div>
              {noting && (
                <ReviewComposer
                  placeholder={`Note on ${selected.path}`}
                  sendLabel="Add note"
                  onSend={(text) => addNote(selected, 0, 0, text).then(() => setNoting(false))}
                  onCancel={() => setNoting(false)}
                />
              )}
              {fileNotes
                .filter((note) => !note.line && !note.oldLine)
                .map((note) => (
                  <ReviewNoteBlock key={note.id} note={note} onEdit={editNote} />
                ))}
              <ChangedFileDiff
                file={selected}
                version={`${range.base}:${range.head}:${selected.oldBlob}:${selected.newBlob}`}
                loadDiff={(file, force, ignoreWhitespace) =>
                  ReadThroughService.FileDiff(range, file.origPath, file.path, force, ignoreWhitespace)
                }
                notes={{
                  items: fileNotes.filter((note) => note.line || note.oldLine),
                  add: (line, oldLine, text) => addNote(selected, line, oldLine, text),
                  edit: editNote,
                }}
                focusLine={focusLine}
                images={{
                  repoPath: range.path,
                  before: { kind: 'commit', rev: range.base },
                  after: range.uncommitted ? { kind: 'worktree', rev: '' } : { kind: 'commit', rev: range.head },
                }}
              />
            </>
          ) : (
            info &&
            files.length > 0 && (
              <>
                <Overview info={info} files={files} />
                <SecondLook
                  report={report}
                  error={reportError}
                  listed={(file) => files.some((f) => f.path === file)}
                  onOpen={(finding) => openFile(finding.path, finding.line, finding.oldLine)}
                />
                <NoteList
                  notes={notes ?? []}
                  onOpen={(note) =>
                    note.outdated ? openFile(note.path) : openFile(note.path, note.line, note.oldLine)
                  }
                />
              </>
            )
          )}
        </div>
      </div>
    </DiffSheet>
  )
}

function Overview({ info, files }: { info: ReadThroughInfo; files: ReadThroughFile[] }) {
  const sum = (count: (file: ReadThroughFile) => number) => files.reduce((total, file) => total + count(file), 0)
  const kinds = totalsByKind(files)
  return (
    <div className="read-through-overview">
      <FactList>
        <Fact label="Files">{plural(files.length, 'file')}</Fact>
        <Fact label="Lines" className="fact-list-data">
          +{sum((f) => f.added)} -{sum((f) => f.removed)}
        </Fact>
        <Fact label="Status">
          <Totals totals={totalsByStatus(files)} />
        </Fact>
        <Fact label="Types" className="fact-list-data read-through-wrapping">
          <Totals totals={totalsByType(files)} />
        </Fact>
        {kinds.length > 0 && (
          <Fact label="Kinds">
            <Totals totals={kinds} />
          </Fact>
        )}
        {info.skipped > 0 && (
          <Fact label="Left out">{plural(info.skipped, 'large untracked file')}, too large to compare</Fact>
        )}
      </FactList>
    </div>
  )
}

function NoteList({ notes, onOpen }: { notes: ReadThroughNote[]; onOpen: (note: ReadThroughNote) => void }) {
  const { copied, copy } = useCopyToClipboard()
  if (notes.length === 0) return null
  return (
    <section className="read-through-notes" aria-label="Notes">
      <div className="changed-files-browser-header">
        <h3 className="changed-files-browser-label">Notes</h3>
        <button type="button" className="read-through-button" onClick={() => copy(notesText(notes))}>
          {copied ? 'Copied' : 'Copy review notes'}
        </button>
      </div>
      <ul className="read-through-note-list">
        {notes.map((note) => (
          <li key={note.id}>
            <button type="button" className="read-through-note" onClick={() => onOpen(note)}>
              <span className="read-through-note-where">
                {note.path} · {noteWhere(note)}
                {note.outdated && <span className="review-note-outdated">Outdated</span>}
              </span>
              <span className="read-through-note-text">{note.text}</span>
            </button>
          </li>
        ))}
      </ul>
    </section>
  )
}

function Totals({ totals }: { totals: Total[] }) {
  return <>{totals.map(({ label, files }) => `${label} ${files}`).join(' · ')}</>
}

export default ReadThroughSheet
