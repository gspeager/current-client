import { useState, type KeyboardEvent, type MouseEvent, useMemo } from 'react'
import { BrushCleaning, Cloud, Ellipsis, GitBranchPlus, Pencil, Search, X } from 'lucide-react'
import {
  type BranchInfo,
  BranchService,
  GitFlowService,
  HostService,
  type OverlapInfo,
  OverlapService,
  PlatformService,
  RemoteService,
} from '@current-client-bindings/app'
import CompareModal from '../history/CompareModal'
import ContextMenu, { isMenuKey, menuAnchor, type ContextMenuState } from '../../components/controls/ContextMenu'
import { useDeleteRemoteBranch } from '../remotes/useDeleteRemoteBranch'
import { localNameFor } from './branches'
import SidebarSection from '../../components/chrome/SidebarSection'
import type { SidebarSections } from '../../lib/useSidebarSections'
import SetUpstreamDialog from './SetUpstreamDialog'
import CleanUpBranchesDialog, { type DeletedBranch } from './CleanUpBranchesDialog'
import { toBranchName } from '../../lib/branchName'
import { createChangeFor } from '../../lib/createChange'
import { errorMessage } from '../../lib/errors'
import { plural } from '../../lib/plural'
import { useLaneColors } from '../../lib/laneColor'
import { relativeTime } from '../../lib/relativeTime'
import { useClockTick } from '../../lib/useClockTick'
import { useAsyncData } from '../../lib/useAsyncData'
import { useBranchStatus } from '../../lib/useBranchStatus'
import { useDialogs } from '../../lib/useDialogs'
import { useHostLinks } from '../../lib/useHostLinks'
import BranchPill from '../../components/git/BranchPill'
import ChangeMarker from '../../components/git/ChangeMarker'
import CIMark from '../../components/git/CIMark'
import { useBranchChanges } from '../../lib/useBranchChanges'
import { useReferenceCards } from '../../lib/useReferenceCards'
import { branchPage, remoteBranchName } from '../../lib/hostPages'
import './BranchSidebar.scss'

interface BranchSidebarProps {
  repoPath: string
  dirty: boolean
  refreshKey: number
  sections: SidebarSections
  onBranchChanged?: () => void
  onCreateChange?: (branch: string) => void
}

type NewBranchKind = 'branch' | 'feature' | 'release' | 'hotfix'

function BranchSidebar({ repoPath, dirty, refreshKey, sections, onBranchChanged, onCreateChange }: BranchSidebarProps) {
  useClockTick()
  const { branchColor } = useLaneColors()
  const {
    data: branches,
    error: loadError,
    reload: loadBranches,
  } = useAsyncData(() => BranchService.ListLocal(repoPath), [repoPath], { refreshKey })
  const { data: defaultBranch } = useAsyncData(() => BranchService.DefaultBranch(repoPath), [repoPath], {
    refreshKey,
  })
  const { data: remoteBranches, error: remoteError } = useAsyncData(
    () => BranchService.ListRemote(repoPath),
    [repoPath],
    {
      refreshKey,
    },
  )
  const { data: remotes } = useAsyncData(() => RemoteService.List(repoPath), [repoPath], { refreshKey })
  const remoteNames = useMemo(() => (remotes ?? []).map((r) => r.name), [remotes])
  const [filter, setFilter] = useState('')
  const branchStatus = useBranchStatus(repoPath, refreshKey)
  const hostLinks = useHostLinks(repoPath)
  const changes = useBranchChanges(repoPath, refreshKey)
  const issueTexts = useMemo(() => (branches ?? []).map((b) => b.name.replace(/^(\d+)-.*/, '#$1')), [branches])
  const referenceCards = useReferenceCards(repoPath, issueTexts, changes)
  // Branch tips, joined so the lookup only reruns when a tip moves.
  const tipsKey = (branches ?? [])
    .map((b) => b.tip)
    .filter(Boolean)
    .join(',')
  const { data: ciMarks } = useAsyncData(
    () => (tipsKey ? HostService.CIStatus(repoPath, tipsKey.split(',')) : null),
    [repoPath, tipsKey],
    { refreshKey },
  )
  const { data: overlapReport } = useAsyncData(() => OverlapService.Predict(repoPath), [repoPath], { refreshKey })
  const overlaps = overlapReport?.overlaps ?? []
  const overlapFor = new Map(overlaps.map((o) => [o.branch, o]))
  const remoteConflicts = overlaps.filter((o) => o.remote).length
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
  const [deleted, setDeleted] = useState<DeletedBranch[] | null>(null)
  const [cleanUp, setCleanUp] = useState(false)
  const [squashed, setSquashed] = useState<string | null>(null)
  const [upstreamFor, setUpstreamFor] = useState<BranchInfo | null>(null)

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

  // mode is '' (fast-forward when possible), 'no-ff' or 'squash'.
  const mergeBranch = (name: string, mode = '') => {
    setActionError(null)
    setSquashed(null)
    BranchService.MergeBranch(repoPath, name, mode)
      .then(() => {
        if (mode === 'squash') setSquashed(name)
      })
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
        message: `${name} is ${plural(target.behind, 'commit')} behind ${target.upstream}. Pull ${name} first to include the latest work, or merge it as it is.`,
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
        setDeleted([{ name, tip }])
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

  const upstreamChanged = () => {
    loadBranches()
    onBranchChanged?.()
  }

  const unsetUpstream = (name: string) => {
    setActionError(null)
    BranchService.UnsetUpstream(repoPath, name)
      .then(upstreamChanged)
      .catch((err: unknown) => setActionError(errorMessage(err)))
  }

  const restoreDeleted = (branches: DeletedBranch[]) => {
    setActionError(null)
    setDeleted(null)
    Promise.all(
      branches.map((branch) =>
        BranchService.CreateBranchAt(repoPath, branch.name, branch.tip).catch((err: unknown) => {
          throw new Error(`Could not restore ${branch.name}: ${errorMessage(err)}`)
        }),
      ),
    )
      .catch((err: unknown) => setActionError(errorMessage(err)))
      .finally(() => {
        loadBranches()
        onBranchChanged?.()
      })
  }

  const cleanedUp = (branches: DeletedBranch[]) => {
    setCleanUp(false)
    setDeleted(branches.length > 0 ? branches : null)
    loadBranches()
    onBranchChanged?.()
  }

  const createChange = (name: string) => createChangeFor(repoPath, name, onCreateChange)

  // The branch's merge request when one is known, else its tree on the host.
  const branchHostPage = (b: BranchInfo): string => {
    const change = b.upstream ? changes[b.upstream] : undefined
    if (change?.webUrl) return change.webUrl
    const name = b.upstream ? remoteBranchName(b.upstream, [b.upstream.split('/')[0]]) : b.name
    return branchPage(hostLinks!, name)
  }

  const branchContextMenu = (b: BranchInfo) => (e: MouseEvent | KeyboardEvent) => {
    e.preventDefault()
    e.stopPropagation()
    setContextMenu({
      ...menuAnchor(e),
      items: [
        { label: 'Check out', onClick: () => checkoutBranch(b.name), disabled: b.current },
        { label: 'Merge into current', onClick: () => mergeBranch(b.name), disabled: b.current },
        {
          label: 'Merge into current (no fast-forward)',
          onClick: () => mergeBranch(b.name, 'no-ff'),
          disabled: b.current,
        },
        { label: 'Squash into current', onClick: () => mergeBranch(b.name, 'squash'), disabled: b.current },
        ...(b.current && defaultBranch && defaultBranch !== b.name
          ? [{ label: `Merge ${defaultBranch} into current`, onClick: () => mergeDefaultBranch(defaultBranch) }]
          : []),
        { label: 'Rebase current onto', onClick: () => rebaseOnto(b.name), disabled: b.current },
        { label: 'Compare with current', onClick: () => setCompareWith(b.name), disabled: b.current },
        ...(hostLinks?.tree
          ? [{ label: 'Open on host', onClick: () => void PlatformService.OpenWebPage(branchHostPage(b)) }]
          : []),
        ...(hostLinks?.tree && b.upstream && !changes[b.upstream]?.state
          ? [
              {
                label: `Create ${hostLinks.style === 'gitlab' ? 'merge request' : 'pull request'}…`,
                onClick: () => void createChange(b.name),
              },
            ]
          : []),
        { label: 'Rename…', onClick: () => renameBranch(b.name) },
        { label: 'Set upstream…', onClick: () => setUpstreamFor(b) },
        ...(b.upstream ? [{ label: 'Unset upstream', onClick: () => unsetUpstream(b.name) }] : []),
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

  const remoteMenu = (name: string) => (e: MouseEvent | KeyboardEvent) => {
    e.preventDefault()
    e.stopPropagation()
    setContextMenu({
      ...menuAnchor(e),
      items: [
        { label: 'Check out', onClick: () => checkoutBranch(localNameFor(name, remoteNames)) },
        {
          label: 'Delete on remote…',
          onClick: () => void deleteRemote.deleteRemoteBranch(name),
          destructive: true,
          disabled: deleteRemote.running,
        },
      ],
    })
  }

  const query = filter.trim().toLowerCase()
  const current = branches?.find((b) => b.current) ?? null
  const localBranches = (branches ?? []).filter((b) => !b.current && b.name.toLowerCase().includes(query))
  const localNames = new Set((branches ?? []).map((b) => b.name))
  const remoteList = (remoteBranches ?? []).filter((name) => name.toLowerCase().includes(query))

  const renderLocalRow = (b: BranchInfo) => (
    <li
      key={b.name}
      className={b.current ? 'branch-row branch-row-current' : 'branch-row'}
      onClick={() => !b.current && checkoutBranch(b.name)}
      onContextMenu={branchContextMenu(b)}
      role="button"
      tabIndex={0}
      onKeyDown={(e) => {
        if (isMenuKey(e)) {
          branchContextMenu(b)(e)
          return
        }
        if (e.target !== e.currentTarget) return
        if ((e.key === 'Enter' || e.key === ' ') && !b.current) {
          e.preventDefault()
          checkoutBranch(b.name)
        }
      }}
    >
      <BranchPill
        name={b.name}
        color={branchColor(b.name, b.current)}
        current={b.current}
        dirty={dirty}
        hostLinks={hostLinks}
        referenceCards={referenceCards}
      />
      {b.current && <span className="branch-row-head">HEAD</span>}
      <ChangeMarker change={b.upstream ? changes[b.upstream] : undefined} />
      {!(b.upstream && changes[b.upstream]) && <CIMark repoPath={repoPath} mark={ciMarks?.[b.tip]} />}
      <ConflictsBadge overlap={overlapFor.get(b.name)} />
      {!b.current && b.worktreePath && (
        <span className="branch-row-worktree" title={`Checked out in the worktree at ${b.worktreePath}`}>
          worktree
        </span>
      )}
      <span className="branch-row-meta">
        {b.ahead > 0 && <span className="branch-row-ahead">↑{b.ahead}</span>}
        {b.behind > 0 && <span className="branch-row-behind">↓{b.behind}</span>}
        {b.lastCommitDate && <span className="branch-row-time">{relativeTime(new Date(b.lastCommitDate))}</span>}
      </span>
      <span className="branch-row-actions">
        <button
          type="button"
          onClick={branchContextMenu(b)}
          aria-label={`More actions for ${b.name}`}
          title="More actions"
        >
          <Ellipsis size={16} strokeWidth={1.75} />
        </button>
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
  )

  const renderRemoteRow = (name: string) => (
    <li
      key={name}
      className="branch-row branch-row-remote"
      onClick={() => checkoutBranch(localNameFor(name, remoteNames))}
      onContextMenu={remoteMenu(name)}
      role="button"
      tabIndex={0}
      onKeyDown={(e) => {
        if (isMenuKey(e)) {
          remoteMenu(name)(e)
          return
        }
        if (e.key === 'Enter' || e.key === ' ') {
          e.preventDefault()
          checkoutBranch(localNameFor(name, remoteNames))
        }
      }}
    >
      <Cloud size={14} strokeWidth={1.5} className="branch-row-remote-icon" />
      <span className="branch-row-remote-name">{name}</span>
      <ConflictsBadge overlap={overlapFor.get(name)} />
      {!localNames.has(localNameFor(name, remoteNames)) && <span className="branch-row-meta">not local</span>}
    </li>
  )

  const localOpen = sections.isOpen('branches-local', true)
  const remoteOpen = sections.isOpen('branches-remote', false)

  return (
    <div className="branch-sidebar">
      <SidebarSection
        label="Branches"
        count={branches?.length}
        open={sections.isOpen('branches', true)}
        onToggle={() => sections.toggle('branches', true)}
        actions={
          <>
            <button
              type="button"
              className="branch-sidebar-add"
              onClick={() => {
                setShowNewBranch((v) => !v)
                sections.open('branches')
              }}
              aria-label="New branch"
              title="New branch"
            >
              <GitBranchPlus size={18} strokeWidth={1.75} />
            </button>
            <button
              type="button"
              className="branch-sidebar-add"
              onClick={() => setCleanUp(true)}
              aria-label="Clean up merged branches"
              title="Clean up merged branches"
            >
              <BrushCleaning size={18} strokeWidth={1.75} />
            </button>
          </>
        }
        pinned={
          showNewBranch && (
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
          )
        }
      >
        {error && <p className="branch-sidebar-error">{error}</p>}
        {deleteRemote.error && <p className="branch-sidebar-error">Could not delete on remote: {deleteRemote.error}</p>}
        {deleted && (
          <p className="branch-sidebar-deleted" role="status">
            <span>
              Deleted{' '}
              {deleted.length === 1 ? (
                <span className="branch-sidebar-deleted-name">{deleted[0].name}</span>
              ) : (
                `${deleted.length} branches`
              )}
            </span>
            <button type="button" onClick={() => restoreDeleted(deleted)}>
              Undo
            </button>
          </p>
        )}
        {squashed && (
          <p className="branch-sidebar-notice" role="status">
            Changes from {squashed} are staged. Commit them to finish the squash.
          </p>
        )}
        {branches === null ? (
          <p className="branch-sidebar-hint">Loading branches…</p>
        ) : branches.length === 0 ? (
          <p className="branch-sidebar-hint">
            {branchStatus?.current ? `No commits on ${branchStatus.current} yet.` : 'No branches.'}
          </p>
        ) : (
          <>
            <div className="branch-sidebar-filter">
              <Search size={14} strokeWidth={1.5} />
              <input
                aria-label="Filter branches"
                placeholder="Filter branches…"
                value={filter}
                onChange={(e) => setFilter(e.target.value)}
              />
            </div>
            {current && (
              <div className="branch-sidebar-group">
                <span className="branch-sidebar-group-label">Checked out</span>
                <ul className="branch-sidebar-list">{renderLocalRow(current)}</ul>
              </div>
            )}
            <div className="branch-sidebar-group">
              <SidebarSection
                label="Local"
                count={localBranches.length}
                open={localOpen}
                onToggle={() => sections.toggle('branches-local', true)}
              >
                <ul className="branch-sidebar-list">{localBranches.map(renderLocalRow)}</ul>
              </SidebarSection>
            </div>
            <div className="branch-sidebar-group">
              <SidebarSection
                label="Remote"
                count={remoteBranches?.length}
                open={remoteOpen}
                onToggle={() => sections.toggle('branches-remote', false)}
                actions={
                  remoteConflicts > 0 && (
                    <span
                      className="branch-conflicts-badge"
                      title={`${plural(remoteConflicts, 'remote branch')} would conflict with the current branch`}
                    >
                      {remoteConflicts} conflicts
                    </span>
                  )
                }
              >
                {remoteError && <p className="branch-sidebar-error">{remoteError}</p>}
                <ul className="branch-sidebar-list">{remoteList.map(renderRemoteRow)}</ul>
              </SidebarSection>
            </div>
          </>
        )}
      </SidebarSection>

      {cleanUp && <CleanUpBranchesDialog repoPath={repoPath} onDeleted={cleanedUp} onClose={() => setCleanUp(false)} />}
      {upstreamFor && (
        <SetUpstreamDialog
          repoPath={repoPath}
          branch={upstreamFor.name}
          upstream={upstreamFor.upstream}
          onClose={() => setUpstreamFor(null)}
          onSet={() => {
            setUpstreamFor(null)
            upstreamChanged()
          }}
        />
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

function ConflictsBadge({ overlap }: { overlap?: OverlapInfo }) {
  if (!overlap) return null
  return (
    <span
      className="branch-conflicts-badge"
      title={`Would conflict with the current branch: ${overlap.files.join(', ')}`}
    >
      conflicts
    </span>
  )
}

export default BranchSidebar
