import { useState } from 'react'
import { FolderGit2, FolderOpen, Plus, X } from 'lucide-react'
import { RowActionButtons, useRowMenu, type RowAction } from '../../components/controls/RowActions'
import { BranchService, RepositoryService, WorktreeService, type WorktreeInfo } from '@current-client-bindings/app'
import PathText from '../../components/git/PathText'
import { toBranchName } from '../../lib/branchName'
import { errorMessage } from '../../lib/errors'
import { baseName, joinPath } from '../../lib/paths'
import { useAsyncData } from '../../lib/useAsyncData'
import { useDialogs } from '../../lib/useDialogs'
import SidebarSection from '../../components/chrome/SidebarSection'
import type { SidebarSections } from '../../lib/useSidebarSections'
import './WorktreesPanel.scss'

const NEW_BRANCH = ''

interface WorktreesPanelProps {
  repoPath: string
  refreshKey?: unknown
  sections: SidebarSections
  onOpenRepository: (path: string) => void
  onWorktreesChanged?: () => void
}

function parentFolder(path: string): string {
  return path.replace(/[\\/]+[^\\/]+[\\/]*$/, '')
}

function WorktreesPanel({ repoPath, refreshKey, sections, onOpenRepository, onWorktreesChanged }: WorktreesPanelProps) {
  const {
    data: worktrees,
    error: loadError,
    reload,
  } = useAsyncData(() => WorktreeService.List(repoPath), [repoPath], { refreshKey })
  const { data: branches } = useAsyncData(() => BranchService.ListLocal(repoPath), [repoPath], { refreshKey })
  const { confirm } = useDialogs()
  const [actionError, setActionError] = useState<string | null>(null)
  const [showNew, setShowNew] = useState(false)
  const [existingBranch, setExistingBranch] = useState(NEW_BRANCH)
  const [newBranchName, setNewBranchName] = useState('')
  const [parent, setParent] = useState<string | null>(null)
  const [created, setCreated] = useState<string | null>(null)
  const rowMenu = useRowMenu()
  const error = loadError ?? actionError

  const main = worktrees?.find((w) => w.main)
  const branch = existingBranch || toBranchName(newBranchName)
  // Next to the main worktree by default, named after the repository and branch.
  const folder =
    main && branch
      ? joinPath(parent ?? parentFolder(main.path), `${baseName(main.path)}-${branch.replace(/\//g, '-')}`)
      : ''
  const freeBranches = branches?.filter((b) => !b.worktreePath) ?? []

  const changed = () => {
    reload()
    onWorktreesChanged?.()
  }

  const chooseParent = () => {
    RepositoryService.PickDestinationFolder()
      .then((picked) => picked && setParent(picked))
      .catch((err: unknown) => setActionError(errorMessage(err)))
  }

  const add = () => {
    setActionError(null)
    WorktreeService.Add(repoPath, folder, branch, existingBranch === NEW_BRANCH)
      .then(() => {
        setCreated(folder)
        setShowNew(false)
        setExistingBranch(NEW_BRANCH)
        setNewBranchName('')
        setParent(null)
        changed()
      })
      .catch((err: unknown) => setActionError(errorMessage(err)))
  }

  const remove = async (w: WorktreeInfo, force = false) => {
    if (!force) {
      const confirmed = await confirm({
        title: 'Remove worktree',
        message: w.missing
          ? `Forget the worktree at ${w.path}? Its folder is already gone.`
          : `Remove the worktree at ${w.path}? Its folder is deleted; commits and branches stay in the repository.`,
        confirmLabel: 'Remove',
        destructive: true,
      })
      if (!confirmed) return
    }
    setActionError(null)
    WorktreeService.Remove(repoPath, w.path, force)
      .then(changed)
      .catch(async (err: unknown) => {
        if (!force && errorMessage(err) === 'This worktree has changes.') {
          const confirmed = await confirm({
            title: 'Remove worktree',
            message: `${baseName(w.path)} has uncommitted changes, which will be lost. Remove it anyway?`,
            confirmLabel: 'Remove',
            destructive: true,
          })
          if (confirmed) void remove(w, true)
          return
        }
        setActionError(errorMessage(err))
      })
  }

  // The main worktree is this repository; the count is the others.
  const others = worktrees === null ? undefined : Math.max(0, worktrees.length - 1)
  const hasOthers = (others ?? 0) > 0

  const worktreeActions = (w: WorktreeInfo): RowAction[] => [
    ...(!w.current && !w.missing
      ? [
          {
            label: 'Open',
            ariaLabel: `Open worktree ${baseName(w.path)}`,
            icon: FolderOpen,
            run: () => onOpenRepository(w.path),
          },
        ]
      : []),
    ...(!w.main && !w.current && !w.locked
      ? [
          {
            label: 'Remove…',
            ariaLabel: `Remove worktree ${baseName(w.path)}`,
            icon: X,
            run: () => void remove(w),
            destructive: true,
          },
        ]
      : []),
  ]

  return (
    <div className="worktrees-panel">
      <SidebarSection
        label="Worktrees"
        count={others}
        open={sections.isOpen('worktrees', hasOthers)}
        onToggle={() => sections.toggle('worktrees', hasOthers)}
        actions={
          <button
            type="button"
            className="worktrees-panel-add"
            onClick={() => setShowNew((v) => !v)}
            aria-label="Add worktree"
            title="Add worktree"
          >
            <Plus size={18} strokeWidth={1.75} />
          </button>
        }
        pinned={
          showNew && (
            <div className="worktrees-panel-new">
              <select
                aria-label="Branch for the new worktree"
                value={existingBranch}
                onChange={(e) => setExistingBranch(e.target.value)}
              >
                <option value={NEW_BRANCH}>New branch</option>
                {freeBranches.map((b) => (
                  <option key={b.name} value={b.name}>
                    {b.name}
                  </option>
                ))}
              </select>
              {existingBranch === NEW_BRANCH && (
                <input
                  aria-label="New branch name"
                  placeholder="Branch name"
                  value={newBranchName}
                  onChange={(e) => setNewBranchName(e.target.value)}
                  autoFocus
                />
              )}
              {folder && (
                <p className="worktrees-panel-folder" title={folder}>
                  <PathText path={folder} className="worktrees-panel-folder-path" />
                  <button type="button" className="worktrees-panel-choose" onClick={chooseParent}>
                    Choose…
                  </button>
                </p>
              )}
              <button type="button" className="worktrees-panel-create" onClick={add} disabled={!folder}>
                Add worktree
              </button>
            </div>
          )
        }
      >
        {created && (
          <p className="worktrees-panel-created" role="status">
            <span>Added {baseName(created)}</span>
            <button type="button" onClick={() => onOpenRepository(created)}>
              Open
            </button>
          </p>
        )}
        {error && <p className="worktrees-panel-error">{error}</p>}
        {worktrees === null ? (
          <p className="worktrees-panel-hint">Loading worktrees…</p>
        ) : worktrees.length <= 1 ? (
          <p className="worktrees-panel-hint">No other worktrees.</p>
        ) : (
          <ul className="worktrees-panel-list">
            {worktrees.map((w) => (
              <li key={w.path} className="worktree-row" title={w.path} onContextMenu={rowMenu.open(worktreeActions(w))}>
                <FolderGit2 size={14} strokeWidth={1.5} className="worktree-row-icon" />
                <span className="worktree-row-name">{baseName(w.path)}</span>
                <span className="worktree-row-branch">{w.branch || `detached ${w.head.slice(0, 7)}`}</span>
                {w.current && <span className="worktree-row-badge">current</span>}
                {w.locked && <span className="worktree-row-badge">locked</span>}
                {w.missing && <span className="worktree-row-badge worktree-row-badge-missing">missing</span>}
                <RowActionButtons actions={worktreeActions(w)} className="worktree-row-actions" />
              </li>
            ))}
          </ul>
        )}
      </SidebarSection>
      {rowMenu.menu}
    </div>
  )
}

export default WorktreesPanel
