import { FolderOpen } from 'lucide-react'
import { RepositoryService } from '@current-client-bindings/app'
import { useAsyncData } from '../../lib/useAsyncData'
import type { useRepositoryLifecycle } from './useRepositoryLifecycle'
import './RepoQuickList.scss'

interface RepoQuickListProps {
  repo: ReturnType<typeof useRepositoryLifecycle>
  // Opens the Open repository dialog, for anything the list doesn't hold.
  onOpenAnother: () => void
}

// The breadcrumb's repository pane: a switcher, not a second Open dialog.
function RepoQuickList({ repo, onOpenAnother }: RepoQuickListProps) {
  const { data: summaries } = useAsyncData(
    () => RepositoryService.GetRepoSummaries(repo.recentRepos).catch(() => []),
    [repo.recentRepos],
    { keepData: true },
  )
  const rows = summaries?.filter((s) => repo.recentRepos.includes(s.path)) ?? []

  return (
    <div className="repo-quick-list">
      <span className="repo-quick-list-label">Recent</span>
      {summaries === null ? (
        <p className="repo-quick-list-hint">Loading…</p>
      ) : rows.length === 0 ? (
        <p className="repo-quick-list-hint">No recent repositories.</p>
      ) : (
        <ul>
          {rows.map((s) => (
            <li key={s.path}>
              <button
                type="button"
                className={s.path === repo.repoPath ? 'repo-quick-row repo-quick-row-current' : 'repo-quick-row'}
                onClick={() => repo.openRecent(s.path)}
                disabled={!s.available}
                title={s.path}
              >
                <span className="repo-quick-row-name">{s.name}</span>
                <span className="repo-quick-row-meta">
                  {s.available ? s.currentBranch : 'Not found'}
                  {repo.tabs.includes(s.path) && <span className="repo-quick-row-open">open</span>}
                </span>
              </button>
            </li>
          ))}
        </ul>
      )}
      <button type="button" className="repo-quick-list-another" onClick={onOpenAnother}>
        <FolderOpen size={14} strokeWidth={1.75} />
        Open another…
      </button>
    </div>
  )
}

export default RepoQuickList
