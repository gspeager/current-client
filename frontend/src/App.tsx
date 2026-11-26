import { useEffect, useImperativeHandle, useMemo, useRef, useState, type ReactNode, type Ref } from 'react'
import { useEventCallback } from 'usehooks-ts'
import {
  BranchService,
  GitService,
  HistoryService,
  InboxService,
  type NoteInfo,
  type PushResult,
} from '@current-client-bindings/app'
import ConflictBanner from './features/branches/ConflictBanner'
import HeaderBar from './features/shell/HeaderBar'
import KeepAlive from './components/chrome/KeepAlive'
import NavPane from './features/shell/NavPane'
import ResizeHandle from './components/chrome/ResizeHandle'
import SettingsPanel, { settingsGroups } from './features/settings/SettingsPanel'
import StatusBar from './components/chrome/StatusBar'
import TabBar, { type AppTab } from './features/repositories/TabBar'
import WelcomeScreen from './features/repositories/WelcomeScreen'
import AppProviders from './features/shell/AppProviders'
import { appPaletteActions } from './features/shell/appPaletteActions'
import { useNavigation } from './features/shell/useNavigation'
import { useWorkspaceActions } from './features/shell/useWorkspaceActions'
import type { ActionResult } from './lib/actionResult'
import { assignLaneSlots } from './lib/laneColor'
import { useAsyncData } from './lib/useAsyncData'
import { usePerRepository } from './lib/usePerRepository'
import { createSidebarCommands } from './lib/sidebarCommands'
import { useWindowKeydown } from './lib/useWindowKeydown'
import { isTypingTarget, matchesShortcut } from './lib/shortcuts'
import { NARROW_SIDEBAR_WIDTH, useWindowNarrowerThan } from './lib/useWindowNarrowerThan'
import { cyclePanes, focusMain, focusMainSoon } from './lib/panes'
import ShortcutSheet from './components/chrome/ShortcutSheet'
import { useAutoFetch } from './features/remotes/useAutoFetch'
import { useConflictState } from './features/branches/useConflictState'
import { useFileWatcher } from './features/repositories/useFileWatcher'
import { useLastFetchTime } from './features/remotes/useLastFetchTime'
import SyncNotices from './features/remotes/SyncNotices'
import { useRemoteSync } from './features/remotes/useRemoteSync'
import { useRepositoryLifecycle } from './features/repositories/useRepositoryLifecycle'
import { useRepoVersion } from './features/repositories/useRepoVersion'
import { useResizableWidth } from './lib/useResizableWidth'
import HostChangesView from './features/host-changes/HostChangesView'
import { useHostState } from './features/host-changes/useHostState'
import { changeNoun } from './lib/changeNoun'
import HostNoticeBanner from './components/chrome/HostNoticeBanner'
import CreateChangeDialog from './features/host-changes/CreateChangeDialog'
import IssuePanel from './features/host-changes/IssuePanel'
import IssuesView from './features/issues/IssuesView'
import NewIssueDialog from './features/host-changes/NewIssueDialog'
import InboxView, { type InboxTarget } from './features/inbox/InboxView'
import StandupDialog from './features/inbox/StandupDialog'
import type { IssueDraft } from './lib/issuePanel'
import { useHostPrompt } from './features/settings/useHostPrompt'
import { useHostLinks } from './lib/useHostLinks'
import { openWebPage } from './lib/hostPages'
import { useBranchChanges } from './lib/useBranchChanges'
import { useSettings } from './features/settings/useSettings'
import { useOpenRequests } from './features/repositories/useOpenRequests'
import TasksView from './features/tasks/TasksView'
import { useTasks } from './features/tasks/useTasks'
import WorktreeBoard from './features/worktrees/WorktreeBoard'
import TerminalPanel from './features/terminal/TerminalPanel'
import { isTerminalToggle } from './features/terminal/terminalKeys'
import { useTerminals } from './features/terminal/useTerminals'
import { useTabMemory } from './features/repositories/useTabMemory'
import { useWorkingTree } from './features/working-copy/useWorkingTree'
import ActivityView from './features/activity/ActivityView'
import ChangelogView from './features/changelog/ChangelogView'
import ChangesView from './features/working-copy/ChangesView'
import HistoryView from './features/history/HistoryView'
import ExploreView from './features/explore/ExploreView'
import ReadThroughSheet from './features/read-through/ReadThroughSheet'
import { openingOf, type ReadThroughOpening } from './lib/readThrough'
import { useCurrentClientRoot } from './lib/currentClientRoot'
import NotesDialogs from './features/notes/NotesDialogs'
import { targetOf, useNotesState } from './features/notes/notes'
import { refFromSaved, savedFromRef } from './features/history/branchFilter'
import './App.scss'

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
  // Leaves Check for updates out of Settings, for an app with its own releases page.
  hideUpdateCheck?: boolean
  // For a host app that has its own terminal, or doesn't want one.
  hideTerminal?: boolean
  hideExplore?: boolean
  // False for a host app that asks about turning on a host's integration in its own words.
  hostPrompt?: boolean
  handle?: Ref<CurrentClientAppHandle>
}

function App({
  headerAccessory,
  activeRepoPath = null,
  onActiveRepoChange,
  hideUpdateCheck = false,
  hideTerminal = false,
  hideExplore = false,
  hostPrompt = true,
  handle,
}: AppProps = {}) {
  const {
    data: gitVersion,
    error: gitError,
    reload: reloadGitVersion,
  } = useAsyncData(() => GitService.GetGitVersion(), [])
  const [settingsOpen, setSettingsOpen] = useState(false)
  const [settingsGroup, setSettingsGroup] = useState<string | null>(null)
  const openSettings = (group?: string) => {
    setSettingsGroup(group ?? null)
    setSettingsOpen(true)
  }
  const [standupOpen, setStandupOpen] = useState(false)
  const [createChangeFor, setCreateChangeFor] = useState<string | null>(null)

  // Hidden behind a host app's view and never shown, nothing reads the repository
  // for the screen: viewPath stays null. Restoring tabs, the fetch timer and the
  // file watcher still use repo.repoPath.
  const { active } = useCurrentClientRoot()
  const [shown, setShown] = useState(active)
  if (active && !shown) setShown(true)
  const repo = useRepositoryLifecycle(activeRepoPath, shown)
  const viewPath = shown ? repo.repoPath : null
  const host = useHostState(repo.repoPath, viewPath)
  const { integration, changesLabel, issuesOn } = host
  const hostLinks = useHostLinks(viewPath ?? '')
  const { repoVersion, bumpRepoVersion, justBumped } = useRepoVersion()
  // The branch's known change, to offer a create link only while it has none.
  const changes = useBranchChanges(viewPath ?? '', repoVersion)
  const { lastFetchedAt, reloadLastFetchTime } = useLastFetchTime(viewPath)
  const navWidth = useResizableWidth('pane-width-nav', 212, 180, 400)
  const settings = useSettings()
  const prefs = settings.values

  // Per repository: another repository starts without them.
  const [pushed, setPushed] = usePerRepository<PushResult | null>(repo.repoPath, null)
  const [actionResult, setActionResult] = usePerRepository<ActionResult | null>(repo.repoPath, null)
  const [openIssueKey, setOpenIssueKey] = usePerRepository<string | null>(repo.repoPath, null)
  const [newIssueDraft, setNewIssueDraft] = usePerRepository<IssueDraft | null>(repo.repoPath, null)
  const [readThrough, setReadThrough] = usePerRepository<ReadThroughOpening | null>(repo.repoPath, null)
  const [boardOpen, setBoardOpen] = usePerRepository(repo.repoPath, false)
  const [issueFocus, setIssueFocus] = usePerRepository<string | null>(repo.repoPath, null)

  const onFetched = () => {
    bumpRepoVersion()
    reloadLastFetchTime()
    host.collectNotices()
  }

  const terminalsOff = hideTerminal || prefs.localCommandsOff
  const tasks = useTasks(terminalsOff ? null : viewPath, repoVersion)
  // A pull fetches too, so it moves "fetched … ago" as well as the branch.
  const { pull, pullMerge, pullRebase, push, forcePush, pushTags, pushSetUpstream, pullOp, pushOp } = useRemoteSync(
    repo.repoPath,
    onFetched,
    setPushed,
    () => void tasks.runChecklist('push'),
    setActionResult,
  )
  // Starting another sync retires the last outcome, which may no longer hold.
  useEffect(() => {
    if (pullOp.running || pushOp.running) setActionResult(null)
  }, [pullOp.running, pushOp.running, setActionResult])
  const workingTree = useWorkingTree(viewPath)
  const { conflictState, reloadConflictState } = useConflictState(viewPath, repoVersion)
  const operation = conflictState?.operation ?? null
  const conventionalCommitsOn = !prefs.disableConventionalCommits
  // With its repository, since a tab switch renders once before the old list is cleared.
  const { data: branchList } = useAsyncData(
    () => {
      const repoPath = viewPath
      return repoPath ? BranchService.ListLocal(repoPath).then((branches) => ({ repoPath, branches })) : null
    },
    [viewPath],
    { refreshKey: repoVersion },
  )
  const localBranches = branchList?.repoPath === repo.repoPath ? branchList.branches : null
  const currentBranch = localBranches?.find((b) => b.current)?.name
  const {
    activeTab: rememberedTab,
    setActiveTab,
    restored,
    remember,
    memoryBranch,
    forgetBranchesExcept,
  } = useTabMemory(repo.repoPath, localBranches ? (currentBranch ?? '') : null, !!operation)
  useEffect(() => {
    if (localBranches) forgetBranchesExcept(localBranches.map((b) => b.name))
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [localBranches])
  const notesState = useNotesState(viewPath, repoVersion)
  const { data: conventional } = useAsyncData(
    () => (viewPath ? HistoryService.FollowsConventionalCommits(viewPath) : null),
    [viewPath],
    { refreshKey: repoVersion },
  )
  const [openedInPlace, setOpenedInPlace] = useState(false)
  const nav = useNavigation({
    repoPath: repo.repoPath,
    setActiveTab,
    remember,
    onChangeTab: () => {
      setOpenIssueKey(null)
      setOpenedInPlace(false)
    },
  })
  const { changeTab, openCommit, openRevision, openWorkingFile, openChange } = nav

  const onBranchChanged = () => {
    bumpRepoVersion()
    workingTree.loadStatus()
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

  useAutoFetch(repo.repoPath, prefs.autoFetchIntervalMinutes, onFetched)
  useFileWatcher(repo.repoPath, onFileWatcherChanged)
  const terminals = useTerminals(repo.repoPath)
  const [terminalFocus, setTerminalFocus] = useState(0)
  const beforeTerminal = useRef<HTMLElement | null>(null)
  const openTasks = () => {
    if (repo.repoPath && !terminalsOff) terminals.showTasks(repo.repoPath)
  }
  // Opens the panel and takes the focus; from inside it, hides it and goes back.
  const toggleTerminal = () => {
    if (!repo.repoPath || terminalsOff) return
    const active = document.activeElement as HTMLElement | null
    if (active?.closest('.terminal-panel')) {
      terminals.hide(repo.repoPath)
      beforeTerminal.current?.focus()
      return
    }
    beforeTerminal.current = active
    terminals.show(repo.repoPath)
    setTerminalFocus((n) => n + 1)
  }
  const openTerminalIn = async (path: string) => {
    setBoardOpen(false)
    const opened = await repo.openRecent(path)
    if (!opened) return
    terminals.show(opened)
    setTerminalFocus((n) => n + 1)
  }
  const [shortcutsOpen, setShortcutsOpen] = useState(false)
  useWindowKeydown((e) => {
    if (e.key === 'Escape' && inboxOpen && !document.querySelector('.modal-panel, .context-menu')) {
      setInboxOpen(false)
      document.querySelector<HTMLElement>('.header-bar-inbox')?.focus()
      return
    }
    if (isTerminalToggle(e)) {
      e.preventDefault()
      toggleTerminal()
      return
    }
    if (e.key === 'F6') {
      e.preventDefault()
      cyclePanes(e.shiftKey)
      return
    }
    if (e.key === '?' && !isTypingTarget(e.target)) {
      setShortcutsOpen(true)
      return
    }
    if (matchesShortcut(e, 'mod+shift+Z')) {
      e.preventDefault()
      undoAction?.run()
      return
    }
    if (matchesShortcut(e, 'mod+B')) {
      e.preventDefault()
      setNavCollapsed(!navCollapsed)
    }
    const view = matchesShortcut(e, `mod+${e.key}`) && /^[1-9]$/.test(e.key) ? viewTabs[Number(e.key) - 1] : undefined
    if (view && repo.repoPath) {
      e.preventDefault()
      changeTab(view)
      requestAnimationFrame(() => focusMainSoon())
    }
  })
  const closeRepoTab = (path: string) =>
    void terminals.confirmCloseRepo(path).then((closed) => {
      if (closed) repo.closeTab(path)
    })
  useOpenRequests(repo)
  useHostPrompt(host.hostIntegration && hostPrompt ? viewPath : null, host.bumpHostsVersion)

  // Settings may have changed the git executable path.
  const closeSettings = () => {
    setSettingsOpen(false)
    reloadGitVersion()
  }

  // A failed action expands the sidebar, so its message is visible.
  const workspace = useWorkspaceActions({
    repoPath: repo.repoPath,
    hostLinks,
    editorPath: prefs.editorPath,
    operation,
    onOpenTerminal: terminalsOff ? undefined : toggleTerminal,
    onBranchChanged,
    onOpenCommit: (sha) => openCommit(sha),
    onFailure: () => setNavCollapsed(false),
    onUndone: setActionResult,
  })
  const undoAction = workspace.actions.find((a) => a.id === 'undo')

  useImperativeHandle(handle, () => ({
    openCommit: async (sha) => {
      if (!repo.repoPath) return
      openCommit(sha, await HistoryService.GetCommitPosition(repo.repoPath, sha))
    },
    openBranch: async (name) => {
      if (repo.repoPath) await openRevision(repo.repoPath, name, name)
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

  // A read-through can include uncommitted files, so it also changes with the working tree.
  const readThroughVersion = useMemo(
    () => ({ repoVersion, files: workingTree.files }),
    [repoVersion, workingTree.files],
  )
  const { hostOn, canWriteIssues, tracker } = host
  const issueActions = useMemo(
    () => ({
      open: hostOn || tracker?.reading ? setOpenIssueKey : undefined,
      create:
        hostOn && canWriteIssues
          ? (draft?: IssueDraft) => setNewIssueDraft(draft ?? { title: '', description: '' })
          : undefined,
      tracker,
    }),
    [hostOn, canWriteIssues, tracker, setOpenIssueKey, setNewIssueDraft],
  )
  const viewTabs: AppTab[] = [
    'working-copy',
    'history',
    ...(changesLabel ? (['changes'] as const) : []),
    ...(issuesOn ? (['issues'] as const) : []),
    'activity',
    'changelog',
    ...(hideExplore ? [] : (['explore'] as const)),
  ]
  const laneSlots = useMemo(() => assignLaneSlots((localBranches ?? []).map((b) => b.name)), [localBranches])
  const activeTab =
    (rememberedTab === 'changes' && !changesLabel) ||
    (rememberedTab === 'issues' && !issuesOn) ||
    (rememberedTab === 'explore' && hideExplore)
      ? 'working-copy'
      : rememberedTab
  const narrowWindow = useWindowNarrowerThan(NARROW_SIDEBAR_WIDTH)
  // Views with their own lists give the sidebar's width to their content; its
  // rail stays, and opening it there lasts until the app closes.
  const railView = activeTab !== 'working-copy' && activeTab !== 'history'
  const navCollapsed = narrowWindow || railView ? !openedInPlace : prefs.navCollapsed
  const setNavCollapsed = (collapsed: boolean) =>
    narrowWindow || railView ? setOpenedInPlace(!collapsed) : settings.setNavCollapsed(collapsed)

  // A row in the Inbox switches to its repository first; the jump itself
  // waits until that repository is the open one.
  const [inboxOpen, setInboxOpen] = useState(false)
  const [inboxTarget, setInboxTarget] = useState<InboxTarget | null>(null)
  // A row opens its target in the Inbox's place; the badge keeps the count.
  const goFromInbox = (target: InboxTarget) => {
    setInboxOpen(false)
    setInboxTarget(target)
    if (target.path !== repo.repoPath) repo.openRecent(target.path)
  }
  useEffect(() => {
    if (!inboxTarget || inboxTarget.path !== repo.repoPath) return
    const target = inboxTarget
    setInboxTarget(null)
    switch (target.kind) {
      case 'changes':
        setActiveTab('working-copy')
        break
      case 'branch':
        void openRevision(repo.repoPath, target.branch, target.branch)
        break
      case 'change':
        openChange(target.number)
        break
      case 'issue':
        if (issuesOn) {
          setIssueFocus(target.key)
          setActiveTab('issues')
        } else {
          setOpenIssueKey(target.key)
        }
        break
      case 'host':
        openWebPage(target.url)
        break
      case 'clean-up':
        requestSidebar('clean-up')
        break
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [inboxTarget, repo.repoPath])

  // With the host on, the in-app form; otherwise the host's pre-filled page.
  const requestCreateChange = integration ? (branch: string) => setCreateChangeFor(branch) : undefined
  // After a merge from the app whose branch is local, the sidebar opens
  // Clean up merged branches; the palette asks for its forms the same way.
  const [sidebarCommands] = useState(createSidebarCommands)
  const requestSidebar = sidebarCommands.run
  // Host rows the Inbox hasn't shown: counted after each refresh, and again
  // once the Inbox closes, since showing them counts as seeing them.
  const [inboxUnseen, setInboxUnseen] = useState(0)
  useEffect(() => {
    if (!shown || inboxOpen || repo.tabs.length === 0) return
    let active = true
    InboxService.Unseen(repo.tabs)
      .then((n) => active && setInboxUnseen(n))
      .catch(() => {})
    return () => {
      active = false
    }
  }, [shown, inboxOpen, repo.tabs, host.hostRefresh])

  const onChangeCreated = (number: number) => {
    setCreateChangeFor(null)
    setPushed(null)
    onBranchChanged()
    openChange(number)
  }

  const openNote = (note: NoteInfo) => {
    if (!repo.repoPath) return
    if (note.stale) {
      notesState.context.edit(targetOf(note))
    } else if (note.kind === 'review') {
      setReadThrough({ ...openingOf(note.target), at: { path: note.path, line: note.line } })
    } else if (note.kind === 'file') {
      openWorkingFile(note.target)
    } else if (note.kind === 'commit') {
      openCommit(note.target)
    } else {
      const ref = `refs/${note.kind === 'tag' ? 'tags' : 'heads'}/${note.target}`
      void openRevision(repo.repoPath, ref, note.kind === 'branch' ? note.target : null)
    }
  }

  // Only Abort reaches here; the message the composer held was the aborted operation's.
  const onConflictResolved = () => {
    remember({ commitDraft: '' })
    reloadConflictState()
    onBranchChanged()
  }

  const resetNotice = settings.resetNotice && (
    <p className="app-inline-error">
      {settings.resetNotice}
      <button onClick={settings.dismissResetNotice}>Dismiss</button>
    </p>
  )

  const settingsPanel = settingsOpen && (
    <SettingsPanel
      settings={settings}
      repoPath={repo.repoPath}
      hostIntegration={host.hostIntegration}
      updateCheck={!hideUpdateCheck}
      initialGroup={settingsGroup}
      onClose={closeSettings}
      onHostsChanged={host.bumpHostsVersion}
    />
  )

  // Drawn the first time it's shown, so its views read nothing before then: the
  // welcome screen's recent repositories too, shown while the saved tab opens.
  if (!shown) return null

  // Without a usable git nothing else works, so it takes over even with a repo open.
  if (!repo.repoPath || gitError) {
    return (
      <WelcomeScreen
        repo={repo}
        gitVersion={gitVersion}
        gitError={gitError}
        onCheckGitAgain={reloadGitVersion}
        defaultLocation={prefs.defaultRepoLocation}
        onOpenSettings={() => openSettings()}
        accessory={headerAccessory}
        notice={resetNotice}
      >
        {settingsPanel}
      </WelcomeScreen>
    )
  }

  const changedCount = workingTree.files?.length ?? 0
  const focus = nav.commitFocus
  const repoPath = repo.repoPath
  const changeKind = changeNoun(integration?.host.provider ?? '')
  const paletteActions = appPaletteActions({
    repoPath,
    currentBranch,
    changesLabel,
    changeKind,
    issuesOn,
    hostOn,
    hideExplore,
    terminalsOff,
    tasks,
    viewShortcut: (tab) => `mod+${viewTabs.indexOf(tab) + 1}`,
    changeTab,
    requestSidebar,
    createIssue: issueActions.create,
    requestCreateChange,
    openShortcuts: () => setShortcutsOpen(true),
    openReadThrough: (ref) => setReadThrough({ ref }),
    openInbox: () => setInboxOpen(true),
    openStandup: () => setStandupOpen(true),
    openTasks,
    openNotes: () => notesState.context.openList(),
    openWorktreeBoard: () => setBoardOpen(true),
  })

  return (
    <AppProviders
      settings={prefs}
      onDiffWrapChange={(wrap) => settings.setDiffNoWrap(!wrap)}
      laneSlots={laneSlots}
      issueActions={issueActions}
      notes={notesState.context}
      onReadThrough={setReadThrough}
      onOpenCommit={nav.openCommitAnywhere}
    >
      <div className="app-shell">
        <h1 className="visually-hidden">Current Client</h1>
        <button type="button" className="app-skip-link" onClick={focusMain}>
          Skip to content
        </button>
        <HeaderBar
          onUndo={undoAction?.run}
          onResult={setActionResult}
          onOpenChange={openChange}
          onShowStashes={() => requestSidebar('show-stash')}
          repoPath={repo.repoPath}
          repoVersion={repoVersion}
          lastFetchedAt={lastFetchedAt}
          pruneOnFetch={prefs.pruneOnFetch}
          repo={repo}
          dirty={changedCount > 0}
          inboxOpen={inboxOpen}
          inboxUnseen={inboxUnseen}
          onToggleInbox={() => setInboxOpen((open) => !open)}
          workspaceActions={workspace.actions}
          paletteActions={paletteActions}
          settingsGroups={settingsGroups(host.hostIntegration)}
          onOpenSettings={openSettings}
          onPull={pull}
          onPullMerge={pullMerge}
          onPullRebase={pullRebase}
          onPush={push}
          onForcePush={() => void forcePush()}
          onPushTags={pushTags}
          onPushSetUpstream={() => void pushSetUpstream()}
          onBranchChanged={onBranchChanged}
          onFetched={onFetched}
          onOpenCommit={openCommit}
          onOpenNote={openNote}
          onCloseTab={closeRepoTab}
          terminalsRunning={terminals.running}
          syncing={pullOp.running || pushOp.running}
          operation={operation}
          defaultRepoLocation={prefs.defaultRepoLocation}
          accessory={headerAccessory}
        />
        {shortcutsOpen && <ShortcutSheet onClose={() => setShortcutsOpen(false)} />}
        <TabBar
          active={activeTab}
          onChange={changeTab}
          changesLabel={changesLabel}
          issueCount={host.issueCount}
          hideExplore={hideExplore}
          offHost={host.offHost}
          onOffHost={() => openSettings('Integrations')}
          changedCount={changedCount}
          conflictCount={conflictState?.conflictedPaths.length ?? 0}
        />

        {conflictState && (
          <ConflictBanner
            repoPath={repo.repoPath}
            state={conflictState}
            onResolved={onConflictResolved}
            onOpenFile={openWorkingFile}
          />
        )}

        {settingsPanel}
        {workspace.modals}
        {standupOpen && <StandupDialog paths={repo.tabs} onClose={() => setStandupOpen(false)} />}
        {newIssueDraft && repo.repoPath && (
          <NewIssueDialog
            repoPath={repo.repoPath}
            draft={newIssueDraft}
            onCreated={(issue) => {
              setNewIssueDraft(null)
              setOpenIssueKey(issue.key)
            }}
            onClose={() => setNewIssueDraft(null)}
          />
        )}
        {boardOpen && repo.repoPath && (
          <WorktreeBoard
            repoPath={repo.repoPath}
            onOpenRepository={(path) => void repo.openRecent(path)}
            onOpenTerminal={terminalsOff ? undefined : (path) => void openTerminalIn(path)}
            tasks={terminalsOff ? undefined : tasks}
            onNewWorktree={() => {
              setBoardOpen(false)
              // Once closing has put the focus back, so the form can take it.
              requestAnimationFrame(() => requestSidebar('new-worktree'))
            }}
            onClose={() => setBoardOpen(false)}
          />
        )}
        {readThrough && repo.repoPath && (
          <ReadThroughSheet
            repoPath={repo.repoPath}
            opening={readThrough}
            refreshKey={readThroughVersion}
            onNotesChanged={notesState.reload}
            onClose={() => setReadThrough(null)}
          />
        )}
        {createChangeFor && repo.repoPath && (
          <CreateChangeDialog
            repoPath={repo.repoPath}
            branch={createChangeFor}
            kind={changeKind}
            onCreated={onChangeCreated}
            onClose={() => setCreateChangeFor(null)}
          />
        )}

        {resetNotice}
        <HostNoticeBanner
          notices={host.notices}
          onOpenChange={openChange}
          onOpenFile={openWorkingFile}
          onDismiss={host.dismissNotices}
        />
        <SyncNotices
          pullOp={pullOp}
          pushOp={pushOp}
          onPullMerge={pullMerge}
          onPullRebase={pullRebase}
          pushed={pushed?.createLink && !changes[pushed.upstream]?.state ? pushed : null}
          onCreateChange={(p) => (requestCreateChange ? requestCreateChange(p.branch) : openWebPage(p.createLink))}
          onDismissPushed={() => setPushed(null)}
        />

        <div className="app-body">
          <NavPane
            key={repo.repoPath}
            repoPath={repo.repoPath}
            repoVersion={repoVersion}
            workingTree={workingTree}
            workspaceActions={workspace.actions}
            workspaceFeedback={workspace.feedback}
            onDismissFeedback={workspace.dismissFeedback}
            hostVersion={host.hostRefresh}
            commands={sidebarCommands}
            onPushToCreate={
              requestCreateChange
                ? () => void pushSetUpstream((result) => requestCreateChange(result.branch))
                : undefined
            }
            conflictState={conflictState}
            onBranchChanged={onBranchChanged}
            onFetched={onFetched}
            pruneOnFetch={prefs.pruneOnFetch}
            onOpenHeadCommit={openCommit}
            onOpenChange={openChange}
            onOpenFile={openWorkingFile}
            onCreateChange={requestCreateChange}
            onOpenRepository={repo.openRecent}
            onOpenWorktreeBoard={() => setBoardOpen(true)}
            width={navWidth.width}
            collapsed={navCollapsed}
            onCollapsedChange={setNavCollapsed}
          />
          {!navCollapsed && <ResizeHandle onDragStart={navWidth.onDragStart} ariaLabel="Resize nav pane" />}

          <div className="app-content">
            <main className="app-main" data-pane="main">
              <KeepAlive active={activeTab === 'working-copy'}>
                <ChangesView
                  key={`${repo.repoPath}-${memoryBranch}-${repoVersion}-${nav.fileFocus}`}
                  repoPath={repo.repoPath}
                  workingTree={workingTree}
                  onPushRequested={push}
                  onBranchChanged={onBranchChanged}
                  onCommitted={(result) => {
                    void tasks.runChecklist('commit')
                    if (result) setActionResult({ text: result, undoable: true })
                  }}
                  defaultIgnoreWhitespace={prefs.diffIgnoreWhitespaceDefault}
                  diffExpanded={prefs.diffExpanded}
                  onDiffExpandedChange={settings.setDiffExpanded}
                  initialSelectedPath={restored.selectedFilePath}
                  onSelectedPathChange={(path) => remember({ selectedFilePath: path })}
                  initialDiffScrollTop={restored.diffScrollTop}
                  onDiffScrollChange={(top) => remember({ diffScrollTop: top })}
                  initialCommitDraft={restored.commitDraft}
                  onCommitDraftChange={(message) => remember({ commitDraft: message })}
                  showCommitTypePicker={conventionalCommitsOn}
                  conflict={conflictState}
                  riskRulesOff={prefs.riskRulesOff}
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
                  defaultRef={refFromSaved(prefs.historyBranchFilter)}
                  onRefChange={(ref) => settings.setHistoryBranchFilter(savedFromRef(ref))}
                  firstParent={prefs.historyFirstParent}
                  onFirstParentChange={settings.setHistoryFirstParent}
                  changedFiles={changedCount}
                  onOpenChanges={() => setActiveTab('working-copy')}
                  onBranchChanged={onBranchChanged}
                  previousTopSha={restored.historyTopSha}
                  onTopShaChange={(sha) => remember({ historyTopSha: sha })}
                  onSelectedShaChange={(sha) => remember({ selectedCommitSha: sha })}
                  currentBranch={currentBranch}
                />
              </KeepAlive>
              {changesLabel && (
                <KeepAlive active={activeTab === 'changes'}>
                  <HostChangesView
                    key={`${repo.repoPath}-${host.hostsVersion}`}
                    repoPath={repo.repoPath}
                    label={changesLabel}
                    selectedNumber={nav.changeFocus}
                    onRefreshed={() => {
                      host.collectNotices()
                      host.bumpHostRefresh()
                    }}
                    onBranchChanged={onBranchChanged}
                    onCreateChange={requestCreateChange}
                    onOpenSettings={() => openSettings()}
                    onMerged={(result) => {
                      onBranchChanged()
                      host.bumpHostRefresh()
                      if (result.localBranch) requestSidebar('clean-up')
                    }}
                  />
                </KeepAlive>
              )}
              <KeepAlive active={activeTab === 'changelog'}>
                <ChangelogView
                  key={repo.repoPath}
                  repoPath={repo.repoPath}
                  repoVersion={repoVersion}
                  prefs={settings.settings ? prefs.changelog : null}
                  onPrefsChange={settings.setChangelogPrefs}
                  onOpenCommit={openCommit}
                  changelog={conventionalCommitsOn}
                  onTagged={onBranchChanged}
                />
              </KeepAlive>
              <KeepAlive active={activeTab === 'activity'}>
                <ActivityView
                  key={repo.repoPath}
                  repoPath={repo.repoPath}
                  repoVersion={repoVersion}
                  dirty={changedCount > 0}
                  changedCount={changedCount}
                  hasConflict={conflictState !== null}
                  onStandup={() => setStandupOpen(true)}
                />
              </KeepAlive>
              {!hideExplore && (
                <KeepAlive active={activeTab === 'explore'}>
                  <ExploreView
                    key={repo.repoPath}
                    repoPath={repo.repoPath}
                    currentBranch={currentBranch ?? null}
                    repoVersion={repoVersion}
                    workingVersion={readThroughVersion}
                    onOpenWorkingFile={openWorkingFile}
                  />
                </KeepAlive>
              )}
              {issuesOn && (
                <KeepAlive active={activeTab === 'issues'}>
                  <IssuesView
                    key={`${repo.repoPath}-${host.hostsVersion}`}
                    repoPath={repo.repoPath}
                    selectedKey={issueFocus}
                    onOpenSettings={() => openSettings()}
                    refreshKey={host.hostRefresh}
                    onRefreshed={host.bumpHostRefresh}
                    onBranchChanged={onBranchChanged}
                  />
                </KeepAlive>
              )}
            </main>
            {repo.repoPath && !terminalsOff && (
              <TerminalPanel
                repoPath={repo.repoPath}
                terminals={terminals}
                focusRequest={terminalFocus}
                theme={prefs.theme}
                tasks={
                  <TasksView
                    key={repo.repoPath}
                    repoPath={repo.repoPath}
                    tasks={tasks}
                    refreshKey={readThroughVersion}
                  />
                }
              />
            )}
            {terminals.error && <p className="app-terminal-error">{terminals.error}</p>}
          </div>
          {openIssueKey && (
            <IssuePanel
              repoPath={repo.repoPath}
              issueKey={openIssueKey}
              onClose={() => setOpenIssueKey(null)}
              onBranchChanged={onBranchChanged}
            />
          )}
          {inboxOpen && <div className="inbox-scrim" onClick={() => setInboxOpen(false)} />}
          {inboxOpen && (
            <InboxView
              paths={repo.tabs}
              activePath={repo.repoPath}
              refreshKey={`${repoVersion}:${host.hostRefresh}`}
              onGo={goFromInbox}
              onRefreshed={host.bumpHostRefresh}
              onClose={() => setInboxOpen(false)}
            />
          )}
        </div>

        <StatusBar
          operation={operation}
          result={actionResult}
          onUndo={undoAction?.disabled ? undefined : undoAction?.run}
          onOpenShortcuts={() => setShortcutsOpen(true)}
          onToggleTerminal={terminalsOff ? undefined : toggleTerminal}
          repoPath={repo.repoPath}
          repoVersion={repoVersion}
          gitVersion={gitVersion}
          integration={integration}
          lastFetchedAt={lastFetchedAt}
          onOpenSettings={() => openSettings()}
        />
      </div>
      {repo.repoPath && <NotesDialogs repoPath={repo.repoPath} state={notesState} onOpen={openNote} />}
    </AppProviders>
  )
}

export default App
