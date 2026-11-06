import type { ReactNode } from 'react'
import {
  ActivityService,
  DashboardService,
  type RepoDiskUsageInfo,
  type RepoStatsInfo,
} from '@current-client-bindings/app'
import { weeklyBuckets } from './commitVelocity'
import { formatAge } from './repoAge'
import { useAsyncData } from '../../lib/useAsyncData'
import './RepoStatStrip.scss'

interface RepoStatStripProps {
  repoPath: string
  repoVersion: number
  dirty: boolean
  action?: ReactNode
}

interface Stat {
  label: string
  value: string
}

function formatDiskSize(kb: number): string {
  if (kb < 1024) return `${kb} KB`
  const mb = kb / 1024
  if (mb < 1024) return `${mb.toFixed(1)} MB`
  return `${(mb / 1024).toFixed(2)} GB`
}

function statsFor(s: RepoStatsInfo, disk: RepoDiskUsageInfo): Stat[] {
  return [
    { label: 'Commits', value: String(s.totalCommits) },
    { label: 'Contributors', value: String(s.contributorCount) },
    { label: 'Tracked files', value: String(s.fileCount) },
    s.totalCommits === 0
      ? { label: 'Repository age', value: '—' }
      : { label: 'Repository age', value: formatAge(s.repoAgeDays) },
    { label: 'Repository size', value: formatDiskSize(disk.totalSizeKb) },
  ]
}

// The last twelve weeks beside the all-time count, so the total has a direction.
function CommitSparkline({ repoPath, repoVersion }: { repoPath: string; repoVersion: number }) {
  const { data } = useAsyncData(() => ActivityService.GetCommitActivity(repoPath, 84), [repoPath], {
    refreshKey: repoVersion,
  })
  if (!data) return null
  const weeks = weeklyBuckets(data)
  const max = Math.max(1, ...weeks.map((w) => w.commits))
  return (
    <svg
      className="repo-stat-strip-spark"
      width={weeks.length * 5}
      height="20"
      role="img"
      aria-label="Commits per week, last 12 weeks"
    >
      {weeks.map((week, i) => {
        const height = Math.max(2, (week.commits / max) * 20)
        return <rect key={week.weekStart} x={i * 5} y={20 - height} width="4" height={height} rx="1" />
      })}
    </svg>
  )
}

function RepoStatStrip({ repoPath, repoVersion, dirty, action }: RepoStatStripProps) {
  const { data, error } = useAsyncData(
    () => Promise.all([DashboardService.GetRepoStats(repoPath), DashboardService.GetRepoDiskUsage(repoPath)]),
    [repoPath],
    { refreshKey: repoVersion },
  )

  return (
    <div className="repo-stat-strip">
      {error ? (
        <p className="repo-stat-strip-hint">Could not load repository stats: {error}</p>
      ) : data === null ? (
        <p className="repo-stat-strip-hint">Loading…</p>
      ) : (
        <>
          <div className="repo-stat-strip-group">
            <div className="repo-stat-strip-item">
              <span className="repo-stat-strip-label">Branch</span>
              <span className="repo-stat-strip-value repo-stat-strip-branch" title={data[0].currentBranch}>
                <span className="repo-stat-strip-branch-name">{data[0].currentBranch}</span>
                <span
                  className={`repo-stat-strip-dot${dirty ? ' repo-stat-strip-dot-dirty' : ' repo-stat-strip-dot-clean'}`}
                />
              </span>
            </div>
          </div>
          {statsFor(...data).map((stat) => (
            <div key={stat.label} className="repo-stat-strip-group">
              <div className="repo-stat-strip-item">
                <span className="repo-stat-strip-label">{stat.label}</span>
                <span className="repo-stat-strip-value">
                  {stat.value}
                  {stat.label === 'Commits' && <CommitSparkline repoPath={repoPath} repoVersion={repoVersion} />}
                </span>
              </div>
            </div>
          ))}
        </>
      )}
      {action}
    </div>
  )
}

export default RepoStatStrip
