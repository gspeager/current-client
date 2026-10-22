import { DiffWrapContext } from './features/diff/diffWrap'
import { useEffect, useImperativeHandle, useRef, useState, type ReactNode, type Ref } from 'react'
import { useEventCallback } from 'usehooks-ts'
import { Settings } from 'lucide-react'
import { GitService, HistoryService, PlatformService } from '@current-client-bindings/app'
import ConflictBanner from './features/branches/ConflictBanner'
import GitMissing from './features/repositories/GitMissing'
import HeaderBar from './components/chrome/HeaderBar'
import KeepAlive from './components/chrome/KeepAlive'
import NavPane from './components/chrome/NavPane'
import RepoSwitcher from './features/repositories/RepoSwitcher'
import ResizeHandle from './components/chrome/ResizeHandle'
import SettingsPanel from './features/settings/SettingsPanel'
import StatusBar from './components/chrome/StatusBar'
import TabBar, { type AppTab } from './features/repositories/TabBar'
import { useWorkspaceActions } from './components/chrome/useWorkspaceActions'
import { LaneThemeContext, toLaneThemeName } from './lib/laneColor'
import { useAsyncData } from './lib/useAsyncData'
import { useAutoFetch } from './features/remotes/useAutoFetch'
import { useConflictState } from './features/branches/useConflictState'
import { useFileWatcher } from './features/repositories/useFileWatcher'
import { useLastFetchTime } from './features/remotes/useLastFetchTime'
import { DIVERGED, useRemoteSync } from './features/remotes/useRemoteSync'
import { useRepositoryLifecycle } from './features/repositories/useRepositoryLifecycle'
import { useRepoVersion } from './features/repositories/useRepoVersion'
import { useResizableWidth } from './lib/useResizableWidth'
import { useHostPrompt } from './features/settings/useHostPrompt'
import { useSettings } from './features/settings/useSettings'
import { useTabMemory } from './features/repositories/useTabMemory'
import { useWorkingTree } from './features/working-copy/useWorkingTree'
import ActivityView from './features/activity/ActivityView'
import ChangelogView from './features/changelog/ChangelogView'
import ChangesView from './features/working-copy/ChangesView'
import HistoryView from './features/history/HistoryView'
import { refFromSaved, savedFromRef } from './features/history/branchFilter'
import './App.scss'

// Each request remounts History so its initial selection, deep load, and scroll apply again.
interface CommitFocus {
  repoPath: string | null
  position: number | null
  ref: string | null
  request: number
}

export interface CurrentClientAppHandle {
  // Selects a commit on the current branch in History.
  openCommit: (sha: string) => Promise<void>
  // Shows History for a branch, with its tip selected.
  openBranch: (name: string) => Promise<void>
}

// Everything here is optional and only set by an app embedding Current Client.
export interface AppProps {
  headerAccessory?: ReactNode
  // The repository both the host and Current Client show; Current Client opens or focuses its tab.
  activeRepoPath?: string | null
  onActiveRepoChange?: (repoPath: string | null) => void
  handle?: Ref<CurrentClientAppHandle>
}

function App({ headerAccessory, activeRepoPath = null, onActiveRepoChange, handle }: AppProps = {}) {
  const {
    data: gitVersion,
    error: gitError,
    reload: reloadGitVersion,
  } = useAsyncData(() => GitService.GetGitVersion(), [])
  const [settingsOpen, setSettingsOpen] = useState(false)
  // Bumped when a host switch may have changed, so the status bar's indicator follows.
  const [hostsVersion, setHostsVersion] = useState(0)
  const bumpHostsVersion = () => setHostsVersion((n) => n + 1)
  // false only in a build made with the nohost tag; until known, assume the full build.
  const { data: hostIntegrationBuilt } = useAsyncData(() => PlatformService.HostIntegrationAvailable(), [])
  const hostIntegration = hostIntegrationBuilt !== false
  const [commitFocus, setCommitFocus] = useState<CommitFocus | null>(null)
  // Bumped to remount Changes on a newly requested file, as commitFocus does for History.
  const [fileFocus, setFileFocus] = useState(0)

  const repo = useRepositoryLifecycle(activeRepoPath)
  const { repoVersion, bumpRepoVersion, justBumped } = useRepoVersion()
  const { lastFetchedAt, reloadLastFetchTime } = useLastFetchTime(repo.repoPath)
  const navWidth = useResizableWidth('pane-width-nav', 212, 180, 400)
  const settings = useSettings()
  const navCollapsed = settings.settings?.navCollapsed ?? false
  // A pull fetches too, so it moves "fetched … ago" as well as the branch.
  const { pull, pullMerge, pullRebase, push, forcePush, pullOp, pushOp } = useRemoteSync(repo.repoPath, () => {
    bumpRepoVersion()
    reloadLastFetchTime()
  })
  const workingTree = useWorkingTree(repo.repoPath)
  const { conflictState, reloadConflictState } = useConflictState(repo.repoPath, repoVersion)
  const conventionalCommitsOn = !(settings.settings?.disableConventionalCommits ?? false)
  const { activeTab: rememberedTab, setActiveTab, restored, remember } = useTabMemory(repo.repoPath)
  const activeTab = rememberedTab === 'changelog' && !conventionalCommitsOn ? 'working-copy' : rememberedTab
  const { data: conventional } = useAsyncData(
    () => (repo.repoPath ? HistoryService.FollowsConventionalCommits(repo.repoPath) : null),
    [repo.repoPath],
    { refreshKey: repoVersion },
  )

  const onBranchChanged = () => {
    bumpRepoVersion()
    workingTree.loadStatus()
  }

  const onFetched = () => {
    bumpRepoVersion()
    reloadLastFetchTime()
  }

  // Only ref moves bump repoVersion, since that remounts History and refetches Activity,
  // conflict state included. A fetch from a terminal moves remote branches, so it also
  // updates the fetch time. The app's own actions have already bumped it.
  const onFileWatcherChanged = (refChanged: boolean) => {
    workingTree.loadStatus()
    if (!refChanged) {
      reloadConflictState()
    } else if (!justBumped()) {
      bumpRepoVersion()
      reloadLastFetchTime()
    }
  }

  useAutoFetch(repo.repoPath, settings.settings?.autoFetchIntervalMinutes ?? 0, onFetched)
  useFileWatcher(repo.repoPath, onFileWatcherChanged)
  useHostPrompt(hostIntegration ? repo.repoPath : null, bumpHostsVersion)

  // Settings may have changed the git executable path.
  const closeSettings = () => {
    setSettingsOpen(false)
    reloadGitVersion()
  }

  const changeTab = (tab: AppTab) => {
    setActiveTab(tab)
    setCommitFocus(null)
  }

  // position is only known when the commit may be past History's first loaded page.
  const openCommit = (sha: string, position: number | null = null, ref: string | null = null) => {
    remember({ selectedCommitSha: sha })
    setActiveTab('history')
    setCommitFocus((prev) => ({ repoPath: repo.repoPath, position, ref, request: (prev?.request ?? 0) + 1 }))
  }

  // A failed action expands the sidebar, so its message is visible.
  const workspace = useWorkspaceActions({
    repoPath: repo.repoPath,
    onBranchChanged,
    onOpenCommit: (sha) => openCommit(sha),
    onFailure: () => settings.setNavCollapsed(false),
  })

  useImperativeHandle(handle, () => ({
    openCommit: async (sha) => {
      if (!repo.repoPath) return
      openCommit(sha, await HistoryService.GetCommitPosition(repo.repoPath, sha))
    },
    openBranch: async (name) => {
      if (!repo.repoPath) return
      openCommit(await HistoryService.ResolveCommit(repo.repoPath, name), null, name)
    },
  }))

  // The initial activeRepoPath is handled by useRepositoryLifecycle; later changes open or focus a tab.
  const hostRepoSeen = useRef(activeRepoPath)
  useEffect(() => {
    if (activeRepoPath === hostRepoSeen.current) return
    hostRepoSeen.current = activeRepoPath
    if (activeRepoPath && activeRepoPath !== repo.repoPath) repo.openRecent(activeRepoPath)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [activeRepoPath])

  const reportActiveRepo = useEventCallback((path: string | null) => onActiveRepoChange?.(path))
  useEffect(() => reportActiveRepo(repo.repoPath), [repo.repoPath, reportActiveRepo])

  const openWorkingFile = (path: string) => {
    remember({ selectedFilePath: path })
    setActiveTab('working-copy')
    setFileFocus((n) => n + 1)
  }

  const onConflictResolved = () => {
    reloadConflictState()
    onBranchChanged()
  }

  const resetNotice = settings.resetNotice && (
    <p className="app-inline-error">
      {settings.resetNotice}
      <button onClick={settings.dismissResetNotice}>Dismiss</button>
    </p>
  )

  // Without a usable git nothing else works, so it takes over even with a repo open.
  if (!repo.repoPath || gitError) {
    return (
      <div className="app-welcome">
        <div className="app-welcome-corner">
          {headerAccessory}
          <button
            type="button"
            className="app-welcome-settings"
            onClick={() => setSettingsOpen(true)}
            aria-label="Settings"
            title="Settings"
          >
            <Settings size={18} strokeWidth={1.75} />
          </button>
        </div>
        <h1>Current Client</h1>
        {resetNotice}
        {gitError ? (
          <GitMissing error={gitError} onCheckAgain={reloadGitVersion} onSetPath={() => setSettingsOpen(true)} />
        ) : gitVersion ? (
          <p className="app-welcome-status">git {gitVersion}</p>
        ) : (
          <p className="app-welcome-status">Detecting Git installation…</p>
        )}
        {!gitError && <RepoSwitcher repo={repo} />}
        {settingsOpen && (
          <SettingsPanel
            settings={settings}
            repoPath={repo.repoPath}
            hostIntegration={hostIntegration}
            onClose={closeSettings}
            onHostsChanged={bumpHostsVersion}
          />
        )}
      </div>
    )
  }

  const dirty = (workingTree.files?.length ?? 0) > 0
  const focus = commitFocus?.repoPath === repo.repoPath ? commitFocus : null

  return (
    <LaneThemeContext.Provider value={toLaneThemeName(settings.settings?.laneColorTheme)}>
      <DiffWrapContext.Provider
        value={{ wrap: !(settings.settings?.diffNoWrap ?? false), setWrap: (wrap) => settings.setDiffNoWrap(!wrap) }}
      >
        <div className="app-shell">
          <HeaderBar
            repoPath={repo.repoPath}
            repoVersion={repoVersion}
            lastFetchedAt={lastFetchedAt}
            pruneOnFetch={settings.settings?.pruneOnFetch ?? false}
            onPruneOnFetchChange={settings.setPruneOnFetch}
            repo={repo}
            dirty={dirty}
            workspaceActions={workspace.actions}
            onOpenSettings={() => setSettingsOpen(true)}
            onPull={pull}
            onPullMerge={pullMerge}
            onPullRebase={pullRebase}
            onPush={push}
            onForcePush={() => void forcePush()}
            onBranchChanged={onBranchChanged}
            onFetched={onFetched}
            onOpenCommit={openCommit}
            syncing={pullOp.running || pushOp.running}
            accessory={headerAccessory}
          />
          <TabBar active={activeTab} onChange={changeTab} showChangelog={conventionalCommitsOn} />

          {conflictState && (
            <ConflictBanner
              repoPath={repo.repoPath}
              state={conflictState}
              onResolved={onConflictResolved}
              onOpenFile={openWorkingFile}
            />
          )}

          {settingsOpen && (
            <SettingsPanel
              settings={settings}
              repoPath={repo.repoPath}
              hostIntegration={hostIntegration}
              onClose={closeSettings}
              onHostsChanged={bumpHostsVersion}
            />
          )}
          {workspace.modals}

          {resetNotice}
          {pullOp.running && (
            <p className="app-inline-notice">
              Pulling… <button onClick={pullOp.cancel}>Cancel</button>
            </p>
          )}
          {pushOp.running && (
            <p className="app-inline-notice">
              Pushing… <button onClick={pushOp.cancel}>Cancel</button>
            </p>
          )}
          {pullOp.error && (
            <p className="app-inline-error">
              Could not pull: {pullOp.error}
              {pullOp.error === DIVERGED && (
                <>
                  <button onClick={pullMerge}>Pull (merge)</button>
                  <button onClick={pullRebase}>Pull (rebase)</button>
                </>
              )}
            </p>
          )}
          {pushOp.error && <p className="app-inline-error">Could not push: {pushOp.error}</p>}

          <div className="app-body">
            <NavPane
              key={repo.repoPath}
              repoPath={repo.repoPath}
              repoVersion={repoVersion}
              workingTree={workingTree}
              workspaceActions={workspace.actions}
              workspaceFeedback={workspace.feedback}
              onBranchChanged={onBranchChanged}
              onFetched={onFetched}
              pruneOnFetch={settings.settings?.pruneOnFetch ?? false}
              onOpenHeadCommit={openCommit}
              onOpenRepository={repo.openRecent}
              width={navWidth.width}
              collapsed={navCollapsed}
              onCollapsedChange={settings.setNavCollapsed}
            />
            {!navCollapsed && <ResizeHandle onDragStart={navWidth.onDragStart} ariaLabel="Resize nav pane" />}

            <main className="app-main">
              <KeepAlive active={activeTab === 'working-copy'}>
                <ChangesView
                  key={`${repo.repoPath}-${repoVersion}-${fileFocus}`}
                  repoPath={repo.repoPath}
                  workingTree={workingTree}
                  onPushRequested={push}
                  onBranchChanged={onBranchChanged}
                  defaultIgnoreWhitespace={settings.settings?.diffIgnoreWhitespaceDefault ?? false}
                  diffExpanded={settings.settings?.diffExpanded ?? false}
                  onDiffExpandedChange={settings.setDiffExpanded}
                  initialSelectedPath={restored.selectedFilePath}
                  onSelectedPathChange={(path) => remember({ selectedFilePath: path })}
                  initialCommitDraft={restored.commitDraft}
                  onCommitDraftChange={(message) => remember({ commitDraft: message })}
                  showCommitTypePicker={conventionalCommitsOn}
                />
              </KeepAlive>
              <KeepAlive active={activeTab === 'history'}>
                <HistoryView
                  key={`${repo.repoPath}-${repoVersion}-${focus?.request ?? 0}`}
                  repoPath={repo.repoPath}
                  conventional={conventional ?? false}
                  initialSelectedSha={restored.selectedCommitSha}
                  initialLoadThrough={focus?.position ?? null}
                  initialRef={focus?.ref ?? null}
                  defaultRef={refFromSaved(settings.settings?.historyBranchFilter)}
                  onRefChange={(ref) => settings.setHistoryBranchFilter(savedFromRef(ref))}
                  onBranchChanged={onBranchChanged}
                  previousTopSha={restored.historyTopSha}
                  onTopShaChange={(sha) => remember({ historyTopSha: sha })}
                  onSelectedShaChange={(sha) => remember({ selectedCommitSha: sha })}
                />
              </KeepAlive>
              <KeepAlive active={activeTab === 'changelog'}>
                <ChangelogView
                  key={repo.repoPath}
                  repoPath={repo.repoPath}
                  repoVersion={repoVersion}
                  prefs={settings.settings?.changelog ?? null}
                  onPrefsChange={settings.setChangelogPrefs}
                  onOpenCommit={openCommit}
                />
              </KeepAlive>
              <KeepAlive active={activeTab === 'activity'}>
                <ActivityView
                  key={repo.repoPath}
                  repoPath={repo.repoPath}
                  repoVersion={repoVersion}
                  dirty={dirty}
                  hasConflict={conflictState !== null}
                />
              </KeepAlive>
            </main>
          </div>

          <StatusBar
            repoPath={repo.repoPath}
            repoVersion={repoVersion}
            gitVersion={gitVersion}
            hostsVersion={hostsVersion}
            hostIntegration={hostIntegration}
            onOpenSettings={() => setSettingsOpen(true)}
          />
        </div>
      </DiffWrapContext.Provider>
    </LaneThemeContext.Provider>
  )
}

export default App
