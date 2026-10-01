import { useState } from 'react'
import {
  ContributorsService,
  IdentityService,
  type ContributorStatInfo,
  type IdentityInfo,
} from '@current-client-bindings/app'
import { sinceDaysAgo } from '../../lib/relativeTime'
import { useAsyncData } from '../../lib/useAsyncData'
import IdentityBadge from '../../components/git/IdentityBadge'
import './ContributorsChart.scss'

interface ContributorsChartProps {
  repoPath: string
  repoVersion: number
}

type Contributor = ContributorStatInfo & IdentityInfo

const SIZE = 168
const STROKE = 22
const RADIUS = (SIZE - STROKE) / 2
const CIRCUMFERENCE = 2 * Math.PI * RADIUS

async function loadContributors(repoPath: string, since: string): Promise<Contributor[]> {
  const stats = await ContributorsService.GetContributors(repoPath, since, '')
  const list = [...stats].sort((a, b) => b.commits - a.commits)
  const identities = await IdentityService.GetIdentities(list)
  return list.map((s, i) => ({ ...s, ...identities[i] }))
}

function ContributorsChart({ repoPath, repoVersion }: ContributorsChartProps) {
  const [sinceDays, setSinceDays] = useState(0)
  const { data: contributors, error } = useAsyncData(
    () => loadContributors(repoPath, sinceDaysAgo(sinceDays)),
    [repoPath, sinceDays],
    { refreshKey: repoVersion },
  )

  const total = contributors?.reduce((sum, c) => sum + c.commits, 0) ?? 0
  const topContributor = contributors?.[0]
  const topSharePercent = topContributor && total > 0 ? Math.round((topContributor.commits / total) * 100) : 0
  let cumulative = 0

  return (
    <div className="contributors-chart">
      <div className="contributors-header">
        <span className="contributors-header-dot" style={{ background: topContributor?.color ?? 'var(--outline)' }} />
        <span className="contributors-title">Contributors</span>
        <select
          className="contributors-date-filter"
          value={sinceDays}
          onChange={(e) => setSinceDays(Number(e.target.value))}
          aria-label="Date range"
        >
          <option value={0}>All time</option>
          <option value={30}>Past 30 days</option>
          <option value={90}>Past 90 days</option>
          <option value={365}>Past year</option>
        </select>
      </div>

      <div className="contributors-body">
        {error ? (
          <p className="contributors-hint">Could not load contributors: {error}</p>
        ) : contributors === null ? (
          <p className="contributors-hint">Loading contributors…</p>
        ) : contributors.length === 0 ? (
          <p className="contributors-hint">No commits in this range.</p>
        ) : (
          <>
            <div className="contributors-donut-wrap">
              <svg width={SIZE} height={SIZE} viewBox={`0 0 ${SIZE} ${SIZE}`} className="contributors-donut">
                <g transform={`rotate(-90 ${SIZE / 2} ${SIZE / 2})`}>
                  {contributors.map((c) => {
                    const fraction = total === 0 ? 0 : c.commits / total
                    const dash = fraction * CIRCUMFERENCE
                    const offset = cumulative
                    cumulative += dash
                    return (
                      <circle
                        key={c.email}
                        cx={SIZE / 2}
                        cy={SIZE / 2}
                        r={RADIUS}
                        fill="none"
                        stroke={c.color}
                        strokeWidth={STROKE}
                        strokeDasharray={`${dash} ${CIRCUMFERENCE - dash}`}
                        strokeDashoffset={-offset}
                      />
                    )
                  })}
                </g>
              </svg>
              <div className="contributors-donut-center">
                <span className="contributors-donut-value">{topSharePercent}%</span>
                <span className="contributors-donut-caption" style={{ color: topContributor?.color }}>
                  {contributors.length} AUTHOR{contributors.length === 1 ? '' : 'S'}
                </span>
              </div>
            </div>

            <ul className="contributors-list">
              {contributors.map((c) => (
                <li key={c.email} className="contributors-row">
                  <IdentityBadge identity={c} name={c.name} email={c.email} size={32} />
                  <div className="contributors-row-identity">
                    <span className="contributors-row-name">{c.name}</span>
                    <span className="contributors-row-email">{c.email}</span>
                  </div>
                  <div className="contributors-row-stats">
                    <span className="contributors-row-commits">
                      {c.commits} commit{c.commits === 1 ? '' : 's'}
                    </span>
                    <span className="contributors-row-share" style={{ color: c.color }}>
                      {total === 0 ? 0 : Math.round((c.commits / total) * 100)}% share
                    </span>
                  </div>
                </li>
              ))}
            </ul>
          </>
        )}
      </div>
    </div>
  )
}

export default ContributorsChart
