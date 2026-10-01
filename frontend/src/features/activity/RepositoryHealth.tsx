import { Check, X } from 'lucide-react'
import { BranchService, DashboardService } from '@current-client-bindings/app'
import { countHealthIssues, isStaleBranch, STALE_THRESHOLD_DAYS } from './repoHealth'
import { useAsyncData } from '../../lib/useAsyncData'
import { useBranchStatus } from '../../lib/useBranchStatus'
import './RepositoryHealth.scss'

interface RepositoryHealthProps {
  repoPath: string
  repoVersion: number
  dirty: boolean
  hasConflict: boolean
}

// Well below git's own gc.auto (6700) so the suggestion surfaces sooner.
const GC_SUGGEST_THRESHOLD = 1000

function RepositoryHealth({ repoPath, repoVersion, dirty, hasConflict }: RepositoryHealthProps) {
  const branchStatus = useBranchStatus(repoPath, repoVersion)
  const { data, error } = useAsyncData(
    () => Promise.all([BranchService.ListLocal(repoPath), DashboardService.GetRepoDiskUsage(repoPath)]),
    [repoPath],
    { refreshKey: repoVersion },
  )
  const staleBranchCount = data ? data[0].filter((b) => isStaleBranch(b.lastCommitDate)).length : null
  const looseObjectCount = data?.[1].looseObjectCount ?? null

  const workingTreeClean = !dirty && !hasConflict
  const unpushedCount = branchStatus?.ahead ?? null

  const loaded = staleBranchCount !== null && unpushedCount !== null
  const issueCount = loaded
    ? countHealthIssues({ staleBranchCount: staleBranchCount!, unpushedCount: unpushedCount!, workingTreeClean })
    : 0

  return (
    <div className="repo-health">
      <div className="repo-health-header">
        <span className="repo-health-title">Repository health</span>
        {loaded && (
          <span className={`repo-health-badge repo-health-badge-${issueCount === 0 ? 'optimal' : 'attention'}`}>
            {issueCount === 0 ? 'Optimal' : `${issueCount} issue${issueCount === 1 ? '' : 's'}`}
          </span>
        )}
      </div>

      <div className="repo-health-body">
        {error ? (
          <p className="repo-health-hint">Could not load repository health: {error}</p>
        ) : !loaded ? (
          <p className="repo-health-hint">Loading…</p>
        ) : (
          <>
            <div className="repo-health-row">
              <div className="repo-health-row-text">
                <span className="repo-health-row-title">Stale branches</span>
                <span className="repo-health-row-subtitle">
                  {staleBranchCount === 0
                    ? `No inactive branches >${STALE_THRESHOLD_DAYS}d`
                    : `${staleBranchCount} branch${staleBranchCount === 1 ? '' : 'es'} inactive >${STALE_THRESHOLD_DAYS}d`}
                </span>
              </div>
              <span className={`repo-health-value repo-health-value-${staleBranchCount === 0 ? 'ok' : 'warn'}`}>
                {staleBranchCount}
              </span>
            </div>

            <div className="repo-health-row">
              <div className="repo-health-row-text">
                <span className="repo-health-row-title">Unpushed commits</span>
                <span className="repo-health-row-subtitle">
                  {!branchStatus?.upstream
                    ? 'No upstream configured'
                    : unpushedCount === 0
                      ? `Synced with ${branchStatus.upstream}`
                      : `${unpushedCount} ahead of ${branchStatus.upstream}`}
                </span>
              </div>
              <span className={`repo-health-value repo-health-value-${!unpushedCount ? 'ok' : 'warn'}`}>
                {unpushedCount ?? 0}
              </span>
            </div>

            <div className="repo-health-row">
              <div className="repo-health-row-text">
                <span className="repo-health-row-title">Working directory</span>
                <span className="repo-health-row-subtitle">
                  {hasConflict
                    ? 'Merge in progress — resolve conflicts'
                    : dirty
                      ? 'Uncommitted changes present'
                      : 'Clean tree, no merge conflicts'}
                </span>
              </div>
              {workingTreeClean ? (
                <Check className="repo-health-icon repo-health-icon-ok" size={16} strokeWidth={2} />
              ) : (
                <X className="repo-health-icon repo-health-icon-warn" size={16} strokeWidth={2} />
              )}
            </div>

            {looseObjectCount !== null && looseObjectCount >= GC_SUGGEST_THRESHOLD && (
              <div className="repo-health-gc">
                <span>Loose objects: {looseObjectCount.toLocaleString()} —</span>
                <code>git gc --prune</code>
              </div>
            )}
          </>
        )}
      </div>
    </div>
  )
}

export default RepositoryHealth
