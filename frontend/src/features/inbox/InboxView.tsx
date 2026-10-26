import { useMemo, useState } from 'react'
import {
  Check,
  ChevronDown,
  ChevronRight,
  CircleDot,
  CircleX,
  Clock,
  RefreshCw,
  Eye,
  FileDiff,
  GitMerge,
  MessageSquare,
  TriangleAlert,
  Upload,
  type LucideIcon,
} from 'lucide-react'
import { HostService, InboxService, type InboxRepo, type InboxRow } from '@current-client-bindings/app'
import { errorMessage } from '../../lib/errors'
import { plural } from '../../lib/plural'
import { relativeTime } from '../../lib/relativeTime'
import { useAsyncData } from '../../lib/useAsyncData'
import { useClockTick } from '../../lib/useClockTick'
import './InboxView.scss'

export type InboxTarget =
  | { path: string; kind: 'changes' }
  | { path: string; kind: 'branch'; branch: string }
  | { path: string; kind: 'change'; number: number }
  | { path: string; kind: 'issue'; key: string }

interface InboxViewProps {
  paths: string[]
  // The repository whose group starts open; the rest start folded.
  activePath?: string | null
  // Rows are read again when this changes; opening the view fetches nothing.
  refreshKey: unknown
  onGo: (target: InboxTarget) => void
  // Called after a group's own Refresh, so the rest of the app re-reads too.
  onRefreshed?: () => void
}

const ICONS: Record<string, LucideIcon> = {
  dirty: FileDiff,
  operation: GitMerge,
  unpushed: Upload,
  stale: Clock,
  merged: Check,
  conflict: TriangleAlert,
  review: Eye,
  'changes-requested': MessageSquare,
  'ci-failed': CircleX,
  assigned: CircleDot,
}

export function targetOf(path: string, row: InboxRow): InboxTarget {
  switch (row.kind) {
    case 'dirty':
    case 'operation':
      return { path, kind: 'changes' }
    case 'review':
    case 'changes-requested':
    case 'ci-failed':
      return { path, kind: 'change', number: row.number }
    case 'assigned':
      return { path, kind: 'issue', key: row.key }
    default:
      return { path, kind: 'branch', branch: row.branch }
  }
}

// What the row says, and why it is here.
function describe(row: InboxRow): { subject: string; reason: string } {
  switch (row.kind) {
    case 'dirty':
      return { subject: plural(row.count, 'changed file'), reason: 'uncommitted' }
    case 'operation':
      return {
        subject: `${row.detail} in progress`,
        reason: row.count > 0 ? plural(row.count, 'conflicted file') : 'ready to continue',
      }
    case 'unpushed':
      return { subject: row.branch, reason: plural(row.count, 'unpushed commit') }
    case 'stale':
      return { subject: row.branch, reason: `no commits in ${plural(row.count, 'day')}` }
    case 'merged':
      return { subject: row.branch, reason: `merged into ${row.detail}` }
    case 'conflict':
      return { subject: row.branch, reason: `would conflict in ${row.detail}` }
    case 'review':
      return { subject: `${row.prefix}${row.number} ${row.title}`, reason: 'review requested' }
    case 'changes-requested':
      return { subject: `${row.prefix}${row.number} ${row.title}`, reason: `changes requested by ${row.detail}` }
    case 'ci-failed':
      return { subject: `${row.prefix}${row.number} ${row.title}`, reason: 'CI failed' }
    case 'assigned':
      return { subject: `#${row.key} ${row.title}`, reason: row.detail ? `assigned, ${row.detail}` : 'assigned to you' }
    default:
      return { subject: row.title || row.branch, reason: row.kind }
  }
}

// Everything waiting on you across the open repositories: local facts from
// Git, host facts from the last refresh's cache, each row a jump to it.
function InboxView({ paths, activePath = null, refreshKey, onGo, onRefreshed }: InboxViewProps) {
  useClockTick()
  const key = paths.join('\n')
  // Showing the rows counts as seeing them: the header's badge clears.
  const { data: repos, error } = useAsyncData(() => (key ? InboxService.Rows(key.split('\n'), true) : null), [key], {
    refreshKey,
    keepData: true,
  })
  const total = useMemo(() => (repos ?? []).reduce((n, r) => n + r.rows.length, 0), [repos])
  // Only the open repository starts expanded, so many repositories with
  // work waiting stay a short list of headers; a header click overrides.
  const [folded, setFolded] = useState<Record<string, boolean>>({})
  const isOpen = (path: string) => folded[path] ?? path === activePath

  return (
    <section className="inbox" aria-label="Inbox">
      <header className="inbox-header">
        <h2 className="inbox-title">Inbox</h2>
        <span className="inbox-summary">
          {repos ? (total === 0 ? 'Nothing waiting' : plural(total, 'item')) : 'Looking…'}
        </span>
      </header>
      {error && <p className="inbox-error">Could not read the repositories: {error}</p>}
      {paths.length === 0 && <p className="inbox-hint">Open a repository to see what waits in it.</p>}
      {repos?.map((repo) => (
        <InboxRepoSection
          key={repo.path}
          repo={repo}
          open={isOpen(repo.path)}
          onToggle={() => setFolded((f) => ({ ...f, [repo.path]: !isOpen(repo.path) }))}
          onGo={onGo}
          onRefreshed={onRefreshed}
        />
      ))}
    </section>
  )
}

interface InboxRepoSectionProps {
  repo: InboxRepo
  open: boolean
  onToggle: () => void
  onGo: (target: InboxTarget) => void
  onRefreshed?: () => void
}

function InboxRepoSection({ repo, open, onToggle, onGo, onRefreshed }: InboxRepoSectionProps) {
  const fresh = repo.rows.filter((row) => row.new).length
  const [refreshing, setRefreshing] = useState(false)
  const [refreshError, setRefreshError] = useState<string | null>(null)
  // A host that is on but hasn't been refreshed this session: its rows
  // aren't empty, they aren't known yet, and this is the one place the
  // Inbox offers a request, on a click.
  const unrefreshed = repo.hostEnabled && !repo.fetchedAt
  const refresh = () => {
    setRefreshing(true)
    setRefreshError(null)
    HostService.Refresh(repo.path)
      .then(() => onRefreshed?.())
      .catch((err: unknown) => setRefreshError(errorMessage(err)))
      .finally(() => setRefreshing(false))
  }
  return (
    <section className="inbox-repo" aria-label={repo.name}>
      <h3 className="inbox-repo-name">
        <button type="button" className="inbox-repo-toggle" aria-expanded={open} onClick={onToggle}>
          {open ? <ChevronDown size={14} strokeWidth={1.5} /> : <ChevronRight size={14} strokeWidth={1.5} />}
          {repo.name}
          <span className="inbox-repo-count">{repo.rows.length}</span>
          {fresh > 0 && <span className="inbox-repo-new">{fresh} new</span>}
        </button>
      </h3>
      {repo.error && <p className="inbox-error">{repo.error}</p>}
      {open && unrefreshed && (
        <p className="inbox-hint inbox-unrefreshed">
          {refreshing ? 'Refreshing from the host…' : 'The host hasn’t been refreshed yet, so its rows aren’t known.'}
          {!refreshing && (
            <button type="button" className="inbox-refresh" onClick={refresh}>
              <RefreshCw size={12} strokeWidth={1.75} />
              Refresh
            </button>
          )}
        </p>
      )}
      {refreshError && <p className="inbox-error">Could not refresh: {refreshError}</p>}
      {open && !repo.error && !unrefreshed && repo.rows.length === 0 && <p className="inbox-hint">Nothing waiting.</p>}
      {open && repo.rows.length > 0 && (
        <ul className="inbox-rows">
          {repo.rows.map((row, i) => {
            const Icon = ICONS[row.kind] ?? CircleDot
            const { subject, reason } = describe(row)
            return (
              <li key={`${row.kind}-${row.branch}-${row.number}-${row.key}-${i}`}>
                <button
                  type="button"
                  className={`inbox-row inbox-row-${row.kind}`}
                  onClick={() => onGo(targetOf(repo.path, row))}
                >
                  <Icon size={14} strokeWidth={1.75} className="inbox-row-icon" />
                  <span className="inbox-row-text">
                    <span className="inbox-row-subject">{subject}</span>
                    <span className="inbox-row-reason">{reason}</span>
                    {row.new && <span className="inbox-row-new">new</span>}
                  </span>
                  {row.fetchedAt && (
                    <span className="inbox-row-age">from the host {relativeTime(new Date(row.fetchedAt))}</span>
                  )}
                </button>
              </li>
            )
          })}
        </ul>
      )}
    </section>
  )
}

export default InboxView
