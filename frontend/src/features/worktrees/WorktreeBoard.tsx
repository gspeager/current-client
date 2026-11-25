import { useMemo, useState } from 'react'
import { BookOpen, FolderOpen, Play, SquareTerminal, TriangleAlert } from 'lucide-react'
import type { BoardColumn, BoardInfo } from '@current-client-bindings/app'
import DialogError from '../../components/chrome/DialogError'
import Modal from '../../components/chrome/Modal'
import BranchPill from '../../components/git/BranchPill'
import ConflictGrid, { type GridCell } from '../../components/git/ConflictGrid'
import DiffStat from '../../components/git/DiffStat'
import PathText from '../../components/git/PathText'
import StatusBadge from '../../components/git/StatusBadge'
import { useLaneColors } from '../../lib/laneColor'
import { baseName } from '../../lib/paths'
import { plural } from '../../lib/plural'
import { shortSha } from '../../lib/text'
import { useOpenReadThrough } from '../read-through/readThrough'
import { useBoard } from './useBoard'
import './WorktreeBoard.scss'

// More of a worktree's files than this say no more than their number.
const MOST_FILES = 100

const UNPREDICTED: Record<string, string> = {
  git: 'Predicting collisions needs Git 2.38 or newer.',
  partial: 'Collisions aren’t predicted in a partial clone, where merging fetches what was left out.',
}

interface WorktreeBoardProps {
  repoPath: string
  onOpenRepository: (path: string) => void
  // Not given when there's no terminal.
  onOpenTerminal?: (path: string) => void
  onNewWorktree: () => void
  onClose: () => void
}

// A folder's name, or its path where another has the same name.
function namesOf(columns: BoardColumn[]): string[] {
  const names = columns.map((c) => baseName(c.worktree.path))
  return names.map((name, i) => (names.indexOf(name) === names.lastIndexOf(name) ? name : columns[i].worktree.path))
}

// By column, each file that collides and with which others.
function collidingFiles(board: BoardInfo, names: string[]): Map<string, string[]>[] {
  const byColumn = names.map(() => new Map<string, string[]>())
  const add = (column: number, file: string, other: number) =>
    byColumn[column].set(file, [...(byColumn[column].get(file) ?? []), names[other]])
  for (const { a, b, files } of board.collisions) {
    for (const file of files) {
      add(a, file, b)
      add(b, file, a)
    }
  }
  return byColumn
}

function WorktreeBoard({
  repoPath,
  onOpenRepository,
  onOpenTerminal,
  onNewWorktree,
  onClose,
}: WorktreeBoardProps) {
  const { board, error } = useBoard(repoPath)
  const names = useMemo(() => namesOf(board?.columns ?? []), [board])
  const colliding = useMemo(() => (board ? collidingFiles(board, names) : []), [board, names])

  return (
    <Modal
      title="Worktree board"
      onClose={onClose}
      className="worktree-board-panel"
      headerContent={
        <button type="button" className="worktree-board-new" onClick={onNewWorktree}>
          New worktree
        </button>
      }
    >
      <div className="worktree-board">
        <DialogError>{error}</DialogError>
        {!board && !error && <p className="dialog-message">Reading each worktree…</p>}
        {board && (
          <>
            <div className="worktree-board-columns">
              {board.columns.map((column, i) => (
                <Column
                  key={column.worktree.path}
                  name={names[i]}
                  column={column}
                  base={board.base}
                  colliding={colliding[i]}
                  onOpen={onOpenRepository}
                  onTerminal={onOpenTerminal}
                />
              ))}
            </div>
            {board.more > 0 && <p className="dialog-note">{plural(board.more, 'more worktree')} not shown.</p>}
            {board.unpredicted ? (
              <p className="dialog-note">{UNPREDICTED[board.unpredicted]}</p>
            ) : (
              names.length > 1 && <Collisions board={board} names={names} />
            )}
          </>
        )}
      </div>
    </Modal>
  )
}

interface ColumnProps {
  name: string
  column: BoardColumn
  base: string
  colliding: Map<string, string[]>
  onOpen: (path: string) => void
  onTerminal?: (path: string) => void
}

function Column({ name, column, base, colliding, onOpen, onTerminal }: ColumnProps) {
  const { branchColor } = useLaneColors()
  const readThrough = useOpenReadThrough()
  const { worktree, files } = column
  const changed = new Set(files.map((file) => file.path))
  // Committed, so not among the uncommitted ones.
  const committed = [...colliding.keys()].filter((path) => !changed.has(path))
  const listed = [
    ...committed.map((path) => ({ path, status: '' })),
    ...[...files].sort((a, b) => Number(colliding.has(b.path)) - Number(colliding.has(a.path))),
  ]
  return (
    <section className="worktree-column" aria-label={name}>
      <h3 className="worktree-column-name" title={worktree.path}>
        {name}
      </h3>
      <div className="worktree-column-branch">
        {worktree.branch ? (
          <BranchPill name={worktree.branch} color={branchColor(worktree.branch, worktree.current)} />
        ) : (
          <span className="worktree-column-detached">detached {shortSha(worktree.head)}</span>
        )}
        {worktree.current && <span className="worktree-row-badge">current</span>}
        {worktree.locked && <span className="worktree-row-badge">locked</span>}
        {worktree.missing && <span className="worktree-row-badge worktree-row-badge-missing">missing</span>}
        {column.running && onTerminal && (
          <span className="worktree-row-badge worktree-column-running" title="A process is running in its terminal">
            <Play size={10} strokeWidth={2} aria-hidden="true" />
            running
          </span>
        )}
      </div>
      {column.problem ? (
        <DialogError>{column.problem}</DialogError>
      ) : (
        !worktree.missing && (
          <>
            <p className="worktree-column-facts">
              {files.length > 0 ? plural(files.length, 'changed file') : 'No changes'}
              <DiffStat added={column.added} removed={column.removed} binary={false} />
            </p>
            {base && (
              <p className="worktree-column-facts" title={`Commits ahead of and behind ${base}`}>
                <span className="worktree-column-counts">
                  ↑{column.ahead} ↓{column.behind}
                </span>
                {base}
              </p>
            )}
            {column.skipped > 0 && (
              <p className="dialog-note">{plural(column.skipped, 'large untracked file')} not counted.</p>
            )}
            <ul className="worktree-column-files">
              {listed.slice(0, MOST_FILES).map(({ path, status }) => {
                const others = colliding.get(path)
                return (
                  <li key={path} className={others ? 'worktree-column-colliding' : undefined}>
                    {status && <StatusBadge status={status} />}
                    <PathText path={path} />
                    {others && (
                      <TriangleAlert
                        size={12}
                        strokeWidth={1.75}
                        role="img"
                        aria-label={`Collides with ${others.join(', ')}`}
                      >
                        <title>Collides with {others.join(', ')}</title>
                      </TriangleAlert>
                    )}
                  </li>
                )
              })}
              {listed.length > MOST_FILES && <li>{plural(listed.length - MOST_FILES, 'more file')}</li>}
            </ul>
          </>
        )
      )}
      {!worktree.missing && (
        <div className="worktree-column-actions">
          {!worktree.current && (
            <button type="button" aria-label={`Open ${name}`} onClick={() => onOpen(worktree.path)}>
              <FolderOpen size={14} strokeWidth={1.75} aria-hidden="true" />
              Open
            </button>
          )}
          {onTerminal && (
            <button type="button" aria-label={`Terminal in ${name}`} onClick={() => onTerminal(worktree.path)}>
              <SquareTerminal size={14} strokeWidth={1.75} aria-hidden="true" />
              Terminal
            </button>
          )}
          <button
            type="button"
            aria-label={`Read through ${name}`}
            onClick={() => readThrough({ path: worktree.path })}
          >
            <BookOpen size={14} strokeWidth={1.75} aria-hidden="true" />
            Read through
          </button>
        </div>
      )}
    </section>
  )
}

function Collisions({ board, names }: { board: BoardInfo; names: string[] }) {
  // By name, which a worktree added or removed doesn't change.
  const [selected, setSelected] = useState<[string, string] | null>(null)
  const cell = (a: number, b: number): GridCell => {
    const same = (x: number, y: number) => (x === a && y === b) || (x === b && y === a)
    const found = board.collisions.find((c) => same(c.a, c.b))
    if (found) return { files: found.files }
    return board.unchecked.some(([x, y]) => same(x, y)) ? 'unchecked' : 'clean'
  }
  const [a, b] = (selected ?? []).map((name) => names.indexOf(name))
  const opened = selected && a >= 0 && b >= 0 ? cell(a, b) : null
  return (
    <section className="worktree-board-collisions" aria-label="Collisions">
      <h3 className="worktree-column-name">Collisions</h3>
      <ConflictGrid
        label="Collisions between worktrees, uncommitted work included"
        names={names}
        cell={cell}
        selected={opened ? [a, b] : null}
        onSelect={(x, y) => setSelected([names[x], names[y]])}
      />
      {selected && opened && (
        <div role="region" aria-label="Opened pair">
          <p className="dialog-message">
            {selected[0]} and {selected[1]}
            {opened === 'clean' && ' don’t collide.'}
            {opened === 'unchecked' && ' aren’t checked yet.'}
            {typeof opened !== 'string' && `: ${plural(opened.files.length, 'file')} would conflict.`}
          </p>
          {typeof opened !== 'string' && (
            <ul className="worktree-column-files">
              {opened.files.map((path) => (
                <li key={path}>
                  <PathText path={path} />
                </li>
              ))}
            </ul>
          )}
        </div>
      )}
    </section>
  )
}

export default WorktreeBoard
