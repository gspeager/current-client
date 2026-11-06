import { useState } from 'react'
import {
  ContributorsService,
  IdentityService,
  type ContributorStatInfo,
  type IdentityInfo,
} from '@current-client-bindings/app'
import { plural } from '../../lib/plural'
import { sinceDaysAgo } from '../../lib/relativeTime'
import { useAsyncData } from '../../lib/useAsyncData'
import IdentityBadge from '../../components/git/IdentityBadge'
import './ContributorsChart.scss'

interface ContributorsChartProps {
  repoPath: string
  repoVersion: number
}

type Contributor = ContributorStatInfo & IdentityInfo

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

  return (
    <div className="contributors-chart">
      <div className="contributors-header">
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
            <ul className="contributors-list">
              {contributors.map((c) => (
                <li key={c.email} className="contributors-row">
                  <IdentityBadge identity={c} name={c.name} email={c.email} size={28} />
                  <div className="contributors-row-identity">
                    <span className="contributors-row-line">
                      <span className="contributors-row-name" title={c.email}>
                        {c.name}
                      </span>
                      <span className="contributors-row-commits">
                        {plural(c.commits, 'commit')} · {total === 0 ? 0 : Math.round((c.commits / total) * 100)}%
                      </span>
                    </span>
                    <span className="contributors-row-bar" aria-hidden="true">
                      <span style={{ width: `${total === 0 ? 0 : (c.commits / total) * 100}%`, background: c.color }} />
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
