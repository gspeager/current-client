import { GitBranch } from 'lucide-react'
import { DashboardService, type RepoDiskUsageInfo, type RepoStatsInfo } from '@current-client-bindings/app'
import { ageColor, formatAge } from './repoAge'
import { useAsyncData } from '../../lib/useAsyncData'
import './RepoStatStrip.scss'

interface RepoStatStripProps {
  repoPath: string
  repoVersion: number
  dirty: boolean
}

interface Stat {
  label: string
  value: string
  color?: string
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
      : { label: 'Repository age', value: formatAge(s.repoAgeDays), color: ageColor(s.repoAgeDays) },
    { label: 'Branches', value: String(s.branchCount) },
    { label: 'Tags / Stashes', value: `${s.tagCount} / ${s.stashCount}` },
    { label: 'Repo size', value: formatDiskSize(disk.totalSizeKb) },
  ]
}

function RepoStatStrip({ repoPath, repoVersion, dirty }: RepoStatStripProps) {
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
            <div className="repo-stat-strip-icon">
              <GitBranch size={16} strokeWidth={1.75} />
            </div>
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
              <span className="repo-stat-strip-divider" />
              <div className="repo-stat-strip-item">
                <span className="repo-stat-strip-label">{stat.label}</span>
                <span className="repo-stat-strip-value" style={stat.color ? { color: stat.color } : undefined}>
                  {stat.value}
                </span>
              </div>
            </div>
          ))}
        </>
      )}
    </div>
  )
}

export default RepoStatStrip
