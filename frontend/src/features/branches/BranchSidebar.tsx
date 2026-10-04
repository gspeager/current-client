import { useState, type MouseEvent } from 'react'
import { GitBranchPlus, Pencil, TriangleAlert, X } from 'lucide-react'
import {
  BranchService,
  GitFlowService,
  OverlapService,
  type BranchInfo,
  type OverlapInfo,
} from '@current-client-bindings/app'
import CompareModal from '../history/CompareModal'
import ContextMenu, { type ContextMenuState } from '../../components/controls/ContextMenu'
import { useDeleteRemoteBranch } from '../remotes/useDeleteRemoteBranch'
import { toBranchName } from '../../lib/branchName'
import { errorMessage } from '../../lib/errors'
import { useLaneColors } from '../../lib/laneColor'
import { relativeTime } from '../../lib/relativeTime'
import { useAsyncData } from '../../lib/useAsyncData'
import { useDialogs } from '../../lib/useDialogs'
import BranchPill from '../../components/git/BranchPill'
import './BranchSidebar.scss'

interface BranchSidebarProps {
  repoPath: string
  dirty: boolean
  refreshKey: number
  onBranchChanged?: () => void
}

type NewBranchKind = 'branch' | 'feature' | 'release' | 'hotfix'

function BranchSidebar({ repoPath, dirty, refreshKey, onBranchChanged }: BranchSidebarProps) {
  const { branchColor } = useLaneColors()
  const {
    data: branches,
    error: loadError,
    reload: loadBranches,
  } = useAsyncData(() => BranchService.ListLocal(repoPath), [repoPath], { refreshKey })
  const { data: defaultBranch } = useAsyncData(() => BranchService.DefaultBranch(repoPath), [repoPath], {
    refreshKey,
  })
  const { data: overlapReport } = useAsyncData(() => OverlapService.Predict(repoPath), [repoPath], { refreshKey })
  const overlaps = overlapReport?.overlaps ?? []
  const localOverlaps = new Map(overlaps.filter((o) => !o.remote).map((o) => [o.branch, o]))
  const remoteOverlaps = overlaps.filter((o) => o.remote)
  const { confirm, prompt } = useDialogs()
  const [actionError, setActionError] = useState<string | null>(null)
  const error = loadError ?? actionError
  const [newBranchName, setNewBranchName] = useState('')
  const [newBranchKind, setNewBranchKind] = useState<NewBranchKind>('branch')
  const [showNewBranch, setShowNewBranch] = useState(false)
  const newBranch = toBranchName(newBranchName)
  const [contextMenu, setContextMenu] = useState<ContextMenuState | null>(null)
  // Refreshes the graph too, which still shows the deleted remote branch.
  const deleteRemote = useDeleteRemoteBranch(repoPath, () => {
    loadBranches()
    onBranchChanged?.()
  })
  const [compareWith, setCompareWith] = useState<string | null>(null)
  const [deleted, setDeleted] = useState<{ name: string; tip: string } | null>(null)

  const checkoutBranch = (name: string) => {
    setActionError(null)
    BranchService.CheckoutBranch(repoPath, name)
      .then(() => {
        loadBranches()
        onBranchChanged?.()
      })
      .catch((err: unknown) => setActionError(errorMessage(err)))
  }

  const createBranch = () => {
    setActionError(null)
    const create =
      newBranchKind === 'branch'
        ? BranchService.CreateBranch(repoPath, newBranch).then(() => BranchService.CheckoutBranch(repoPath, newBranch))
        : GitFlowService.StartBranch(repoPath, newBranchKind, newBranch)
    create
      .then(() => {
        setNewBranchName('')
        setNewBranchKind('branch')
        setShowNewBranch(false)
        loadBranches()
        onBranchChanged?.()
      })
      .catch((err: unknown) => setActionError(errorMessage(err)))
  }

  const renameBranch = async (oldName: string) => {
    const newName = await prompt({
      title: 'Rename branch',
      label: 'New branch name',
      initialValue: oldName,
      confirmLabel: 'Rename',
      transform: toBranchName,
    })
    if (!newName) return
    setActionError(null)
    BranchService.RenameBranch(repoPath, oldName, newName)
      .then(loadBranches)
      .catch((err: unknown) => setActionError(errorMessage(err)))
  }

  const mergeBranch = (name: string) => {
    setActionError(null)
    BranchService.MergeBranch(repoPath, name)
      .catch((err: unknown) => setActionError(errorMessage(err)))
      .finally(() => {
        // A failed merge leaves conflicts the banner must pick up.
        loadBranches()
        onBranchChanged?.()
      })
  }

  const mergeDefaultBranch = async (name: string) => {
    const target = branches?.find((b) => b.name === name)
    if (target && target.behind > 0) {
      const confirmed = await confirm({
        title: `Merge ${name}`,
        message: `${name} is ${target.behind} commit${target.behind === 1 ? '' : 's'} behind ${target.upstream}. Pull ${name} first to include the latest work, or merge it as it is.`,
        confirmLabel: 'Merge anyway',
      })
      if (!confirmed) return
    }
    mergeBranch(name)
  }

  const rebaseOnto = (name: string) => {
    setActionError(null)
    BranchService.RebaseOnto(repoPath, name)
      .catch((err: unknown) => setActionError(errorMessage(err)))
      .finally(() => {
        loadBranches()
        onBranchChanged?.()
      })
  }

  const deleteBranch = (name: string, force: boolean) => {
    setActionError(null)
    setDeleted(null)
    BranchService.DeleteBranch(repoPath, name, force)
      .then((tip) => {
        setDeleted({ name, tip })
        loadBranches()
      })
      .catch(async (err: unknown) => {
        if (!force && errorMessage(err) === 'This branch has unmerged changes.') {
          const confirmed = await confirm({
            title: 'Delete branch',
            message: `"${name}" has unmerged changes. Delete anyway?`,
            confirmLabel: 'Delete',
            destructive: true,
          })
          if (confirmed) deleteBranch(name, true)
          return
        }
        setActionError(errorMessage(err))
      })
  }

  const restoreDeleted = (branch: { name: string; tip: string }) => {
    setActionError(null)
    setDeleted(null)
    BranchService.CreateBranchAt(repoPath, branch.name, branch.tip)
      .then(loadBranches)
      .catch((err: unknown) => setActionError(`Could not restore ${branch.name}: ${errorMessage(err)}`))
  }

  const branchContextMenu = (b: BranchInfo) => (e: MouseEvent) => {
    e.preventDefault()
    setContextMenu({
      x: e.clientX,
      y: e.clientY,
      items: [
        { label: 'Checkout', onClick: () => checkoutBranch(b.name), disabled: b.current },
        { label: 'Merge into current', onClick: () => mergeBranch(b.name), disabled: b.current },
        ...(b.current && defaultBranch && defaultBranch !== b.name
          ? [{ label: `Merge ${defaultBranch} into current`, onClick: () => mergeDefaultBranch(defaultBranch) }]
          : []),
        { label: 'Rebase current onto', onClick: () => rebaseOnto(b.name), disabled: b.current },
        { label: 'Compare with current', onClick: () => setCompareWith(b.name), disabled: b.current },
        { label: 'Rename…', onClick: () => renameBranch(b.name) },
        { label: 'Delete', onClick: () => deleteBranch(b.name, false), destructive: true, disabled: b.current },
        ...(b.upstream
          ? [
              {
                label: `Delete ${b.upstream} on remote…`,
                onClick: () => void deleteRemote.deleteRemoteBranch(b.upstream),
                destructive: true,
                disabled: deleteRemote.running,
              },
            ]
          : []),
      ],
    })
  }

  return (
    <div className="branch-sidebar">
      <div className="branch-sidebar-header">
        <span className="branch-sidebar-label">Branches</span>
        <button
          type="button"
          className="branch-sidebar-add"
          onClick={() => setShowNewBranch((v) => !v)}
          aria-label="New branch"
          title="New branch"
        >
          <GitBranchPlus size={18} strokeWidth={1.75} />
        </button>
      </div>

      {showNewBranch && (
        <div className="branch-sidebar-new">
          <select
            aria-label="new branch kind"
            className="branch-sidebar-new-kind"
            value={newBranchKind}
            onChange={(e) => setNewBranchKind(e.target.value as NewBranchKind)}
          >
            <option value="branch">Branch</option>
            <option value="feature">Start feature</option>
            <option value="release">Start release</option>
            <option value="hotfix">Start hotfix</option>
          </select>
          <div className="branch-sidebar-new-row">
            {newBranchKind !== 'branch' && <span className="branch-sidebar-new-prefix">{newBranchKind}/</span>}
            <input
              aria-label="new branch name"
              placeholder={newBranchKind === 'branch' ? 'New branch name' : 'Name'}
              value={newBranchName}
              onChange={(e) => setNewBranchName(e.target.value)}
              autoFocus
            />
            <button type="button" onClick={createBranch} disabled={newBranch === ''}>
              Create
            </button>
          </div>
          {newBranch !== '' && newBranch !== newBranchName.trim() && (
            <p className="branch-sidebar-new-result" role="status">
              Will be created as{' '}
              <span className="branch-sidebar-new-result-name">
                {newBranchKind === 'branch' ? '' : `${newBranchKind}/`}
                {newBranch}
              </span>
            </p>
          )}
        </div>
      )}

      {error && <p className="branch-sidebar-error">{error}</p>}
      {deleteRemote.error && <p className="branch-sidebar-error">Could not delete on remote: {deleteRemote.error}</p>}
      {deleted && (
        <p className="branch-sidebar-deleted" role="status">
          <span>
            Deleted <span className="branch-sidebar-deleted-name">{deleted.name}</span>
          </span>
          <button type="button" onClick={() => restoreDeleted(deleted)}>
            Undo
          </button>
        </p>
      )}
      {branches === null ? (
        <p className="branch-sidebar-hint">Loading branches…</p>
      ) : branches.length === 0 ? (
        <p className="branch-sidebar-hint">No branches.</p>
      ) : (
        <ul className="branch-sidebar-list">
          {branches.map((b) => (
            <li
              key={b.name}
              className={b.current ? 'branch-row branch-row-current' : 'branch-row'}
              onClick={() => !b.current && checkoutBranch(b.name)}
              onContextMenu={branchContextMenu(b)}
              role="button"
              tabIndex={b.current ? -1 : 0}
              onKeyDown={(e) => {
                if ((e.key === 'Enter' || e.key === ' ') && !b.current) {
                  e.preventDefault()
                  checkoutBranch(b.name)
                }
              }}
            >
              <BranchPill name={b.name} color={branchColor(b.name, b.current)} current={b.current} dirty={dirty} />
              {b.current && <span className="branch-row-head">HEAD</span>}
              <OverlapMark overlap={localOverlaps.get(b.name)} />
              <span className="branch-row-meta">
                {b.ahead > 0 && <span className="branch-row-ahead">↑{b.ahead}</span>}
                {b.behind > 0 && <span className="branch-row-behind">↓{b.behind}</span>}
                {b.lastCommitDate && (
                  <span className="branch-row-time">{relativeTime(new Date(b.lastCommitDate))}</span>
                )}
              </span>
              <span className="branch-row-actions">
                <button
                  type="button"
                  onClick={(e) => {
                    e.stopPropagation()
                    renameBranch(b.name)
                  }}
                  aria-label={`Rename ${b.name}`}
                  title="Rename"
                >
                  <Pencil size={16} strokeWidth={1.75} />
                </button>
                {!b.current && (
                  <button
                    type="button"
                    onClick={(e) => {
                      e.stopPropagation()
                      deleteBranch(b.name, false)
                    }}
                    aria-label={`Delete ${b.name}`}
                    title="Delete"
                  >
                    <X size={16} strokeWidth={1.75} />
                  </button>
                )}
              </span>
            </li>
          ))}
        </ul>
      )}

      {remoteOverlaps.length > 0 && (
        <div className="branch-sidebar-overlaps">
          <span className="branch-sidebar-overlaps-label">Remote branches that would conflict</span>
          <ul>
            {remoteOverlaps.map((o) => (
              <li key={o.branch}>
                <OverlapMark overlap={o} />
                <span className="branch-sidebar-overlaps-name">{o.branch}</span>
              </li>
            ))}
          </ul>
        </div>
      )}

      {contextMenu && <ContextMenu state={contextMenu} onClose={() => setContextMenu(null)} />}
      {compareWith && (
        <CompareModal
          repoPath={repoPath}
          initialFromRef={branches?.find((b) => b.current)?.name ?? compareWith}
          initialToRef={compareWith}
          onClose={() => setCompareWith(null)}
        />
      )}
    </div>
  )
}

function OverlapMark({ overlap }: { overlap?: OverlapInfo }) {
  if (!overlap) return null
  const label = `Would conflict with the current branch: ${overlap.files.join(', ')}`
  return (
    <span className="branch-overlap-mark" role="img" aria-label={label} title={label}>
      <TriangleAlert size={12} strokeWidth={1.75} />
    </span>
  )
}

export default BranchSidebar
