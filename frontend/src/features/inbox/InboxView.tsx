import { useMemo } from 'react'
import {
  Check,
  CircleDot,
  CircleX,
  Clock,
  Eye,
  FileDiff,
  GitMerge,
  MessageSquare,
  TriangleAlert,
  Upload,
  type LucideIcon,
} from 'lucide-react'
import { InboxService, type InboxRepo, type InboxRow } from '@current-client-bindings/app'
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
  // Rows are read again when this changes; opening the view fetches nothing.
  refreshKey: unknown
  onGo: (target: InboxTarget) => void
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
function InboxView({ paths, refreshKey, onGo }: InboxViewProps) {
  useClockTick()
  const key = paths.join('\n')
  // Showing the rows counts as seeing them: the header's badge clears.
  const { data: repos, error } = useAsyncData(() => (key ? InboxService.Rows(key.split('\n'), true) : null), [key], {
    refreshKey,
    keepData: true,
  })
  const total = useMemo(() => (repos ?? []).reduce((n, r) => n + r.rows.length, 0), [repos])

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
        <InboxRepoSection key={repo.path} repo={repo} onGo={onGo} />
      ))}
    </section>
  )
}

function InboxRepoSection({ repo, onGo }: { repo: InboxRepo; onGo: (target: InboxTarget) => void }) {
  return (
    <section className="inbox-repo" aria-label={repo.name}>
      <h3 className="inbox-repo-name">
        {repo.name}
        <span className="inbox-repo-count">{repo.rows.length}</span>
      </h3>
      {repo.error && <p className="inbox-error">{repo.error}</p>}
      {!repo.error && repo.rows.length === 0 && <p className="inbox-hint">Nothing waiting.</p>}
      {repo.rows.length > 0 && (
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
