import { useState, type KeyboardEvent, type MouseEvent, useMemo } from 'react'
import { BrushCleaning, GitBranchPlus, Search, X, Table } from 'lucide-react'
import {
  StackService,
  type BranchInfo,
  BranchService,
  type ConflictStateInfo,
  HostService,
  OverlapService,
  RemoteService,
} from '@current-client-bindings/app'
import CompareModal from '../history/CompareModal'
import { useOpenReadThrough } from '../../lib/readThrough'
import ContextMenu, { menuAnchor, type ContextMenuState } from '../../components/controls/ContextMenu'
import { useDeleteRemoteBranch } from '../remotes/useDeleteRemoteBranch'
import { localNameFor } from './branches'
import SidebarSection from '../../components/chrome/SidebarSection'
import type { SidebarSections } from '../../lib/useSidebarSections'
import SetUpstreamDialog from './SetUpstreamDialog'
import { noteMenuLabel, useNotes } from '../notes/notes'
import BranchReviewDialog from './BranchReviewDialog'
import ConflictGridDialog from './ConflictGridDialog'
import RebasePreviewDialog from './RebasePreviewDialog'
import StackSection from './StackSection'
import TidyUpDialog from './TidyUpDialog'
import CleanUpBranchesDialog, { type DeletedBranch } from './CleanUpBranchesDialog'
import { operationNoun } from '../../lib/conflictWords'
import { createChangeFor } from '../../lib/createChange'
import { plural } from '../../lib/plural'
import { useLaneColors } from '../../lib/laneColor'
import { useClockTick } from '../../lib/useClockTick'
import { useSidebarCommand } from '../../lib/sidebarCommands'
import { useAsyncData } from '../../lib/useAsyncData'
import { useBranchStatus } from '../../lib/useBranchStatus'
import { useHostLinks } from '../../lib/useHostLinks'
import { arrowIntoList } from '../../lib/useRovingRows'
import { useBranchChanges } from '../../lib/useBranchChanges'
import { useReferenceCards } from '../../lib/useReferenceCards'
import { branchPage, openWebPage, remoteBranchName } from '../../lib/hostPages'
import BranchRow from './BranchRow'
import NewBranchForm from './NewBranchForm'
import RemoteBranchRow from './RemoteBranchRow'
import { useBranchActions } from './useBranchActions'
import './BranchSidebar.scss'
import { changeNoun } from '../../lib/changeNoun'

interface BranchSidebarProps {
  repoPath: string
  dirty: boolean
  refreshKey: number
  sections: SidebarSections
  // The operation in progress, for the "merging" and "incoming" badges.
  conflict?: ConflictStateInfo | null
  onBranchChanged?: () => void
  onCreateChange?: (branch: string) => void
}

function BranchSidebar({
  repoPath,
  dirty,
  refreshKey,
  sections,
  conflict = null,
  onBranchChanged,
  onCreateChange,
}: BranchSidebarProps) {
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
  // Where each tip has reached, from the last refresh's deployments and Git.
  const { data: tipReach } = useAsyncData(
    () => (tipsKey ? HostService.Reach(repoPath, tipsKey.split(',')).catch(() => null) : null),
    [repoPath, tipsKey],
    { refreshKey },
  )
  const { data: overlapReport } = useAsyncData(() => OverlapService.Predict(repoPath), [repoPath], { refreshKey })
  const overlaps = overlapReport?.overlaps ?? []
  const overlapFor = new Map(overlaps.map((o) => [o.branch, o]))
  const remoteConflicts = overlaps.filter((o) => o.remote).length
  const branchesChanged = () => {
    loadBranches()
    onBranchChanged?.()
  }
  const [showNewBranch, setShowNewBranch] = useState(false)
  const [contextMenu, setContextMenu] = useState<ContextMenuState | null>(null)
  // Refreshes the graph too, which still shows the deleted remote branch.
  const deleteRemote = useDeleteRemoteBranch(repoPath, branchesChanged)
  const readThrough = useOpenReadThrough()
  const [compareWith, setCompareWith] = useState<string | null>(null)
  const notes = useNotes()
  const [cleanUp, setCleanUp] = useState(false)
  const [branchReview, setBranchReview] = useState(false)
  const [conflictGrid, setConflictGrid] = useState(false)
  const [rebasePreview, setRebasePreview] = useState<string | null>(null)
  const [tidyUp, setTidyUp] = useState(false)
  const { data: stackList, reload: reloadStacks } = useAsyncData(() => StackService.List(repoPath), [repoPath], {
    refreshKey,
  })
  const [restack, setRestack] = useState<string | null>(null)
  const stackWith = (name: string) => stackList?.stacks.find((stack) => stack.layers.some((l) => l.branch === name))
  const actions = useBranchActions({
    repoPath,
    branches,
    currentBranch: branchStatus?.current ?? '',
    reloadBranches: loadBranches,
    reloadStacks,
    onBranchesChanged: branchesChanged,
  })
  const error = loadError ?? actions.error
  useSidebarCommand('clean-up', () => setCleanUp(true))
  useSidebarCommand('branch-review', () => setBranchReview(true))
  useSidebarCommand('conflict-grid', () => setConflictGrid(true))
  useSidebarCommand('tidy-up', () => setTidyUp(true))
  useSidebarCommand('new-branch', () => setShowNewBranch(true))
  useSidebarCommand(
    'restack',
    () => {
      const stack = stackWith(branchStatus?.current ?? '')
      if (!stack) {
        actions.setError("The current branch isn't in a stack.")
        return
      }
      if (!sections.isOpen('branches', true)) sections.toggle('branches', true)
      if (!sections.isOpen('branches-stacks', true)) sections.toggle('branches-stacks', true)
      setRestack(stack.root)
    },
    stackList !== null && !!branchStatus,
  )
  const [upstreamFor, setUpstreamFor] = useState<BranchInfo | null>(null)

  const cleanedUp = (branches: DeletedBranch[]) => {
    setCleanUp(false)
    setBranchReview(false)
    actions.setDeleted(branches.length > 0 ? branches : null)
    branchesChanged()
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
        { label: 'Check out', onClick: () => actions.checkout(b.name), disabled: b.current || !!blocked },
        {
          separatorBefore: true,
          label: 'Merge into current',
          onClick: () => actions.merge(b.name),
          disabled: b.current,
        },
        {
          label: 'Merge into current (no fast-forward)',
          onClick: () => actions.merge(b.name, 'no-ff'),
          disabled: b.current,
        },
        { label: 'Squash into current', onClick: () => actions.merge(b.name, 'squash'), disabled: b.current },
        ...(b.current && defaultBranch && defaultBranch !== b.name
          ? [{ label: `Merge ${defaultBranch} into current`, onClick: () => actions.mergeDefault(defaultBranch) }]
          : []),
        ...(b.current ? [{ label: 'Tidy up history…', onClick: () => setTidyUp(true) }] : []),
        { label: 'Rebase current onto…', onClick: () => setRebasePreview(b.name), disabled: b.current },
        {
          separatorBefore: true,
          label: 'Read through',
          onClick: () => readThrough({ ref: `refs/heads/${b.name}` }),
        },
        {
          label: 'Compare with current',
          onClick: () => setCompareWith(b.name),
          disabled: b.current,
        },
        { label: 'New stacked branch…', onClick: () => void actions.newStacked(b.name) },
        {
          label: 'Stack current on this',
          onClick: () => actions.stackCurrentOn(b.name),
          disabled: b.current || !branchStatus?.current || branchStatus.current === defaultBranch,
        },
        ...(stackWith(b.name) ? [{ label: 'Remove from stack', onClick: () => actions.removeFromStack(b.name) }] : []),
        ...(hostLinks?.tree ? [{ label: 'Open on host', onClick: () => openWebPage(branchHostPage(b)) }] : []),
        ...(hostLinks?.tree && b.upstream && !changes[b.upstream]?.state
          ? [
              {
                label: `Create ${changeNoun(hostLinks.style)}…`,
                onClick: () => void createChange(b.name),
              },
            ]
          : []),
        {
          label: noteMenuLabel(notes, 'branch', b.name),
          onClick: () => notes.edit({ kind: 'branch', target: b.name }),
        },
        { separatorBefore: true, label: 'Rename…', onClick: () => actions.rename(b.name) },
        { label: 'Set upstream…', onClick: () => setUpstreamFor(b) },
        ...(b.upstream ? [{ label: 'Unset upstream', onClick: () => actions.unsetUpstream(b.name) }] : []),
        {
          separatorBefore: true,
          label: 'Delete',
          onClick: () => actions.remove(b.name, false),
          destructive: true,
          disabled: b.current,
        },
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
        { label: 'Check out', onClick: () => actions.checkout(localNameFor(name, remoteNames)), disabled: !!blocked },
        { label: 'Read through', onClick: () => readThrough({ ref: `refs/remotes/${name}` }) },
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
    <BranchRow
      key={b.name}
      repoPath={repoPath}
      branch={b}
      color={branchColor(b.name, b.current)}
      dirty={dirty}
      blocked={blocked}
      conflict={conflict}
      hostLinks={hostLinks}
      referenceCards={referenceCards}
      change={b.upstream ? changes[b.upstream] : undefined}
      ciMark={ciMarks?.[b.tip]}
      reached={(tipReach?.[b.tip] ?? []).filter((r) => r.state === 'in').map((r) => r.environment)}
      overlap={overlapFor.get(b.name)}
      onCheckout={() => actions.checkout(b.name)}
      onMenu={branchContextMenu(b)}
      onRename={() => void actions.rename(b.name)}
      onDelete={() => actions.remove(b.name, false)}
    />
  )

  const renderRemoteRow = (name: string) => (
    <RemoteBranchRow
      key={name}
      name={name}
      blocked={blocked}
      incoming={!!conflict && name === conflict.theirs}
      local={localNames.has(localNameFor(name, remoteNames))}
      overlap={overlapFor.get(name)}
      onCheckout={() => actions.checkout(localNameFor(name, remoteNames))}
      onMenu={remoteMenu(name)}
    />
  )

  // Git refuses a checkout mid-operation, so the rows say so instead of failing.
  const blocked = conflict ? `Finish the ${operationNoun(conflict.operation)} first` : null
  const localOpen = sections.isOpen('branches-local', true)
  const { deleted, squashed } = actions
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
              title={blocked ?? 'New branch'}
              disabled={!!blocked}
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
            <button
              type="button"
              className="branch-sidebar-add"
              onClick={() => setBranchReview(true)}
              aria-label="Branch review"
              title="Branch review"
            >
              <Table size={18} strokeWidth={1.75} />
            </button>
          </>
        }
        onPinnedEscape={() => setShowNewBranch(false)}
        pinned={
          showNewBranch && (
            <NewBranchForm
              repoPath={repoPath}
              onCreate={(kind, name) => actions.create(kind, name, () => setShowNewBranch(false))}
            />
          )
        }
      >
        {error && (
          <p className="branch-sidebar-error" role="alert">
            <span>{error}</span>
            {actions.error && (
              <button type="button" aria-label="Dismiss" title="Dismiss" onClick={() => actions.setError(null)}>
                <X size={14} strokeWidth={1.75} />
              </button>
            )}
          </p>
        )}
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
            <button type="button" onClick={() => actions.restore(deleted)}>
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
                onKeyDown={(e) => arrowIntoList(e, '.branch-row')}
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
            {stackList && stackList.stacks.length > 0 && (
              <div className="branch-sidebar-group">
                <SidebarSection
                  label="Stacks"
                  count={stackList.stacks.length}
                  open={sections.isOpen('branches-stacks', true)}
                  onToggle={() => sections.toggle('branches-stacks', true)}
                >
                  <StackSection
                    repoPath={repoPath}
                    list={stackList}
                    current={branchStatus?.current ?? null}
                    inOperation={!!conflict}
                    restack={restack}
                    onRestackChange={setRestack}
                    onCheckout={(name) => !blocked && actions.checkout(name)}
                    onRowMenu={(name) => {
                      const branch = branches.find((b) => b.name === name)
                      return branch && branchContextMenu(branch)
                    }}
                    onChanged={actions.stacksChanged}
                  />
                </SidebarSection>
              </div>
            )}
            <div className="branch-sidebar-group">
              <SidebarSection
                label="Local branches"
                count={localBranches.length}
                open={localOpen}
                onToggle={() => sections.toggle('branches-local', true)}
              >
                <ul className="branch-sidebar-list">{localBranches.map(renderLocalRow)}</ul>
              </SidebarSection>
            </div>
            <div className="branch-sidebar-group">
              <SidebarSection
                label="Remote branches"
                count={remoteBranches === null ? undefined : remoteList.length}
                open={remoteOpen}
                onToggle={() => sections.toggle('branches-remote', false)}
                actions={
                  remoteConflicts > 0 && (
                    <span
                      className="branch-conflicts-badge"
                      title={`${plural(remoteConflicts, 'remote branch', 'remote branches')} would conflict with the current branch`}
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
      {tidyUp && (
        <TidyUpDialog
          repoPath={repoPath}
          onDone={() => {
            setTidyUp(false)
            branchesChanged()
          }}
          onClose={() => setTidyUp(false)}
        />
      )}
      {rebasePreview && (
        <RebasePreviewDialog
          repoPath={repoPath}
          onto={rebasePreview}
          onRebase={() => {
            setRebasePreview(null)
            actions.rebaseOnto(rebasePreview)
          }}
          onClose={() => setRebasePreview(null)}
        />
      )}
      {branchReview && (
        <BranchReviewDialog
          repoPath={repoPath}
          onDeleted={cleanedUp}
          onConflictGrid={() => setConflictGrid(true)}
          onClose={() => setBranchReview(false)}
        />
      )}
      {conflictGrid && (
        <ConflictGridDialog repoPath={repoPath} refreshKey={refreshKey} onClose={() => setConflictGrid(false)} />
      )}
      {upstreamFor && (
        <SetUpstreamDialog
          repoPath={repoPath}
          branch={upstreamFor.name}
          upstream={upstreamFor.upstream}
          onClose={() => setUpstreamFor(null)}
          onSet={() => {
            setUpstreamFor(null)
            branchesChanged()
          }}
        />
      )}
      {contextMenu && <ContextMenu state={contextMenu} onClose={() => setContextMenu(null)} />}
      {compareWith && (
        <CompareModal
          repoPath={repoPath}
          initialFromRef={current?.name ?? compareWith}
          initialToRef={compareWith}
          onClose={() => setCompareWith(null)}
        />
      )}
    </div>
  )
}

export default BranchSidebar
