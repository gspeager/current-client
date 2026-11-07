import { useEffect, useState } from 'react'
import {
  DndContext,
  KeyboardSensor,
  PointerSensor,
  closestCenter,
  useSensor,
  useSensors,
  type DragEndEvent,
} from '@dnd-kit/core'
import { restrictToVerticalAxis } from '@dnd-kit/modifiers'
import {
  SortableContext,
  arrayMove,
  sortableKeyboardCoordinates,
  useSortable,
  verticalListSortingStrategy,
} from '@dnd-kit/sortable'
import { CSS } from '@dnd-kit/utilities'
import { ArrowDown, ArrowUp, GripVertical } from 'lucide-react'
import { TidyService, type TidyPlanInfo } from '@current-client-bindings/app'
import Modal from '../../components/chrome/Modal'
import { errorMessage } from '../../lib/errors'
import { plural } from '../../lib/plural'
import { useAsyncData } from '../../lib/useAsyncData'
import { shortSha } from '../../lib/text'
import DialogError from '../../components/chrome/DialogError'
import './TidyUpDialog.scss'

interface Row {
  sha: string
  subject: string
  original: string
  action: string
  message: string
}

const ACTIONS = [
  { value: 'keep', label: 'Keep' },
  { value: 'reword', label: 'Reword' },
  { value: 'squash', label: 'Squash' },
  { value: 'fixup', label: 'Fixup' },
  { value: 'drop', label: 'Drop' },
]

const FOLDS = new Set(['squash', 'fixup'])

function rowsFrom(plan: TidyPlanInfo): Row[] {
  return plan.commits.map((commit) => ({
    sha: commit.sha,
    subject: commit.subject,
    original: commit.message,
    action: commit.suggested || 'keep',
    message: commit.message,
  }))
}

const reducedMotion = () => window.matchMedia?.('(prefers-reduced-motion: reduce)').matches ?? false

interface CommitRowProps {
  row: Row
  index: number
  count: number
  // Nothing above it to fold into.
  canFold: boolean
  outcome?: { outcome: string; files: string[] }
  onChange: (patch: Partial<Row>) => void
  onMove: (to: number) => void
}

function CommitRow({ row, index, count, canFold, outcome, onChange, onMove }: CommitRowProps) {
  // The one place the design rules let a row move: aside, within 140 ms, with no settle on drop.
  const { attributes, listeners, setNodeRef, transform, transition, isDragging } = useSortable({
    id: row.sha,
    animateLayoutChanges: () => false,
    transition: reducedMotion() ? null : { duration: 140, easing: 'cubic-bezier(0.2, 0, 0.3, 1)' },
  })
  const label = `${shortSha(row.sha)} ${row.subject}`
  return (
    <li
      ref={setNodeRef}
      className={isDragging ? 'tidy-row tidy-row-dragging' : 'tidy-row'}
      style={{ transform: CSS.Translate.toString(transform), transition }}
    >
      <div className="tidy-row-line">
        <button type="button" className="tidy-handle" aria-label={`Move ${label}`} {...attributes} {...listeners}>
          <GripVertical size={14} strokeWidth={1.75} />
        </button>
        <span className="tidy-sha">{shortSha(row.sha)}</span>
        <span
          className={row.action === 'drop' ? 'tidy-subject tidy-subject-dropped' : 'tidy-subject'}
          title={row.subject}
        >
          {row.subject}
        </span>
        {outcome?.outcome === 'conflict' && <span className="tidy-conflict">Conflict</span>}
        <button
          type="button"
          className="tidy-move"
          aria-label={`Move ${label} up`}
          disabled={index === 0}
          onClick={() => onMove(index - 1)}
        >
          <ArrowUp size={14} strokeWidth={1.75} />
        </button>
        <button
          type="button"
          className="tidy-move"
          aria-label={`Move ${label} down`}
          disabled={index === count - 1}
          onClick={() => onMove(index + 1)}
        >
          <ArrowDown size={14} strokeWidth={1.75} />
        </button>
        <select
          className="tidy-action"
          aria-label={`Action for ${label}`}
          value={row.action}
          onChange={(e) => onChange({ action: e.target.value })}
        >
          {ACTIONS.filter((action) => canFold || !FOLDS.has(action.value)).map((action) => (
            <option key={action.value} value={action.value}>
              {action.label}
            </option>
          ))}
        </select>
      </div>
      {row.action === 'reword' && (
        <textarea
          className="tidy-message"
          aria-label={`Message for ${label}`}
          value={row.message}
          onChange={(e) => onChange({ message: e.target.value })}
        />
      )}
      {outcome?.outcome === 'conflict' && outcome.files.length > 0 && (
        <p className="tidy-files">{outcome.files.join(', ')}</p>
      )}
    </li>
  )
}

interface TidyUpDialogProps {
  repoPath: string
  onDone: () => void
  onClose: () => void
}

function TidyUpDialog({ repoPath, onDone, onClose }: TidyUpDialogProps) {
  const { data: plan, error: planError } = useAsyncData(() => TidyService.Plan(repoPath), [repoPath])
  const [rows, setRows] = useState<Row[]>([])
  const [running, setRunning] = useState(false)
  const [runError, setRunError] = useState<string | null>(null)
  useEffect(() => {
    if (plan) setRows(rowsFrom(plan))
  }, [plan])

  const sensors = useSensors(
    useSensor(PointerSensor, { activationConstraint: { distance: 4 } }),
    useSensor(KeyboardSensor, { coordinateGetter: sortableKeyboardCoordinates }),
  )

  const steps = rows.map((row) => ({
    sha: row.sha,
    action: row.action,
    message: row.action === 'reword' ? row.message : '',
  }))
  const invalid = (() => {
    let kept = 0
    for (const row of rows) {
      if (row.action === 'keep' || row.action === 'reword') kept++
      if (FOLDS.has(row.action) && kept === 0) return 'The first commit kept has nothing to fold into.'
      if (row.action === 'reword' && row.message.trim() === '') return 'A reworded commit needs a message.'
    }
    return rows.length > 0 && kept === 0 ? 'At least one commit has to stay.' : null
  })()
  const changed = plan
    ? rows.some(
        (row, i) =>
          row.sha !== plan.commits[i]?.sha ||
          (row.action !== 'keep' && !(row.action === 'reword' && row.message === row.original)),
      )
    : false

  const stepsKey = JSON.stringify(steps)
  const { data: preview } = useAsyncData(
    () => (plan && rows.length > 0 && !invalid && !plan.hasMerges ? TidyService.Preview(repoPath, steps) : null),
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [repoPath, plan, stepsKey, invalid],
    { keepData: true },
  )

  const patch = (sha: string, change: Partial<Row>) =>
    setRows((prev) => prev.map((row) => (row.sha === sha ? { ...row, ...change } : row)))
  const move = (from: number, to: number) => setRows((prev) => arrayMove(prev, from, to))
  const dragEnd = ({ active, over }: DragEndEvent) => {
    if (!over || active.id === over.id) return
    const from = rows.findIndex((row) => row.sha === active.id)
    const to = rows.findIndex((row) => row.sha === over.id)
    move(from, to)
  }

  const run = () => {
    if (!plan) return
    setRunning(true)
    setRunError(null)
    TidyService.Run(repoPath, plan.head, steps)
      .then(onDone)
      .catch((err: unknown) => {
        setRunError(errorMessage(err))
        setRunning(false)
      })
  }

  let kept = 0
  const blocked = !plan || !changed || !!invalid || plan.dirty || plan.hasMerges || running
  return (
    <Modal title="Tidy up history" onClose={onClose} className="dialog-panel tidy-panel">
      <div className="dialog-body">
        <DialogError>{planError}</DialogError>
        {plan && rows.length === 0 && <p className="dialog-message">No commits to tidy on this branch.</p>}
        {plan && rows.length > 0 && (
          <>
            <p className="dialog-note">
              {plural(rows.length, 'commit')}
              {plan.baseName ? ` since ${plan.baseName}` : ''}, oldest first.
            </p>
            <DndContext
              sensors={sensors}
              collisionDetection={closestCenter}
              modifiers={[restrictToVerticalAxis]}
              onDragEnd={dragEnd}
            >
              <SortableContext items={rows.map((row) => row.sha)} strategy={verticalListSortingStrategy}>
                <ol className="tidy-list">
                  {rows.map((row, index) => {
                    const canFold = kept > 0
                    if (row.action === 'keep' || row.action === 'reword') kept++
                    return (
                      <CommitRow
                        key={row.sha}
                        row={row}
                        index={index}
                        count={rows.length}
                        canFold={canFold || FOLDS.has(row.action)}
                        outcome={preview?.outcomes.find((step) => step.sha === row.sha)}
                        onChange={(change) => patch(row.sha, change)}
                        onMove={(to) => move(index, to)}
                      />
                    )
                  })}
                </ol>
              </SortableContext>
            </DndContext>
            {preview && !invalid && (
              <div className="tidy-result">
                <p className="dialog-note">Makes {plural(preview.result.length, 'commit')}:</p>
                <ol className="tidy-result-list">
                  {preview.result.map((commit, index) => (
                    <li key={index}>
                      {commit.subject}
                      {commit.folded > 0 && <span className="tidy-folded"> (+{commit.folded} folded in)</span>}
                    </li>
                  ))}
                </ol>
              </div>
            )}
            {preview && !preview.predicted && (
              <p className="dialog-note">Conflicts can't be predicted here. Predicting needs Git 2.40 or newer.</p>
            )}
            {preview?.outcomes.some((step) => step.outcome === 'conflict') && (
              <p className="dialog-note">Stops on the conflict, with Continue and Abort in the banner.</p>
            )}
            {preview?.forcePush && (
              <p className="dialog-note">
                {preview.upstream} has these commits, so pushing afterwards needs a force push with --force-with-lease.
              </p>
            )}
            {plan.hasMerges && <DialogError>These commits include a merge, which a rewrite would flatten.</DialogError>}
            {plan.dirty && <DialogError>Commit or stash the changes in the working tree first.</DialogError>}
            <DialogError>{invalid}</DialogError>
            <DialogError>{runError}</DialogError>
          </>
        )}
        <div className="dialog-actions">
          <button type="button" className="dialog-button" onClick={onClose}>
            Cancel
          </button>
          <button type="button" className="dialog-button dialog-button-primary" disabled={blocked} onClick={run}>
            {running ? 'Rewriting…' : 'Rewrite'}
          </button>
        </div>
      </div>
    </Modal>
  )
}

export default TidyUpDialog
