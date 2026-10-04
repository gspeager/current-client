import { useEffect, useRef, useState, type DragEvent, type ReactNode } from 'react'
import { System } from '@wailsio/runtime'
import { ArrowDown, ArrowUp, ChevronDown, Plus, RefreshCw, Settings, X } from 'lucide-react'
import { useOnClickOutside } from 'usehooks-ts'
import BranchPill from '../git/BranchPill'
import Checkbox from '../forms/Checkbox'
import FileHistoryPanel from '../../features/history/FileHistoryPanel'
import IdentityBadge from '../git/IdentityBadge'
import Kbd from '../controls/Kbd'
import { useBranchStatus } from '../../lib/useBranchStatus'
import { useCurrentUser } from '../../lib/useCurrentUser'
import { useFetchAll } from '../../features/remotes/useFetchAll'
import { useLaneColors } from '../../lib/laneColor'
import { baseName } from '../../lib/paths'
import { relativeTime } from '../../lib/relativeTime'
import type { useRepositoryLifecycle } from '../../features/repositories/useRepositoryLifecycle'
import BreadcrumbPanel, { type PanelView } from './BreadcrumbPanel'
import QuickSwitchPalette from './QuickSwitchPalette'
import RepoPickerModal from '../../features/repositories/RepoPickerModal'
import RepoTabChip from '../../features/repositories/RepoTabChip'
import './HeaderBar.scss'
import { useWindowKeydown } from '../../lib/useWindowKeydown'

interface HeaderBarProps {
  repoPath: string
  repoVersion: number
  lastFetchedAt: Date | null
  pruneOnFetch: boolean
  onPruneOnFetchChange: (prune: boolean) => void
  repo: ReturnType<typeof useRepositoryLifecycle>
  dirty: boolean
  onOpenSettings: () => void
  onPull: () => void
  onPush: () => void
  onForcePush: () => void
  onBranchChanged?: () => void
  onFetched?: () => void
  onOpenCommit: (sha: string, position: number) => void
  syncing: boolean
  // Supplied by an app embedding Current Client, such as a switch between its views.
  accessory?: ReactNode
}

function HeaderBar({
  repoPath,
  repoVersion,
  lastFetchedAt,
  pruneOnFetch,
  onPruneOnFetchChange,
  repo,
  dirty,
  onOpenSettings,
  onPull,
  onPush,
  onForcePush,
  onBranchChanged,
  onFetched,
  onOpenCommit,
  syncing,
  accessory,
}: HeaderBarProps) {
  const { branchColor } = useLaneColors()
  const currentUser = useCurrentUser(repoPath)
  const branchStatus = useBranchStatus(repoPath, repoVersion)
  const fetchAllOp = useFetchAll(repoPath, onFetched, pruneOnFetch)
  const [branchSwitcherOpen, setBranchSwitcherOpen] = useState(false)
  const [panelView, setPanelView] = useState<PanelView>('branch')
  const [quickSwitchOpen, setQuickSwitchOpen] = useState(false)
  const [fileHistoryPath, setFileHistoryPath] = useState<string | null>(null)
  const [dragPath, setDragPath] = useState<string | null>(null)
  const [pickerOpen, setPickerOpen] = useState(false)
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

  useWindowKeydown((e) => {
    if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === 'k') {
      e.preventDefault()
      setQuickSwitchOpen(true)
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
                        name={branchStatus.current}
                        color={branchColor(branchStatus.current, true)}
                        current
                        bare
                        dirty={dirty}
                      />
                    </span>
                  </>
                )}
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

      {branchStatus && branchStatus.behind > 0 && (
        <button
          type="button"
          className="header-bar-sync header-bar-sync-behind"
          onClick={syncing ? undefined : onPull}
          title="Pull"
        >
          <ArrowDown size={12} strokeWidth={1.5} />
          Pull {branchStatus.behind}
        </button>
      )}

      {branchStatus && branchStatus.ahead > 0 && (
        <button
          type="button"
          className="header-bar-sync header-bar-sync-ahead"
          onClick={syncing ? undefined : (e) => (e.shiftKey ? onForcePush() : onPush())}
          title="Push (shift-click to force push)"
        >
          <ArrowUp size={12} strokeWidth={1.5} />
          Push {branchStatus.ahead}
        </button>
      )}

      <span className="header-bar-fetch-status">
        {fetchAllOp.running
          ? 'Fetching…'
          : syncing
            ? 'Syncing…'
            : lastFetchedAt
              ? `fetched ${relativeTime(lastFetchedAt)}`
              : 'Not fetched yet'}
      </span>
      <button
        type="button"
        className="header-bar-fetch"
        onClick={fetchAllOp.running ? fetchAllOp.cancel : fetchAllOp.fetchAll}
        disabled={syncing}
        title={fetchAllOp.running ? 'Cancel fetch' : 'Fetch all remotes'}
      >
        <RefreshCw size={12} strokeWidth={1.5} />
        {fetchAllOp.running ? 'Cancel' : 'Fetch'}
      </button>
      <span
        className="header-bar-fetch-prune"
        title="Delete remote branches that are gone from the remote when fetching"
      >
        <Checkbox checked={pruneOnFetch} onChange={onPruneOnFetchChange} label="Prune" />
      </span>
      {fetchAllOp.error && <span className="header-bar-fetch-error">Could not fetch: {fetchAllOp.error}</span>}

      <button type="button" className="header-bar-quick-switch" onClick={() => setQuickSwitchOpen(true)}>
        Quick switch <Kbd>{System.IsMac() ? '⌘K' : 'Ctrl K'}</Kbd>
      </button>

      {accessory}

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
          onOpenSettings={onOpenSettings}
          onPull={onPull}
          onPush={onPush}
          onFetchAll={fetchAllOp.fetchAll}
          onOpenFile={setFileHistoryPath}
          onOpenCommit={onOpenCommit}
        />
      )}
      {fileHistoryPath && (
        <FileHistoryPanel repoPath={repoPath} path={fileHistoryPath} onClose={() => setFileHistoryPath(null)} />
      )}
      {pickerOpen && <RepoPickerModal repo={repo} onClose={() => setPickerOpen(false)} />}
    </header>
  )
}

export default HeaderBar
