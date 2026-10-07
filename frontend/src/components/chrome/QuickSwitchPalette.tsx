import { useState, type ReactNode } from 'react'
import { Command } from 'cmdk'
import { useDebounceValue } from 'usehooks-ts'
import {
  ArrowDown,
  ArrowUp,
  Cloud,
  Code2,
  CornerDownLeft,
  FileText,
  FolderGit2,
  FolderOpen,
  GitBranch,
  GitCommitVertical,
  Search,
  Settings as SettingsIcon,
  Terminal,
  type LucideIcon,
} from 'lucide-react'
import {
  BranchService,
  HistoryService,
  PlatformService,
  RemoteService,
  StatusService,
} from '@current-client-bindings/app'
import { errorMessage } from '../../lib/errors'
import { highlightMatch } from '../../lib/highlightMatch'
import { localNameFor } from '../../features/branches/branches'
import { useLaneColors } from '../../lib/laneColor'
import { baseName } from '../../lib/paths'
import { relativeTime } from '../../lib/relativeTime'
import { useAsyncData } from '../../lib/useAsyncData'
import { useBranchStatus } from '../../lib/useBranchStatus'
import type { useRepositoryLifecycle } from '../../features/repositories/useRepositoryLifecycle'
import Kbd from '../controls/Kbd'
import PathText from '../git/PathText'
import Modal from './Modal'
import './QuickSwitchPalette.scss'

interface QuickSwitchPaletteProps {
  repoPath: string
  repo: ReturnType<typeof useRepositoryLifecycle>
  onClose: () => void
  onBranchChanged?: () => void
  onOpenSettings: () => void
  onPull: () => void
  onPullMerge: () => void
  onPullRebase: () => void
  onPush: () => void
  onFetchAll: () => void
  onOpenFile: (path: string) => void
  onSearchFiles: () => void
  onOpenCommit: (sha: string, position: number) => void
}

interface BranchResult {
  name: string
  // What checking it out uses: the local branch, or the one a remote branch would track as.
  checkoutName: string
  remote: boolean
  current: boolean
}

interface RepoResult {
  path: string
  uncommitted: number
  currentBranch: string
}

interface Action {
  id: string
  label: string
  icon: LucideIcon
  run: () => void
}

const SETTINGS_ITEMS = [
  { id: 'theme', label: 'Theme', hint: 'Light, dark, or system' },
  { id: 'git-path', label: 'Git executable path', hint: 'Settings' },
  { id: 'repo-location', label: 'Default repository location', hint: 'Settings' },
  { id: 'diff', label: 'Diff', hint: 'Ignore whitespace by default' },
  { id: 'editor', label: 'Editor', hint: 'Settings' },
  { id: 'auto-fetch', label: 'Background auto-fetch', hint: 'Settings' },
  { id: 'lane-colors', label: 'Lane colors', hint: 'Settings' },
  { id: 'identity', label: 'Identity & global user', hint: 'Settings' },
]

const MAX_FILE_RESULTS = 20
const MAX_COMMIT_RESULTS = 10
const COMMIT_SEARCH_DEBOUNCE_MS = 200

function loadBranchResults(repoPath: string): Promise<BranchResult[]> {
  return Promise.all([
    BranchService.ListLocal(repoPath),
    BranchService.ListRemote(repoPath),
    RemoteService.List(repoPath),
  ])
    .then(([local, remote, remotes]) => {
      const remoteNames = remotes.map((r) => r.name)
      return [
        ...local.map((b) => ({ name: b.name, checkoutName: b.name, remote: false, current: b.current })),
        ...remote.map((name) => ({
          name,
          checkoutName: localNameFor(name, remoteNames),
          remote: true,
          current: false,
        })),
      ]
    })
    .catch(() => [])
}

function loadRepoResults(paths: string[]): Promise<RepoResult[]> {
  return Promise.all(
    paths.map((path) =>
      Promise.all([StatusService.GetStatus(path), BranchService.CurrentBranchStatus(path)])
        .then(([status, branch]) => ({ path, uncommitted: status.length, currentBranch: branch.current }))
        .catch(() => ({ path, uncommitted: 0, currentBranch: '' })),
    ),
  )
}

function matches(text: string, query: string): boolean {
  return text.toLowerCase().includes(query)
}

interface SectionProps {
  heading: string
  icon?: LucideIcon
  count: number
  children: ReactNode
}

function Section({ heading, icon: Icon, count, children }: SectionProps) {
  if (count === 0) return null
  return (
    <Command.Group
      className="quick-switch-section"
      heading={
        <div className="quick-switch-section-header">
          <span className="quick-switch-label">
            {Icon && <Icon size={12} strokeWidth={1.75} />}
            {heading}
          </span>
          <span className="quick-switch-count">{count}</span>
        </div>
      }
    >
      {children}
    </Command.Group>
  )
}

function QuickSwitchPalette({
  repoPath,
  repo,
  onClose,
  onBranchChanged,
  onOpenSettings,
  onPull,
  onPullMerge,
  onPullRebase,
  onPush,
  onFetchAll,
  onOpenFile,
  onSearchFiles,
  onOpenCommit,
}: QuickSwitchPaletteProps) {
  const { branchColor } = useLaneColors()
  const [query, setQuery] = useState('')
  const [error, setError] = useState<string | null>(null)
  const q = query.trim().toLowerCase()
  const [debouncedQuery] = useDebounceValue(q, COMMIT_SEARCH_DEBOUNCE_MS)

  const ahead = useBranchStatus(repoPath, 0)?.ahead ?? 0
  const branches = useAsyncData(() => loadBranchResults(repoPath), [repoPath]).data ?? []
  const files = useAsyncData(() => StatusService.ListFiles(repoPath).catch(() => []), [repoPath]).data ?? []
  const repos =
    useAsyncData(
      () => loadRepoResults(repo.recentRepos.filter((path) => path !== repoPath)),
      [repoPath, repo.recentRepos],
    ).data ?? []
  const searchResults = useAsyncData(
    () =>
      debouncedQuery
        ? HistoryService.SearchHistory(repoPath, debouncedQuery, MAX_COMMIT_RESULTS).catch(() => [])
        : null,
    [repoPath, debouncedQuery],
    { keepData: true },
  ).data
  const commits = q ? (searchResults ?? []) : []

  const actions: Action[] = [
    { id: 'search-files', label: 'Search in files', icon: Search, run: onSearchFiles },
    { id: 'fetch-all', label: 'Fetch all remotes', icon: Cloud, run: onFetchAll },
    { id: 'pull', label: 'Pull', icon: ArrowDown, run: onPull },
    { id: 'pull-merge', label: 'Pull (merge)', icon: ArrowDown, run: onPullMerge },
    { id: 'pull-rebase', label: 'Pull (rebase)', icon: ArrowDown, run: onPullRebase },
    { id: 'push', label: 'Push', icon: ArrowUp, run: onPush },
    { id: 'open-terminal', label: 'Open Terminal', icon: Terminal, run: () => PlatformService.OpenTerminal(repoPath) },
    {
      id: 'reveal-file-manager',
      label: 'Reveal in File Explorer',
      icon: FolderOpen,
      run: () => PlatformService.RevealInFileManager(repoPath),
    },
    { id: 'open-editor', label: 'Open in Editor', icon: Code2, run: () => PlatformService.OpenInEditor(repoPath) },
    { id: 'open-settings', label: 'Open Settings', icon: SettingsIcon, run: onOpenSettings },
  ]

  const branchResults = branches.filter((b) => matches(b.name, q))
  const repoResults = repos.filter((r) => matches(r.path, q))
  // A large repo has thousands of files, so they only appear once there's a query.
  const fileResults = q ? files.filter((p) => matches(p, q)).slice(0, MAX_FILE_RESULTS) : []
  const settingsResults = SETTINGS_ITEMS.filter((s) => matches(s.label, q))
  const actionResults = actions.filter((a) => matches(a.label, q))
  const resultCount =
    branchResults.length +
    repoResults.length +
    fileResults.length +
    commits.length +
    settingsResults.length +
    actionResults.length

  const runAndClose = (run: () => void) => {
    run()
    onClose()
  }

  const checkout = (branch: BranchResult) => {
    if (branch.current) {
      onClose()
      return
    }
    setError(null)
    BranchService.CheckoutBranch(repoPath, branch.checkoutName)
      .then(() => runAndClose(() => onBranchChanged?.()))
      .catch((err: unknown) => setError(errorMessage(err)))
  }

  const openCommit = (sha: string) => {
    setError(null)
    HistoryService.GetCommitPosition(repoPath, sha)
      .then((position) => runAndClose(() => onOpenCommit(sha, position)))
      .catch((err: unknown) => setError(errorMessage(err)))
  }

  const highlight = (text: string) => highlightMatch(text, query, 'search-match')

  return (
    <Modal title="Command palette" onClose={onClose} className="quick-switch-palette" placement="top" hideHeader>
      <Command shouldFilter={false} loop className="quick-switch-command">
        <div className="quick-switch-search">
          <Search size={16} strokeWidth={1.5} />
          <Command.Input
            autoFocus
            aria-label="Search branches, repositories, files, commits, settings, and actions"
            placeholder="Search branches, files, commits, settings, actions…"
            value={query}
            onValueChange={setQuery}
          />
          <Kbd>ESC</Kbd>
        </div>

        {error && <p className="quick-switch-error">{error}</p>}

        <Command.List className="quick-switch-results">
          <Command.Empty className="quick-switch-empty">No matches.</Command.Empty>

          <Section heading={`Branches in ${baseName(repoPath)}`} icon={GitBranch} count={branchResults.length}>
            {branchResults.map((b) => (
              <Command.Item
                key={`branch:${b.name}`}
                value={`branch:${b.name}`}
                aria-label={b.current ? `${b.name}, current branch` : b.remote ? `${b.name}, remote branch` : b.name}
                className="quick-switch-row"
                onSelect={() => checkout(b)}
              >
                {b.remote ? (
                  <Cloud size={14} strokeWidth={1.5} className="quick-switch-row-icon" />
                ) : (
                  <span className="quick-switch-dot" style={{ background: branchColor(b.name, b.current) }} />
                )}
                <span className="quick-switch-row-name">{highlight(b.name)}</span>
                <span className="quick-switch-row-meta">
                  {b.current && <span className="quick-switch-head-badge">HEAD</span>}
                  {b.current && <span className="quick-switch-current-label">current</span>}
                  {b.current && ahead > 0 && <span className="quick-switch-ahead">↑{ahead}</span>}
                </span>
              </Command.Item>
            ))}
          </Section>

          <Section heading="Repositories" icon={FolderGit2} count={repoResults.length}>
            {repoResults.map((r) => (
              <Command.Item
                key={`repo:${r.path}`}
                value={`repo:${r.path}`}
                aria-label={`${baseName(r.path)}, ${r.uncommitted > 0 ? `${r.uncommitted} uncommitted` : 'clean'}${r.currentBranch ? `, ${r.currentBranch}` : ''}`}
                className="quick-switch-row"
                onSelect={() => runAndClose(() => repo.openRecent(r.path))}
              >
                <FolderGit2 size={14} strokeWidth={1.5} className="quick-switch-row-icon" />
                <span className="quick-switch-row-name">{highlight(baseName(r.path))}</span>
                <PathText path={r.path} className="quick-switch-path" />
                <span className="quick-switch-row-meta">
                  <span className={`quick-switch-dot quick-switch-dot-${r.uncommitted > 0 ? 'dirty' : 'clean'}`} />
                  <span className={r.uncommitted > 0 ? 'quick-switch-dirty' : 'quick-switch-clean'}>
                    {r.uncommitted > 0 ? `${r.uncommitted} uncommitted` : 'clean'}
                  </span>
                  {r.currentBranch && <span className="quick-switch-repo-branch">{r.currentBranch}</span>}
                </span>
              </Command.Item>
            ))}
          </Section>

          <Section heading="Files" icon={FileText} count={fileResults.length}>
            {fileResults.map((path) => (
              <Command.Item
                key={`file:${path}`}
                value={`file:${path}`}
                aria-label={`File history of ${path}`}
                className="quick-switch-row"
                onSelect={() => runAndClose(() => onOpenFile(path))}
              >
                <FileText size={14} strokeWidth={1.5} className="quick-switch-row-icon" />
                <span className="quick-switch-row-name">{highlight(baseName(path))}</span>
                <PathText path={path} className="quick-switch-path" />
              </Command.Item>
            ))}
          </Section>

          <Section heading="Commits" icon={GitCommitVertical} count={commits.length}>
            {commits.map((c) => (
              <Command.Item
                key={`commit:${c.sha}`}
                value={`commit:${c.sha}`}
                aria-label={`Commit ${c.sha.slice(0, 7)}: ${c.subject}`}
                className="quick-switch-row"
                onSelect={() => openCommit(c.sha)}
              >
                <GitCommitVertical size={14} strokeWidth={1.5} className="quick-switch-row-icon" />
                <span className="quick-switch-row-name">{highlight(c.subject)}</span>
                <span className="quick-switch-row-meta">
                  <span className="quick-switch-commit-sha">{c.sha.slice(0, 7)}</span>
                  <span className="quick-switch-commit-date">{relativeTime(new Date(c.date))}</span>
                </span>
              </Command.Item>
            ))}
          </Section>

          <Section heading="Settings" icon={SettingsIcon} count={settingsResults.length}>
            {settingsResults.map((s) => (
              <Command.Item
                key={`settings:${s.id}`}
                value={`settings:${s.id}`}
                aria-label={`Settings: ${s.label}`}
                className="quick-switch-row"
                onSelect={() => runAndClose(onOpenSettings)}
              >
                <SettingsIcon size={14} strokeWidth={1.5} className="quick-switch-row-icon" />
                <span className="quick-switch-row-name">{highlight(s.label)}</span>
                <span className="quick-switch-row-meta">{s.hint}</span>
              </Command.Item>
            ))}
          </Section>

          <Section heading="Actions" count={actionResults.length}>
            {actionResults.map((a) => (
              <Command.Item
                key={`action:${a.id}`}
                value={`action:${a.id}`}
                aria-label={a.label}
                className="quick-switch-row"
                onSelect={() => runAndClose(a.run)}
              >
                <a.icon size={14} strokeWidth={1.5} className="quick-switch-row-icon" />
                <span className="quick-switch-row-name">{highlight(a.label)}</span>
              </Command.Item>
            ))}
          </Section>
        </Command.List>

        <div className="quick-switch-footer">
          <span className="quick-switch-hint">
            <ArrowUp size={12} strokeWidth={1.5} />
            <ArrowDown size={12} strokeWidth={1.5} /> navigate
          </span>
          <span className="quick-switch-hint">
            <CornerDownLeft size={12} strokeWidth={1.5} /> select
          </span>
          <span className="quick-switch-result-count">{resultCount} results</span>
        </div>
      </Command>
    </Modal>
  )
}

export default QuickSwitchPalette
