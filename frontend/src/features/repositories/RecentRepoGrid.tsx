import { X } from 'lucide-react'
import { RepositoryService } from '@current-client-bindings/app'
import PathText from '../../components/git/PathText'
import { useAsyncData } from '../../lib/useAsyncData'
import type { useRepositoryLifecycle } from './useRepositoryLifecycle'
import './RecentRepoGrid.scss'

interface RecentRepoGridProps {
  repo: ReturnType<typeof useRepositoryLifecycle>
  // Narrows the cards by name or path.
  query?: string
}

function Sparkline({ activity }: { activity: number[] }) {
  const max = Math.max(1, ...activity)
  return (
    <div className="repo-card-sparkline" aria-hidden="true">
      {activity.map((count, i) => (
        <span key={i} className="repo-card-sparkline-bar" style={{ height: `${(count / max) * 100}%` }} />
      ))}
    </div>
  )
}

// The recent repositories as cards: name, state, path, branch, with the
// ones already open badged.
function RecentRepoGrid({ repo, query = '' }: RecentRepoGridProps) {
  // keepData leaves the current cards up while the list refreshes, and the
  // filter drops a removed card at once, so the grid never empties and redraws.
  const { data: loadedSummaries } = useAsyncData(
    () => RepositoryService.GetRepoSummaries(repo.recentRepos).catch(() => []),
    [repo.recentRepos],
    { keepData: true },
  )
  const q = query.trim().toLowerCase()
  const summaries =
    loadedSummaries
      ?.filter((s) => repo.recentRepos.includes(s.path))
      .filter((s) => !q || s.name.toLowerCase().includes(q) || s.path.toLowerCase().includes(q)) ?? null

  if (repo.recentRepos.length === 0) return <p className="repo-grid-hint">No recent repositories.</p>
  if (summaries === null) return <p className="repo-grid-hint">Loading…</p>
  if (summaries.length === 0) return <p className="repo-grid-hint">No repository matches.</p>

  return (
    <div className="repo-card-grid">
      {summaries.map((summary) => {
        const open = repo.tabs.includes(summary.path)
        return (
          // The remove button sits beside the card, since a button can't contain one.
          <div key={summary.path} className="repo-card-wrap">
            <button
              type="button"
              className="repo-card"
              onClick={() => repo.openRecent(summary.path)}
              disabled={!summary.available}
              title={summary.path}
            >
              <div className="repo-card-header">
                <span
                  className={
                    summary.dirty > 0 ? 'repo-card-dot repo-card-dot-dirty' : 'repo-card-dot repo-card-dot-clean'
                  }
                />
                <span className="repo-card-name">{summary.name}</span>
                {open && <span className="repo-card-open">open</span>}
              </div>
              <PathText path={summary.path} className="repo-card-path" />
              {summary.available ? (
                <>
                  <div className="repo-card-meta">
                    <span className="repo-card-branch">{summary.currentBranch}</span>
                    {summary.ahead > 0 && <span className="repo-card-ahead">↑{summary.ahead}</span>}
                    {summary.behind > 0 && <span className="repo-card-behind">↓{summary.behind}</span>}
                  </div>
                  <Sparkline activity={summary.activity} />
                </>
              ) : (
                <span className="repo-card-unavailable">Not found</span>
              )}
            </button>
            <button
              type="button"
              className="repo-card-remove"
              onClick={() => repo.removeRecent(summary.path)}
              aria-label={`Remove ${summary.name} from recent`}
              title="Remove from recent"
            >
              <X size={14} strokeWidth={1.75} />
            </button>
          </div>
        )
      })}
    </div>
  )
}

export default RecentRepoGrid
