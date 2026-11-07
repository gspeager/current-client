import { useMemo, useState } from 'react'
import { BranchService, type BranchReviewRowInfo } from '@current-client-bindings/app'
import Modal from '../../components/chrome/Modal'
import SegmentedControl from '../../components/controls/SegmentedControl'
import Checkbox from '../../components/forms/Checkbox'
import IdentityBadge from '../../components/git/IdentityBadge'
import { errorMessage } from '../../lib/errors'
import { plural } from '../../lib/plural'
import { relativeTime } from '../../lib/relativeTime'
import { toggled } from '../../lib/sets'
import { useAsyncData } from '../../lib/useAsyncData'
import { useDialogs } from '../../lib/useDialogs'
import type { DeletedBranch } from './CleanUpBranchesDialog'
import DialogError from '../../components/chrome/DialogError'
import './BranchReviewDialog.scss'

type Filter = 'all' | 'merged' | 'stale' | 'unpushed'
type Column = 'name' | 'upstream' | 'base' | 'merged' | 'date' | 'author'

const COLUMN_VALUE: Record<Column, (row: BranchReviewRowInfo) => string | number> = {
  name: (row) => row.name,
  upstream: (row) => row.ahead + row.behind,
  base: (row) => row.aheadBase + row.behindBase,
  merged: (row) => (row.merged ? 1 : 0),
  date: (row) => Date.parse(row.lastCommitDate) || 0,
  author: (row) => row.authorName,
}

function compare(a: string | number, b: string | number): number {
  return typeof a === 'number' && typeof b === 'number' ? a - b : String(a).localeCompare(String(b))
}

interface BranchReviewDialogProps {
  repoPath: string
  onDeleted: (branches: DeletedBranch[]) => void
  onClose: () => void
}

function BranchReviewDialog({ repoPath, onDeleted, onClose }: BranchReviewDialogProps) {
  const { confirm } = useDialogs()
  const { data: review, error: loadError } = useAsyncData(() => BranchService.ReviewBranches(repoPath), [repoPath])
  const [filter, setFilter] = useState<Filter>('all')
  const [sort, setSort] = useState<{ column: Column; ascending: boolean }>({ column: 'date', ascending: false })
  const [selected, setSelected] = useState<Set<string>>(new Set())
  const [running, setRunning] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const rows = useMemo(() => {
    const shown = (review?.rows ?? []).filter((row) => filter === 'all' || row[filter])
    const value = COLUMN_VALUE[sort.column]
    return [...shown].sort((a, b) => compare(value(a), value(b)) * (sort.ascending ? 1 : -1))
  }, [review, filter, sort])

  const shownDeletable = rows.filter((row) => row.deletable)
  // Only what's on screen: a selection hidden by the filter isn't deleted.
  const toDelete = shownDeletable.filter((row) => selected.has(row.name))
  const allSelected = shownDeletable.length > 0 && shownDeletable.every((row) => selected.has(row.name))

  const header = (column: Column, label: string) => (
    <th scope="col" aria-sort={sort.column === column ? (sort.ascending ? 'ascending' : 'descending') : 'none'}>
      <button
        type="button"
        onClick={() =>
          setSort((prev) => ({ column, ascending: prev.column === column ? !prev.ascending : column === 'name' }))
        }
      >
        {label}
        {sort.column === column && <span aria-hidden="true">{sort.ascending ? ' ↑' : ' ↓'}</span>}
      </button>
    </th>
  )

  const run = async () => {
    const unmerged = toDelete.filter((row) => !row.merged)
    if (unmerged.length > 0) {
      const confirmed = await confirm({
        title: 'Delete unmerged branches',
        message: `${plural(unmerged.length, 'branch', 'branches')} not merged into ${review?.base}: ${unmerged
          .map((row) => row.name)
          .join(', ')}. Undo restores them until the next action.`,
        confirmLabel: 'Delete',
      })
      if (!confirmed) return
    }
    setRunning(true)
    setError(null)
    const deleted: DeletedBranch[] = []
    try {
      for (const row of toDelete) {
        deleted.push({ name: row.name, tip: await BranchService.DeleteBranch(repoPath, row.name, true) })
      }
    } catch (err: unknown) {
      setError(errorMessage(err))
      setRunning(false)
      if (deleted.length > 0) onDeleted(deleted)
      return
    }
    onDeleted(deleted)
  }

  return (
    <Modal title="Branch review" onClose={onClose} className="dialog-panel branch-review-panel">
      <div className="dialog-body">
        <DialogError>{loadError}</DialogError>
        <div className="branch-review-filters">
          <SegmentedControl
            value={filter}
            onChange={setFilter}
            options={[
              { value: 'all', label: 'All' },
              { value: 'merged', label: 'Merged' },
              { value: 'stale', label: 'Stale' },
              { value: 'unpushed', label: 'Unpushed' },
            ]}
          />
          {review && <span className="branch-review-base">Compared with {review.base || 'no default branch'}</span>}
        </div>
        {review && rows.length === 0 && <p className="dialog-message">No branches match.</p>}
        {rows.length > 0 && (
          <div className="branch-review-scroll">
            <table className="branch-review-table">
              <thead>
                <tr>
                  <th scope="col">
                    <Checkbox
                      checked={allSelected}
                      onChange={(checked) =>
                        setSelected((prev) =>
                          shownDeletable.reduce((next, row) => toggled(next, row.name, checked), prev),
                        )
                      }
                      ariaLabel="Select all shown"
                    />
                  </th>
                  {header('name', 'Branch')}
                  {header('upstream', 'Upstream')}
                  {header('base', 'Base')}
                  {header('merged', 'Merged')}
                  {header('date', 'Last commit')}
                  {header('author', 'Author')}
                </tr>
              </thead>
              <tbody>
                {rows.map((row) => (
                  <tr key={row.name}>
                    <td>
                      <Checkbox
                        checked={selected.has(row.name)}
                        disabled={!row.deletable}
                        onChange={(checked) => setSelected((prev) => toggled(prev, row.name, checked))}
                        ariaLabel={row.name}
                      />
                    </td>
                    <td className="branch-review-name" title={row.name}>
                      {row.name}
                      {row.openChange && <span className="branch-review-change">{row.openChange}</span>}
                    </td>
                    <td className="branch-review-data">
                      {row.upstream ? `↑${row.ahead} ↓${row.behind}` : <span title="No upstream">none</span>}
                    </td>
                    <td className="branch-review-data">{`↑${row.aheadBase} ↓${row.behindBase}`}</td>
                    <td className="branch-review-data">{row.mergedChange || (row.merged ? 'Merged' : '')}</td>
                    <td className="branch-review-date">
                      {row.lastCommitDate && relativeTime(new Date(row.lastCommitDate))}
                    </td>
                    <td>
                      {row.authorName && (
                        <IdentityBadge identity={row.author} name={row.authorName} email={row.authorEmail} />
                      )}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
        <DialogError>{error}</DialogError>
        <div className="dialog-actions">
          <button type="button" className="dialog-button" onClick={onClose}>
            Close
          </button>
          <button
            type="button"
            className="dialog-button dialog-button-destructive"
            disabled={toDelete.length === 0 || running}
            onClick={() => void run()}
          >
            {running ? 'Deleting…' : `Delete ${plural(toDelete.length, 'branch', 'branches')}`}
          </button>
        </div>
      </div>
    </Modal>
  )
}

export default BranchReviewDialog
