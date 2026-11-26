import {
  BookOpen,
  Broom,
  ChartColumn,
  CircleDot,
  ClipboardCheck,
  ClipboardList,
  Cloud,
  Columns3,
  FileDiff,
  FolderGit2,
  GitBranchPlus,
  GitCommitVertical,
  GitPullRequest,
  Grid3x3,
  Inbox,
  Keyboard,
  Layers,
  ListChecks,
  ListOrdered,
  Map as MapIcon,
  Package,
  PackagePlus,
  Play,
  StickyNote,
  Table,
  Tag,
} from 'lucide-react'
import { BranchService } from '@current-client-bindings/app'
import type { AppTab } from '../repositories/TabBar'
import type { useTasks } from '../tasks/useTasks'
import type { SidebarRequest } from './NavPane'
import type { PaletteAction } from './paletteActions'

interface AppPaletteOptions {
  repoPath: string
  currentBranch: string | undefined
  changesLabel: string | null
  changeKind: string
  issuesOn: boolean
  hostOn: boolean
  hideExplore: boolean
  terminalsOff: boolean
  tasks: ReturnType<typeof useTasks>
  viewShortcut: (tab: AppTab) => string
  changeTab: (tab: AppTab) => void
  requestSidebar: (action: SidebarRequest['action']) => void
  createIssue: (() => void) | undefined
  requestCreateChange: ((branch: string) => void) | undefined
  openShortcuts: () => void
  openReadThrough: (ref: string) => void
  openInbox: () => void
  openStandup: () => void
  openTasks: () => void
  openNotes: () => void
  openWorktreeBoard: () => void
}

// Go to anything's own entries, beside the workspace actions: the views, the
// create forms and the tools, each only while it can run.
export function appPaletteActions({
  repoPath,
  currentBranch,
  changesLabel,
  changeKind,
  issuesOn,
  hostOn,
  hideExplore,
  terminalsOff,
  tasks,
  viewShortcut,
  changeTab,
  requestSidebar,
  createIssue,
  requestCreateChange,
  openShortcuts,
  openReadThrough,
  openInbox,
  openStandup,
  openTasks,
  openNotes,
  openWorktreeBoard,
}: AppPaletteOptions): PaletteAction[] {
  return [
    {
      id: 'keyboard-shortcuts',
      label: 'Keyboard shortcuts',
      section: 'Views',
      icon: Keyboard,
      keywords: ['keys', 'help', 'hotkeys'],
      shortcut: '?',
      run: openShortcuts,
    },
    {
      id: 'view-changes',
      label: 'Changes',
      shortcut: viewShortcut('working-copy'),
      section: 'Views',
      icon: FileDiff,
      keywords: ['working tree', 'staged', 'diff', 'commit', 'stage', 'discard', 'amend'],
      run: () => changeTab('working-copy'),
    },
    {
      id: 'view-history',
      label: 'History',
      shortcut: viewShortcut('history'),
      section: 'Views',
      icon: GitCommitVertical,
      keywords: ['log', 'graph', 'commits', 'cherry-pick', 'revert', 'reset', 'blame', 'rebase'],
      run: () => changeTab('history'),
    },
    ...(currentBranch
      ? [
          {
            id: 'read-through',
            label: `Read through ${currentBranch}`,
            section: 'Tools' as const,
            icon: BookOpen,
            keywords: ['review', 'diff', 'files', 'changes', 'notes', 'read'],
            run: () => openReadThrough(`refs/heads/${currentBranch}`),
          },
        ]
      : []),
    ...(changesLabel
      ? [
          {
            id: 'view-host-changes',
            shortcut: viewShortcut('changes'),
            label: changesLabel,
            section: 'Views' as const,
            icon: GitPullRequest,
            keywords: ['pr', 'mr', 'pull requests', 'merge requests', 'review'],
            run: () => changeTab('changes'),
          },
        ]
      : []),
    ...(issuesOn
      ? [
          {
            id: 'view-issues',
            shortcut: viewShortcut('issues'),
            label: 'Issues',
            section: 'Views' as const,
            icon: CircleDot,
            keywords: ['assigned', 'my issues', 'jira', 'tickets'],
            run: () => changeTab('issues'),
          },
        ]
      : []),
    {
      id: 'view-activity',
      shortcut: viewShortcut('activity'),
      label: 'Activity',
      section: 'Views',
      icon: ChartColumn,
      keywords: ['stats', 'contributors', 'health'],
      run: () => changeTab('activity'),
    },
    ...(hideExplore
      ? []
      : [
          {
            id: 'view-explore',
            shortcut: viewShortcut('explore'),
            label: 'Explore',
            section: 'Views' as const,
            icon: MapIcon,
            keywords: ['code map', 'treemap', 'folders', 'files', 'structure', 'changes'],
            run: () => changeTab('explore'),
          },
        ]),
    {
      id: 'view-releases',
      shortcut: viewShortcut('changelog'),
      label: 'Releases',
      section: 'Views',
      icon: Tag,
      keywords: ['changelog', 'release notes', 'next release', 'version'],
      run: () => changeTab('changelog'),
    },
    {
      id: 'view-inbox',
      label: 'Inbox',
      section: 'Views',
      icon: Inbox,
      keywords: ['todo', 'to-do', 'notifications', 'waiting', 'review requests'],
      run: openInbox,
    },
    {
      id: 'new-branch',
      label: 'New branch…',
      section: 'Create',
      icon: GitBranchPlus,
      keywords: ['create branch', 'from an issue'],
      run: () => requestSidebar('new-branch'),
    },
    ...(createIssue
      ? [
          {
            id: 'new-issue',
            label: 'New issue…',
            section: 'Create' as const,
            icon: CircleDot,
            keywords: ['create issue'],
            run: () => createIssue(),
          },
        ]
      : []),
    ...(requestCreateChange
      ? [
          {
            id: 'new-change',
            label: `New ${changeKind}…`,
            section: 'Create' as const,
            icon: GitPullRequest,
            keywords: ['pr', 'mr', 'create pull request', 'create merge request'],
            run: () =>
              void BranchService.CurrentBranchStatus(repoPath).then((s) => s.current && requestCreateChange(s.current)),
          },
        ]
      : []),
    {
      id: 'new-stash',
      label: 'Stash changes',
      section: 'Create',
      icon: Package,
      keywords: ['stash', 'shelve', 'save'],
      run: () => requestSidebar('new-stash'),
    },
    {
      id: 'new-worktree',
      label: 'Add worktree',
      section: 'Create',
      icon: FolderGit2,
      keywords: ['worktree'],
      run: () => requestSidebar('new-worktree'),
    },
    {
      id: 'new-remote',
      label: 'Add remote',
      section: 'Create',
      icon: Cloud,
      keywords: ['remote', 'origin'],
      run: () => requestSidebar('new-remote'),
    },
    {
      id: 'new-tag',
      label: 'New tag…',
      section: 'Create',
      icon: Tag,
      keywords: ['create tag'],
      run: () => requestSidebar('new-tag'),
    },
    {
      id: 'release-readiness',
      label: 'Release readiness',
      section: 'Release',
      icon: ClipboardCheck,
      keywords: ['release', 'ready', 'version', 'changelog'],
      run: () => changeTab('changelog'),
    },
    ...(hostOn
      ? [
          {
            id: 'publish-release',
            label: 'Publish release…',
            section: 'Release' as const,
            icon: PackagePlus,
            keywords: ['release', 'publish'],
            run: () => requestSidebar('publish-release'),
          },
        ]
      : []),
    {
      id: 'standup',
      label: 'Standup…',
      section: 'Tools',
      icon: ClipboardList,
      keywords: ['what i did', 'summary', 'yesterday', 'this week'],
      run: openStandup,
    },
    ...(terminalsOff
      ? []
      : [
          {
            id: 'tasks-panel',
            label: 'Tasks',
            section: 'Tools' as const,
            icon: ListChecks,
            keywords: ['run', 'test', 'build', 'lint', 'make', 'npm', 'commands'],
            run: openTasks,
          },
          ...(tasks.list?.tasks ?? []).map((task) => ({
            id: `task-${task.id}`,
            label: `Run ${task.name}`,
            section: 'Tools' as const,
            icon: Play,
            keywords: [task.command, task.source],
            run: () => {
              openTasks()
              void tasks.run(task)
            },
          })),
        ]),
    {
      id: 'restack',
      label: 'Restack current stack',
      section: 'Tools',
      icon: Layers,
      keywords: ['stacked branches', 'rebase'],
      run: () => requestSidebar('restack'),
    },
    {
      id: 'tidy-up',
      label: 'Tidy up history',
      section: 'Tools',
      icon: ListOrdered,
      keywords: ['interactive rebase', 'reorder', 'squash', 'fixup', 'reword', 'drop'],
      run: () => requestSidebar('tidy-up'),
    },
    {
      id: 'notes',
      label: 'Notes',
      section: 'Tools',
      icon: StickyNote,
      keywords: ['annotations', 'private'],
      run: openNotes,
    },
    {
      id: 'branch-review',
      label: 'Branch review',
      section: 'Tools',
      icon: Table,
      keywords: ['branches', 'stale', 'unpushed', 'bulk delete', 'grid'],
      run: () => requestSidebar('branch-review'),
    },
    {
      id: 'worktree-board',
      label: 'Worktree board',
      section: 'Tools',
      icon: Columns3,
      keywords: ['worktrees', 'parallel', 'agents', 'collide', 'conflict'],
      run: openWorktreeBoard,
    },
    {
      id: 'conflict-grid',
      label: 'Conflict grid',
      section: 'Tools',
      icon: Grid3x3,
      keywords: ['branches', 'merge', 'overlap', 'predict', 'pairs'],
      run: () => requestSidebar('conflict-grid'),
    },
    {
      id: 'clean-up-branches',
      label: 'Clean up merged branches',
      section: 'Tools',
      icon: Broom,
      keywords: ['delete merged', 'tidy', 'stale'],
      run: () => requestSidebar('clean-up'),
    },
  ]
}
