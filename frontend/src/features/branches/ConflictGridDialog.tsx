import { useMemo, useState } from 'react'
import { OverlapService } from '@current-client-bindings/app'
import DialogError from '../../components/chrome/DialogError'
import Modal from '../../components/chrome/Modal'
import Checkbox from '../../components/forms/Checkbox'
import ConflictGrid, { type GridCell } from '../../components/git/ConflictGrid'
import PathText from '../../components/git/PathText'
import { plural } from '../../lib/plural'
import { useAsyncData } from '../../lib/useAsyncData'
import CompareModal from '../history/CompareModal'
import './ConflictGridDialog.scss'

interface ConflictGridDialogProps {
  repoPath: string
  refreshKey?: unknown
  onClose: () => void
}

const pairKey = (a: number, b: number) => `${Math.min(a, b)},${Math.max(a, b)}`

function ConflictGridDialog({ repoPath, refreshKey, onClose }: ConflictGridDialogProps) {
  const {
    data: grid,
    error,
    loading,
    reload,
  } = useAsyncData(() => OverlapService.Grid(repoPath), [repoPath], { refreshKey })
  const [conflictsOnly, setConflictsOnly] = useState(false)
  // By name, which a refresh that reorders the branches doesn't change.
  const [selected, setSelected] = useState<[string, string] | null>(null)
  const [comparing, setComparing] = useState<{ from: string; to: string; path?: string } | null>(null)

  const { cells, names, shown } = useMemo(() => {
    const cells = new Map<string, GridCell>()
    const conflicting = new Set<number>()
    for (const [a, b] of grid?.unchecked ?? []) cells.set(pairKey(a, b), 'unchecked')
    for (const { a, b, files } of grid?.conflicts ?? []) {
      cells.set(pairKey(a, b), { files })
      conflicting.add(a).add(b)
    }
    const names = (grid?.branches ?? []).map((b) => b.name)
    const shown = names.map((_, i) => i).filter((i) => !conflictsOnly || conflicting.has(i))
    return { cells, names, shown }
  }, [grid, conflictsOnly])

  const cellAt = (a: number, b: number): GridCell => cells.get(pairKey(a, b)) ?? 'clean'
  const [from, to] = (selected ?? []).map((name) => shown.findIndex((i) => names[i] === name))
  const opened = selected && from >= 0 && to >= 0 ? cellAt(shown[from], shown[to]) : null

  const leftOut = grid
    ? [
        grid.older > 0 && `${plural(grid.older, 'older branch', 'older branches')}`,
        grid.unrelated.length > 0 && `${grid.unrelated.join(', ')} (no shared history)`,
      ].filter(Boolean)
    : []

  return (
    <Modal title="Conflict grid" onClose={onClose} className="dialog-panel conflict-grid-panel">
      <div className="dialog-body">
        <DialogError>{error}</DialogError>
        {!grid && loading && <p className="dialog-message">Checking each pair of branches…</p>}
        {grid && !grid.supported && <p className="dialog-message">Predicting conflicts needs Git 2.38 or newer.</p>}
        {grid?.supported && names.length < 2 && <p className="dialog-message">No two unmerged local branches.</p>}
        {grid?.supported && names.length >= 2 && (
          <>
            <Checkbox checked={conflictsOnly} onChange={setConflictsOnly} label="Only branches with a conflict" />
            {shown.length < 2 ? (
              <p className="dialog-message">No branches would conflict.</p>
            ) : (
              <ConflictGrid
                label="Conflicts between branches"
                names={shown.map((i) => names[i])}
                cell={(a, b) => cellAt(shown[a], shown[b])}
                selected={opened ? [from, to] : null}
                onSelect={(a, b) => setSelected([names[shown[a]], names[shown[b]]])}
              />
            )}
            {selected && opened && (
              <section className="conflict-grid-opened" aria-label="Opened pair">
                <p className="dialog-message">
                  <span className="conflict-grid-ref">{selected[0]}</span> and{' '}
                  <span className="conflict-grid-ref">{selected[1]}</span>
                  {opened === 'clean' && ' merge without conflicts.'}
                  {opened === 'unchecked' && ' aren’t checked yet.'}
                  {typeof opened !== 'string' && `: ${plural(opened.files.length, 'file')} would conflict.`}
                </p>
                {typeof opened !== 'string' && (
                  <ul className="conflict-grid-files">
                    {opened.files.map((path) => (
                      <li key={path}>
                        <button
                          type="button"
                          title={path}
                          onClick={() => setComparing({ from: selected[0], to: selected[1], path })}
                        >
                          <PathText path={path} />
                        </button>
                      </li>
                    ))}
                  </ul>
                )}
                <button
                  type="button"
                  className="dialog-button"
                  onClick={() => setComparing({ from: selected[0], to: selected[1] })}
                >
                  Compare
                </button>
              </section>
            )}
            <p className="dialog-note">
              Committed work only, newest first, without merged branches.
              {leftOut.length > 0 && ` Left out: ${leftOut.join('; ')}.`}
            </p>
            {grid.unchecked.length > 0 && (
              <p className="dialog-note" role="status">
                {plural(grid.unchecked.length, 'pair')} not checked in time.{' '}
                <button type="button" className="dialog-button" disabled={loading} onClick={reload}>
                  {loading ? 'Checking…' : 'Check more'}
                </button>
              </p>
            )}
          </>
        )}
        <div className="dialog-actions">
          <button type="button" className="dialog-button" onClick={onClose}>
            Close
          </button>
        </div>
      </div>
      {comparing && (
        <CompareModal
          repoPath={repoPath}
          initialFromRef={comparing.from}
          initialToRef={comparing.to}
          initialPath={comparing.path}
          onClose={() => setComparing(null)}
        />
      )}
    </Modal>
  )
}

export default ConflictGridDialog
