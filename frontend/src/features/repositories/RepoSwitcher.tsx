import { X } from 'lucide-react'
import { RepositoryService } from '@current-client-bindings/app'
import PathText from '../../components/git/PathText'
import { joinPath } from '../../lib/paths'
import { useAsyncData } from '../../lib/useAsyncData'
import type { useRepositoryLifecycle } from './useRepositoryLifecycle'
import './RepoSwitcher.scss'

interface RepoSwitcherProps {
  repo: ReturnType<typeof useRepositoryLifecycle>
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

function RepoSwitcher({ repo }: RepoSwitcherProps) {
  // keepData leaves the current cards up while the list refreshes, and the
  // filter drops a removed card at once, so the grid never empties and redraws.
  const { data: loadedSummaries } = useAsyncData(
    () => RepositoryService.GetRepoSummaries(repo.recentRepos).catch(() => []),
    [repo.recentRepos],
    { keepData: true },
  )
  const summaries = loadedSummaries?.filter((s) => repo.recentRepos.includes(s.path)) ?? null

  return (
    <div className="repo-switcher">
      {repo.repoError && <p className="repo-switcher-error">Could not open repository: {repo.repoError}</p>}

      <div className="repo-switcher-actions">
        <button type="button" className="repo-switcher-primary" onClick={repo.openRepository}>
          Open repository
        </button>
        <button type="button" className="repo-switcher-secondary" onClick={repo.initRepository}>
          Initialize repository
        </button>
      </div>

      <div className="repo-switcher-clone">
        <input
          aria-label="repository url"
          placeholder="Repository URL"
          value={repo.cloneUrl}
          onChange={(e) => repo.setCloneUrl(e.target.value)}
        />
        <div className="repo-switcher-clone-actions">
          <button type="button" className="repo-switcher-secondary" onClick={repo.chooseCloneDestination}>
            {repo.cloneDest || 'Choose destination'}
          </button>
          <input
            aria-label="folder name"
            placeholder="Folder name"
            value={repo.cloneFolder}
            onChange={(e) => repo.setCloneFolder(e.target.value)}
          />
        </div>
        {repo.cloneDest && repo.cloneFolder.trim() && (
          <p className="repo-switcher-clone-target">
            <span>Clones into</span>
            <PathText path={joinPath(repo.cloneDest, repo.cloneFolder.trim())} className="repo-switcher-clone-path" />
          </p>
        )}
        <div className="repo-switcher-clone-actions">
          <button
            type="button"
            className="repo-switcher-secondary"
            onClick={repo.cloneRepository}
            disabled={!repo.cloneUrl || !repo.cloneDest || !repo.cloneFolder.trim() || repo.cloneOp.running}
          >
            {repo.cloneOp.running ? 'Cloning…' : 'Clone repository'}
          </button>
          {repo.cloneOp.running && (
            <button type="button" className="repo-switcher-secondary" onClick={repo.cloneOp.cancel}>
              Cancel
            </button>
          )}
        </div>
        {repo.cloneOp.error && <p className="repo-switcher-error">Could not clone: {repo.cloneOp.error}</p>}
      </div>

      {repo.recentRepos.length > 0 && (
        <div className="repo-switcher-recent">
          <span className="repo-switcher-label">Recent</span>
          {summaries === null ? (
            <p className="repo-switcher-loading">Loading…</p>
          ) : (
            <div className="repo-card-grid">
              {summaries.map((summary) => (
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
              ))}
            </div>
          )}
        </div>
      )}
    </div>
  )
}

export default RepoSwitcher
