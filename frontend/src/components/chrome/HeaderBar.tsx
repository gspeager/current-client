import { useEffect, useRef, useState, type DragEvent, type ReactNode } from 'react'
import { System } from '@wailsio/runtime'
import { ArrowDown, ArrowUp, ChevronDown, Inbox, Plus, RefreshCw, Settings, type LucideIcon, X } from 'lucide-react'
import { useOnClickOutside } from 'usehooks-ts'
import BranchPill from '../git/BranchPill'
import ContextMenu, { type ContextMenuItem, type ContextMenuState } from '../controls/ContextMenu'
import FileHistoryPanel from '../../features/history/FileHistoryPanel'
import IdentityBadge from '../git/IdentityBadge'
import Kbd from '../controls/Kbd'
import { useBranchStatus } from '../../lib/useBranchStatus'
import { useCurrentUser } from '../../lib/useCurrentUser'
import { useFetchAll } from '../../features/remotes/useFetchAll'
import { useLaneColors } from '../../lib/laneColor'
import { operationBadge, operationNoun } from '../../lib/conflictWords'
import { baseName } from '../../lib/paths'
import { relativeTime } from '../../lib/relativeTime'
import { useClockTick } from '../../lib/useClockTick'
import type { useRepositoryLifecycle } from '../../features/repositories/useRepositoryLifecycle'
import BreadcrumbPanel, { type PanelView } from './BreadcrumbPanel'
import QuickSwitchPalette from './QuickSwitchPalette'
import type { WorkspaceAction } from './useWorkspaceActions'
import RepoPickerModal from '../../features/repositories/RepoPickerModal'
import RepoTabChip from '../../features/repositories/RepoTabChip'
import SearchModal from '../../features/search/SearchModal'
import './HeaderBar.scss'
import { useWindowKeydown } from '../../lib/useWindowKeydown'

interface HeaderBarProps {
  repoPath: string
  repoVersion: number
  lastFetchedAt: Date | null
  pruneOnFetch: boolean
  repo: ReturnType<typeof useRepositoryLifecycle>
  dirty: boolean
  // The Inbox, a view across every open repository, beside the tabs.
  inboxOpen?: boolean
  onToggleInbox?: () => void
  workspaceActions: WorkspaceAction[]
  onOpenSettings: () => void
  onPull: () => void
  onPullMerge: () => void
  onPullRebase: () => void
  onPush: () => void
  onForcePush: () => void
  onPushTags: () => void
  onPushSetUpstream: () => void
  onBranchChanged?: () => void
  onFetched?: () => void
  onOpenCommit: (sha: string, position: number) => void
  syncing: boolean
  // "merge", "rebase", "cherry-pick", "revert" or "am" while one is in progress.
  operation?: string | null
  // Settings' default repository location, shown in the Open repository dialog.
  defaultRepoLocation?: string
  // Supplied by an app embedding Current Client, such as a switch between its views.
  accessory?: ReactNode
}

function HeaderBar({
  repoPath,
  repoVersion,
  lastFetchedAt,
  pruneOnFetch,
  repo,
  dirty,
  inboxOpen = false,
  onToggleInbox,
  workspaceActions,
  onOpenSettings,
  onPull,
  onPullMerge,
  onPullRebase,
  onPush,
  onForcePush,
  onPushTags,
  onPushSetUpstream,
  onBranchChanged,
  onFetched,
  onOpenCommit,
  syncing,
  operation = null,
  defaultRepoLocation = '',
  accessory,
}: HeaderBarProps) {
  useClockTick()
  const { branchColor } = useLaneColors()
  const currentUser = useCurrentUser(repoPath)
  const branchStatus = useBranchStatus(repoPath, repoVersion)
  const fetchAllOp = useFetchAll(repoPath, onFetched, pruneOnFetch)
  const [branchSwitcherOpen, setBranchSwitcherOpen] = useState(false)
  const [panelView, setPanelView] = useState<PanelView>('branch')
  const [quickSwitchOpen, setQuickSwitchOpen] = useState(false)
  const [fileHistoryPath, setFileHistoryPath] = useState<string | null>(null)
  const [searchOpen, setSearchOpen] = useState(false)
  const [dragPath, setDragPath] = useState<string | null>(null)
  const [pickerOpen, setPickerOpen] = useState(false)
  const [syncMenu, setSyncMenu] = useState<ContextMenuState | null>(null)
  const [panelPosition, setPanelPosition] = useState<{ top: number; left: number } | null>(null)
  const branchSwitcherRef = useRef<HTMLDetailsElement>(null)

  useOnClickOutside(branchSwitcherRef, () => setBranchSwitcherOpen(false))

  // position: fixed escapes the scrolling tab strip's overflow clipping.
  useEffect(() => {
    if (!branchSwitcherOpen) {
      setPanelPosition(null)
      return
    }
    const rect = branchSwitcherRef.current?.getBoundingClientRect()
    if (rect) setPanelPosition({ top: rect.bottom + 4, left: rect.left })
  }, [branchSwitcherOpen])

  // preventDefault stops <summary> from closing an open panel when switching halves.
  const openPanel = (view: PanelView) => (e: React.MouseEvent) => {
    e.preventDefault()
    setPanelView(view)
    setBranchSwitcherOpen(true)
  }

  const onDropTab = (targetPath: string) => {
    if (!dragPath || dragPath === targetPath) {
      setDragPath(null)
      return
    }
    const withoutDragged = repo.tabs.filter((p) => p !== dragPath)
    const targetIndex = withoutDragged.indexOf(targetPath)
    const next = [...withoutDragged.slice(0, targetIndex), dragPath, ...withoutDragged.slice(targetIndex)]
    repo.reorderTabs(next)
    setDragPath(null)
  }

  const fetchedLabel = fetchAllOp.running
    ? 'Fetching…'
    : lastFetchedAt
      ? `Fetched ${relativeTime(lastFetchedAt)}`
      : 'Not fetched yet'

  // A sync button with a chevron beside it that drops the rest of its choices.
  // `due` colors it while there is something to pull or push.
  const syncButton = (
    id: 'fetch' | 'pull' | 'push',
    label: string,
    Icon: LucideIcon,
    title: string,
    onClick: (e: React.MouseEvent) => void,
    items: ContextMenuItem[],
    disabled = false,
    due = false,
  ) => {
    const name = id[0].toUpperCase() + id.slice(1)
    const blocked = operation ? `Finish the ${operationNoun(operation)} first` : null
    const className = due ? 'header-bar-sync header-bar-sync-due' : 'header-bar-sync'
    return (
      <span className={`header-bar-sync-group header-bar-sync-${id}`}>
        <button
          type="button"
          className={className}
          onClick={disabled || blocked ? undefined : onClick}
          disabled={!!blocked}
          title={blocked ?? title}
        >
          <Icon size={12} strokeWidth={1.5} />
          {label}
        </button>
        <button
          type="button"
          className={`${className} header-bar-sync-more`}
          aria-label={`${name} options`}
          title={blocked ?? `${name} options`}
          disabled={disabled || !!blocked}
          onClick={(e) => {
            const rect = e.currentTarget.getBoundingClientRect()
            setSyncMenu({ x: rect.right, y: rect.bottom + 4, items, align: 'end' })
          }}
        >
          <ChevronDown size={12} strokeWidth={1.5} />
        </button>
      </span>
    )
  }

  useWindowKeydown((e) => {
    if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === 'k') {
      e.preventDefault()
      setQuickSwitchOpen(true)
    }
    if ((e.metaKey || e.ctrlKey) && e.shiftKey && e.key.toLowerCase() === 'f') {
      e.preventDefault()
      setSearchOpen(true)
    }
  })

  return (
    <header className="header-bar">
      <div className="header-bar-tabs" aria-label="Open repositories">
        {repo.tabs.map((path) =>
          path === repoPath ? (
            <details
              key={path}
              ref={branchSwitcherRef}
              className="header-bar-switcher"
              open={branchSwitcherOpen}
              onToggle={(e) => setBranchSwitcherOpen(e.currentTarget.open)}
              draggable
              onDragStart={() => setDragPath(path)}
              onDragOver={(e: DragEvent) => e.preventDefault()}
              onDrop={() => onDropTab(path)}
            >
              <summary className="header-bar-repo">
                <span className="header-bar-repo-name" onClick={openPanel('repo')}>
                  {baseName(repoPath)}
                </span>
                {branchStatus && (
                  <>
                    <span className="header-bar-repo-sep">|</span>
                    <span className="header-bar-branch-target" onClick={openPanel('branch')}>
                      <BranchPill
                        name={branchStatus.detachedAt ? `HEAD ${branchStatus.detachedAt}` : branchStatus.current}
                        color={branchColor(branchStatus.current, true)}
                        current
                        bare
                        dirty={dirty}
                      />
                    </span>
                  </>
                )}
                {operation && <span className="header-bar-mode">{operationBadge(operation)}</span>}
                <ChevronDown size={12} strokeWidth={1.5} className="header-bar-repo-chevron" />
                {repo.tabs.length > 1 && (
                  <button
                    type="button"
                    className="header-bar-repo-close"
                    aria-label={`Close ${baseName(repoPath)}`}
                    onClick={(e) => {
                      e.preventDefault()
                      e.stopPropagation()
                      repo.closeTab(path)
                    }}
                  >
                    <X size={11} strokeWidth={1.75} />
                  </button>
                )}
              </summary>
              {branchSwitcherOpen && panelPosition && (
                <div
                  className="header-bar-breadcrumb-anchor"
                  style={{ top: panelPosition.top, left: panelPosition.left }}
                >
                  <BreadcrumbPanel
                    repoPath={repoPath}
                    repo={repo}
                    view={panelView}
                    onViewChange={setPanelView}
                    onBranchChanged={onBranchChanged}
                    onFetched={onFetched}
                    pruneOnFetch={pruneOnFetch}
                    onOpenAnother={() => {
                      setBranchSwitcherOpen(false)
                      setPickerOpen(true)
                    }}
                  />
                </div>
              )}
            </details>
          ) : (
            <RepoTabChip
              key={path}
              path={path}
              summary={repo.tabSummaries[path]}
              onFocus={() => repo.openRecent(path)}
              onClose={() => repo.closeTab(path)}
              onDragStart={() => setDragPath(path)}
              onDragOver={(e) => e.preventDefault()}
              onDrop={() => onDropTab(path)}
            />
          ),
        )}
        <button
          type="button"
          className="header-bar-icon-button"
          aria-label="Open another repository"
          title="Open another repository"
          onClick={() => setPickerOpen(true)}
        >
          <Plus size={16} strokeWidth={1.75} />
        </button>
      </div>

      <div className="header-bar-sync-cluster">
        {syncButton(
          'fetch',
          fetchAllOp.running ? 'Cancel' : 'Fetch',
          RefreshCw,
          fetchAllOp.running ? 'Cancel fetch' : `${fetchedLabel}. Fetch${pruneOnFetch ? ' and prune' : ''}`,
          fetchAllOp.running ? fetchAllOp.cancel : () => fetchAllOp.fetchPrimary(),
          [
            { label: 'Fetch and prune', onClick: fetchAllOp.fetchPrune },
            { label: 'Fetch all remotes', onClick: fetchAllOp.fetchAll },
          ],
          syncing,
        )}
        {syncButton(
          'pull',
          branchStatus && branchStatus.behind > 0 ? `Pull ${branchStatus.behind}` : 'Pull',
          ArrowDown,
          'Pull (shift-click to rebase)',
          (e?: React.MouseEvent) => (e?.shiftKey ? onPullRebase() : onPull()),
          [
            { label: 'Pull (merge)', onClick: onPullMerge },
            { label: 'Pull (rebase)', onClick: onPullRebase },
          ],
          syncing,
          !!branchStatus && branchStatus.behind > 0,
        )}
        {syncButton(
          'push',
          branchStatus && branchStatus.ahead > 0 ? `Push ${branchStatus.ahead}` : 'Push',
          ArrowUp,
          'Push (shift-click to force-push)',
          (e?: React.MouseEvent) => (e?.shiftKey ? onForcePush() : onPush()),
          [
            { label: 'Push tags', onClick: onPushTags },
            { label: 'Set upstream…', onClick: onPushSetUpstream },
            { label: 'Force-push (with lease)…', onClick: onForcePush, destructive: true },
          ],
          syncing,
          !!branchStatus && branchStatus.ahead > 0,
        )}
        {fetchAllOp.error && <span className="header-bar-fetch-error">Could not fetch: {fetchAllOp.error}</span>}
      </div>
      {syncMenu && <ContextMenu state={syncMenu} onClose={() => setSyncMenu(null)} />}
      <span className="header-bar-divider" />

      <button type="button" className="header-bar-quick-switch" onClick={() => setQuickSwitchOpen(true)}>
        Go to anything <Kbd>{System.IsMac() ? '⌘K' : 'Ctrl K'}</Kbd>
      </button>

      {accessory}

      {onToggleInbox && (
        <button
          type="button"
          className={inboxOpen ? 'header-bar-icon-button header-bar-icon-button-active' : 'header-bar-icon-button'}
          aria-pressed={inboxOpen}
          aria-label="Inbox"
          title="Inbox: what waits on you across the open repositories"
          onClick={onToggleInbox}
        >
          <Inbox size={18} strokeWidth={1.75} />
        </button>
      )}

      <button
        type="button"
        className="header-bar-icon-button"
        onClick={onOpenSettings}
        aria-label="Settings"
        title="Settings"
      >
        <Settings size={18} strokeWidth={1.75} />
      </button>

      {currentUser && <IdentityBadge identity={currentUser} name={currentUser.name} email={currentUser.email} />}

      {quickSwitchOpen && (
        <QuickSwitchPalette
          repoPath={repoPath}
          repo={repo}
          onClose={() => setQuickSwitchOpen(false)}
          onBranchChanged={onBranchChanged}
          workspaceActions={workspaceActions}
          onOpenSettings={onOpenSettings}
          onPull={onPull}
          onPullMerge={onPullMerge}
          onPullRebase={onPullRebase}
          onPush={onPush}
          onFetchAll={fetchAllOp.fetchAll}
          onOpenFile={setFileHistoryPath}
          onSearchFiles={() => setSearchOpen(true)}
          onOpenCommit={onOpenCommit}
        />
      )}
      {fileHistoryPath && (
        <FileHistoryPanel repoPath={repoPath} path={fileHistoryPath} onClose={() => setFileHistoryPath(null)} />
      )}
      {searchOpen && (
        <SearchModal
          repoPath={repoPath}
          onClose={() => setSearchOpen(false)}
          onOpenFileHistory={(path) => {
            setSearchOpen(false)
            setFileHistoryPath(path)
          }}
        />
      )}
      {pickerOpen && (
        <RepoPickerModal
          repo={repo}
          defaultLocation={defaultRepoLocation}
          onOpenSettings={() => {
            setPickerOpen(false)
            onOpenSettings()
          }}
          onClose={() => setPickerOpen(false)}
        />
      )}
    </header>
  )
}

export default HeaderBar
