import { useLayoutEffect, useRef, useState, type ReactNode } from 'react'
import { Command } from 'cmdk'
import { useDebounceValue } from 'usehooks-ts'
import {
  ArrowDown,
  ArrowUp,
  Cloud,
  CornerDownLeft,
  FileText,
  FolderGit2,
  GitBranch,
  GitCommitVertical,
  GitPullRequest,
  Package,
  Search,
  Settings as SettingsIcon,
  StickyNote,
  Tag,
  type LucideIcon,
} from 'lucide-react'
import {
  BranchService,
  HistoryService,
  HostService,
  StashService,
  type NoteInfo,
  RemoteService,
  StatusService,
  TagService,
} from '@current-client-bindings/app'
import { errorMessage } from '../../lib/errors'
import { noteLabel } from '../../features/notes/NotesDialogs'
import { useNotes } from '../../features/notes/notes'
import { highlightMatch } from '../../lib/highlightMatch'
import { localNameFor } from '../../features/branches/branches'
import { useLaneColors } from '../../lib/laneColor'
import { baseName } from '../../lib/paths'
import { relativeTime } from '../../lib/relativeTime'
import { useClockTick } from '../../lib/useClockTick'
import { useAsyncData } from '../../lib/useAsyncData'
import { useBranchStatus } from '../../lib/useBranchStatus'
import type { useRepositoryLifecycle } from '../../features/repositories/useRepositoryLifecycle'
import Kbd from '../controls/Kbd'
import PathText from '../git/PathText'
import Modal from './Modal'
import { shortcutLabel } from '../../lib/shortcuts'
import { plural } from '../../lib/plural'
import { PALETTE_SECTIONS, matchesAction, type PaletteAction } from './paletteActions'
import type { WorkspaceAction } from './useWorkspaceActions'
import './QuickSwitchPalette.scss'

interface QuickSwitchPaletteProps {
  repoPath: string
  repo: ReturnType<typeof useRepositoryLifecycle>
  onClose: () => void
  onBranchChanged?: () => void
  workspaceActions: WorkspaceAction[]
  // The views, create, release and tool entries the app owns; the palette adds sync and Settings.
  actions?: PaletteAction[]
  // Each Settings group is an entry that opens Settings on it.
  settingsGroups?: string[]
  onOpenSettings: (group?: string) => void
  onPull: () => void
  onPullMerge: () => void
  onPullRebase: () => void
  onPush: () => void
  onFetchAll: () => void
  onOpenFile: (path: string) => void
  onBlameFile?: (path: string) => void
  onOpenChange?: (number: number) => void
  onShowStashes?: () => void
  onSearchFiles: () => void
  onOpenCommit: (sha: string, position: number) => void
  onOpenNote?: (note: NoteInfo) => void
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

// What each Settings group holds, shown beside it and searched with its name.
const SETTINGS_HINTS: Record<string, string> = {
  Appearance: 'Theme, lane colors',
  Git: 'Git executable, Conventional Commits',
  'Tools and folders': 'Editor, default repository location',
  Diff: 'Ignore whitespace',
  Network: 'Background fetch, prune on fetch',
  Integrations: 'Hosts, issue trackers, tokens',
  'Identity colors': 'Badge colors',
  Support: 'Version, check for updates, diagnostics',
}

const SETTINGS_KEYWORDS: Record<string, string> = {
  Appearance: 'dark light system',
  Git: 'path',
  'Tools and folders': 'terminal folder',
  Network: 'auto-fetch',
  Integrations: 'github gitlab jira token releases url',
  'Identity colors': 'initials avatar',
  Support: 'update logs about',
}

const MAX_FILE_RESULTS = 20
// At rest the palette shows where you can go, not an inventory: this many
// local branches, the recent repositories and the actions.
const MAX_IDLE_BRANCHES = 4
const MAX_COMMIT_RESULTS = 10
const MAX_TAG_RESULTS = 10
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
  workspaceActions,
  actions: appActions = [],
  settingsGroups = [],
  onOpenSettings,
  onPull,
  onPullMerge,
  onPullRebase,
  onPush,
  onFetchAll,
  onOpenFile,
  onBlameFile,
  onOpenChange,
  onShowStashes,
  onSearchFiles,
  onOpenCommit,
  onOpenNote,
}: QuickSwitchPaletteProps) {
  useClockTick()
  const { branchColor } = useLaneColors()
  const [query, setQuery] = useState('')
  const [error, setError] = useState<string | null>(null)
  const q = query.trim().toLowerCase()
  const [debouncedQuery] = useDebounceValue(q, COMMIT_SEARCH_DEBOUNCE_MS)

  const ahead = useBranchStatus(repoPath, 0)?.ahead ?? 0
  const branches = useAsyncData(() => loadBranchResults(repoPath), [repoPath]).data ?? []
  const files = useAsyncData(() => StatusService.ListFiles(repoPath).catch(() => []), [repoPath]).data ?? []
  const tags = useAsyncData(() => TagService.ListTags(repoPath).catch(() => []), [repoPath]).data ?? []
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
  const hostChanges = useAsyncData(() => HostService.Changes(repoPath).catch(() => null), [repoPath]).data
  const changeResults = q
    ? (hostChanges?.changes ?? [])
        .filter((c) => matches(`${hostChanges?.prefix}${c.number} ${c.title}`, q))
        .slice(0, MAX_TAG_RESULTS)
    : []
  const stashes = useAsyncData(() => StashService.ListStash(repoPath).catch(() => []), [repoPath]).data ?? []
  const stashResults = q ? stashes.filter((s) => matches(s.message, q)) : []
  const blameQuery = q.startsWith('blame') ? q.replace(/^blame\s*/, '') : null
  const blameResults =
    blameQuery !== null && onBlameFile ? files.filter((p) => matches(p, blameQuery)).slice(0, MAX_FILE_RESULTS) : []
  const notes = useNotes()
  const noteResults = q
    ? notes.notes.filter((note) => matches(note.text, q) || matches(note.target, q) || matches(note.subject, q))
    : []

  const actions: PaletteAction[] = [
    ...appActions,
    { id: 'fetch-all', label: 'Fetch all remotes', section: 'Sync', icon: Cloud, run: onFetchAll },
    { id: 'pull', label: 'Pull', section: 'Sync', icon: ArrowDown, run: onPull },
    { id: 'pull-merge', label: 'Pull (merge)', section: 'Sync', icon: ArrowDown, run: onPullMerge },
    { id: 'pull-rebase', label: 'Pull (rebase)', section: 'Sync', icon: ArrowDown, run: onPullRebase },
    { id: 'push', label: 'Push', section: 'Sync', icon: ArrowUp, run: onPush },
    {
      id: 'search-files',
      label: 'Search in files',
      section: 'Tools',
      icon: Search,
      keywords: ['find', 'grep'],
      run: onSearchFiles,
    },
    ...workspaceActions.filter((a) => !a.disabled).map((a) => ({ ...a, section: 'Tools' as const })),
    {
      id: 'open-settings',
      label: 'Open Settings',
      section: 'Settings',
      icon: SettingsIcon,
      run: () => onOpenSettings(),
    },
  ]

  const branchResults = q
    ? branches.filter((b) => matches(b.name, q))
    : branches
        .filter((b) => !b.remote)
        .sort((a, b) => Number(b.current) - Number(a.current))
        .slice(0, MAX_IDLE_BRANCHES)
  const repoResults = repos.filter((r) => matches(r.path, q))
  // A large repo has thousands of files, so they only appear once there's a query.
  const fileResults = q ? files.filter((p) => matches(p, q)).slice(0, MAX_FILE_RESULTS) : []
  const tagResults = q ? tags.filter((t) => matches(t.name, q)).slice(0, MAX_TAG_RESULTS) : []
  const settingsResults = q
    ? settingsGroups.filter((g) => [g, SETTINGS_HINTS[g] ?? '', SETTINGS_KEYWORDS[g] ?? ''].some((t) => matches(t, q)))
    : []
  const actionResults = actions.filter((a) => matchesAction(a, q))
  const sectionActions = (section: PaletteAction['section']) => actionResults.filter((a) => a.section === section)
  const resultCount =
    branchResults.length +
    repoResults.length +
    fileResults.length +
    commits.length +
    tagResults.length +
    settingsResults.length +
    actionResults.length +
    changeResults.length +
    stashResults.length +
    blameResults.length

  // The list fills in as searches return, so the first row can change after
  // it opens; the selection follows the top of the list until the arrows move it.
  const commandRef = useRef<HTMLDivElement>(null)
  const [selectedValue, setSelectedValue] = useState('')
  useLayoutEffect(() => {
    const first = commandRef.current?.querySelector('[cmdk-item]')?.getAttribute('data-value')
    setSelectedValue(first ?? '')
  }, [q, resultCount])

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

  const actionRow = (a: PaletteAction) => (
    <Command.Item
      key={`action:${a.id}`}
      value={`action:${a.id}`}
      aria-label={a.label}
      className="quick-switch-row quick-switch-row-action"
      onSelect={() => runAndClose(a.run)}
    >
      <a.icon size={14} strokeWidth={1.5} className="quick-switch-row-icon" />
      <span className="quick-switch-row-name">{highlight(a.label)}</span>
      {a.shortcut && <Kbd>{shortcutLabel(a.shortcut)}</Kbd>}
    </Command.Item>
  )

  return (
    <Modal title="Go to anything" onClose={onClose} className="quick-switch-palette" placement="top" hideHeader>
      <Command
        ref={commandRef}
        shouldFilter={false}
        loop
        className="quick-switch-command"
        value={selectedValue}
        onValueChange={setSelectedValue}
      >
        <div className="quick-switch-search">
          <Search size={16} strokeWidth={1.5} />
          <Command.Input
            autoFocus
            aria-label="Search branches, repositories, files, commits, tags, settings, and actions"
            placeholder="Go to a branch, file, commit, #pull request, tag or action…"
            value={query}
            onValueChange={setQuery}
          />
          <Kbd>ESC</Kbd>
        </div>

        {error && <p className="quick-switch-error">{error}</p>}

        <Command.List className="quick-switch-results">
          {q && resultCount === 0 && <p className="quick-switch-empty">Nothing named “{query.trim()}”.</p>}

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
                <PathText path={r.path.slice(0, -baseName(r.path).length - 1)} className="quick-switch-path" />
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

          <Section
            heading={hostChanges?.prefix === '!' ? 'Merge requests' : 'Pull requests'}
            icon={GitPullRequest}
            count={changeResults.length}
          >
            {changeResults.map((c) => (
              <Command.Item
                key={`change:${c.number}`}
                value={`change:${c.number}`}
                aria-label={`${hostChanges?.prefix}${c.number}: ${c.title}`}
                className="quick-switch-row quick-switch-row-action"
                onSelect={() => runAndClose(() => onOpenChange?.(c.number))}
              >
                <GitPullRequest size={14} strokeWidth={1.5} className="quick-switch-row-icon" />
                <span className="quick-switch-row-name">{highlight(c.title)}</span>
                <span className="quick-switch-commit-sha">
                  {hostChanges?.prefix}
                  {c.number}
                </span>
              </Command.Item>
            ))}
          </Section>

          <Section heading="Stashes" icon={Package} count={stashResults.length}>
            {stashResults.map((s) => (
              <Command.Item
                key={`stash:${s.index}`}
                value={`stash:${s.index}`}
                aria-label={`Stash: ${s.message}`}
                className="quick-switch-row quick-switch-row-action"
                onSelect={() => runAndClose(() => onShowStashes?.())}
              >
                <Package size={14} strokeWidth={1.5} className="quick-switch-row-icon" />
                <span className="quick-switch-row-name">{highlight(s.message)}</span>
              </Command.Item>
            ))}
          </Section>

          <Section heading="Blame" icon={FileText} count={blameResults.length}>
            {blameResults.map((path) => (
              <Command.Item
                key={`blame:${path}`}
                value={`blame:${path}`}
                aria-label={`Blame ${path}`}
                className="quick-switch-row"
                onSelect={() => runAndClose(() => onBlameFile?.(path))}
              >
                <FileText size={14} strokeWidth={1.5} className="quick-switch-row-icon" />
                <span className="quick-switch-row-name">{baseName(path)}</span>
                <PathText path={path} className="quick-switch-path" />
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
                className="quick-switch-row quick-switch-row-action"
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

          <Section heading="Notes" icon={StickyNote} count={noteResults.length}>
            {noteResults.map((note) => (
              <Command.Item
                key={`note:${note.kind}:${note.target}`}
                value={`note:${note.kind}:${note.target}`}
                aria-label={`Note on ${noteLabel(note)}: ${note.text}`}
                className="quick-switch-row"
                onSelect={() => {
                  onClose()
                  onOpenNote?.(note)
                }}
              >
                <StickyNote size={14} strokeWidth={1.5} className="quick-switch-row-icon" />
                <span className="quick-switch-row-name">{highlight(note.text)}</span>
                <span className="quick-switch-row-meta">
                  {note.stale ? 'Gone · ' : ''}
                  {noteLabel(note)}
                </span>
              </Command.Item>
            ))}
          </Section>

          <Section heading="Tags" icon={Tag} count={tagResults.length}>
            {tagResults.map((t) => (
              <Command.Item
                key={`tag:${t.name}`}
                value={`tag:${t.name}`}
                aria-label={`Tag ${t.name}`}
                className="quick-switch-row"
                onSelect={() => openCommit(t.sha)}
              >
                <Tag size={14} strokeWidth={1.5} className="quick-switch-row-icon" />
                <span className="quick-switch-row-name">{highlight(t.name)}</span>
                <span className="quick-switch-row-meta">
                  <span className="quick-switch-commit-sha">{t.sha.slice(0, 7)}</span>
                  {t.date && <span className="quick-switch-commit-date">{relativeTime(new Date(t.date))}</span>}
                </span>
              </Command.Item>
            ))}
          </Section>

          {PALETTE_SECTIONS.map((section) => (
            <Section
              key={section}
              heading={section}
              count={sectionActions(section).length + (section === 'Settings' ? settingsResults.length : 0)}
            >
              {sectionActions(section).map(actionRow)}
              {section === 'Settings' &&
                settingsResults.map((g) => (
                  <Command.Item
                    key={`settings:${g}`}
                    value={`settings:${g}`}
                    aria-label={`Settings: ${g}`}
                    className="quick-switch-row"
                    onSelect={() => runAndClose(() => onOpenSettings(g))}
                  >
                    <SettingsIcon size={14} strokeWidth={1.5} className="quick-switch-row-icon" />
                    <span className="quick-switch-row-name">{highlight(g)}</span>
                    <span className="quick-switch-row-meta">{SETTINGS_HINTS[g]}</span>
                  </Command.Item>
                ))}
            </Section>
          ))}
        </Command.List>

        <div className="quick-switch-footer">
          <span className="quick-switch-hint">
            <ArrowUp size={12} strokeWidth={1.5} />
            <ArrowDown size={12} strokeWidth={1.5} /> navigate
          </span>
          <span className="quick-switch-hint">
            <CornerDownLeft size={12} strokeWidth={1.5} /> select
          </span>
          <span className="quick-switch-hint">Type blame and a file name to blame it</span>
          {resultCount > 0 && <span className="quick-switch-result-count">{plural(resultCount, 'result')}</span>}
        </div>
      </Command>
    </Modal>
  )
}

export default QuickSwitchPalette
