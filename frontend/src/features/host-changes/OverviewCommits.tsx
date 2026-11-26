import type { CSSProperties } from 'react'
import { useLaneColors } from '../../lib/laneColor'
import { HistoryService } from '@current-client-bindings/app'
import { plural } from '../../lib/plural'
import { relativeTime } from '../../lib/relativeTime'
import { useAsyncData } from '../../lib/useAsyncData'
import { shortSha } from '../../lib/text'
import './changeRows.scss'
import './OverviewCommits.scss'

// The change's commits from what is already fetched, so opening it asks the host
// for nothing more. Silent when the branch isn't here locally.
function OverviewCommits({
  repoPath,
  target,
  source,
  current,
}: {
  repoPath: string
  target: string
  source: string
  current: boolean
}) {
  const { data } = useAsyncData(
    () => HistoryService.RangeCommits(repoPath, `origin/${target}`, `origin/${source}`).catch(() => null),
    [repoPath, target, source],
  )
  const { branchColor } = useLaneColors()
  if (!data || data.length === 0) return null
  return (
    <section
      className="host-change-overview-commits"
      style={{ '--lane': branchColor(source, current) } as CSSProperties}
      aria-label="Commits"
    >
      <h4 className="host-change-overview-heading">{plural(data.length, 'commit')}</h4>
      <ul className="host-change-commits">
        {data.map((commit) => (
          <li key={commit.sha} className="host-change-commit">
            <span className="host-change-commit-subject">{commit.subject}</span>
            <span className="host-change-commit-sha">{shortSha(commit.sha)}</span>
            <span className="host-change-commit-meta">
              {commit.authorName} · {relativeTime(new Date(commit.date))}
            </span>
          </li>
        ))}
      </ul>
    </section>
  )
}

export default OverviewCommits
