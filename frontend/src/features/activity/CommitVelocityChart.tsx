import { ActivityService } from '@current-client-bindings/app'
import { averageWeeklyCommits, busiestWeekday, weeklyBuckets } from './commitVelocity'
import { plural } from '../../lib/plural'
import { useAsyncData } from '../../lib/useAsyncData'
import './CommitVelocityChart.scss'

interface CommitVelocityChartProps {
  repoPath: string
  repoVersion: number
}

const DAYS = 84
const WEEK_COUNT = DAYS / 7

function CommitVelocityChart({ repoPath, repoVersion }: CommitVelocityChartProps) {
  const { data: activity, error } = useAsyncData(() => ActivityService.GetCommitActivity(repoPath, DAYS), [repoPath], {
    refreshKey: repoVersion,
  })

  const weeks = activity ? weeklyBuckets(activity) : []
  const busiest = activity ? busiestWeekday(activity) : null
  const maxCommits = Math.max(1, ...weeks.map((w) => w.commits))
  const average = weeks.length > 0 ? averageWeeklyCommits(weeks) : null

  return (
    <div className="commit-velocity">
      <div className="commit-velocity-header">
        <div className="commit-velocity-heading">
          <span className="commit-velocity-title">Commit velocity</span>
          <span className="commit-velocity-subtitle">Cadence frequency by {WEEK_COUNT}-week trend</span>
        </div>
        {busiest && <span className="commit-velocity-badge">Most active: {busiest.day.slice(0, 3)}</span>}
      </div>

      <div className="commit-velocity-body">
        {error ? (
          <p className="commit-velocity-hint">Could not load commit velocity: {error}</p>
        ) : activity === null ? (
          <p className="commit-velocity-hint">Loading…</p>
        ) : (
          <div className="commit-velocity-bars">
            {weeks.map((week, i) => (
              <div
                key={week.weekStart}
                className="commit-velocity-bar-wrap"
                title={`Week of ${week.weekStart}: ${plural(week.commits, 'commit')}`}
              >
                <span
                  className="commit-velocity-bar"
                  style={{ height: `${Math.max(4, (week.commits / maxCommits) * 100)}%` }}
                />
                <span
                  className={`commit-velocity-week-label${i === weeks.length - 1 ? ' commit-velocity-week-label-current' : ''}`}
                >
                  W{i + 1}
                </span>
              </div>
            ))}
          </div>
        )}
      </div>

      {average !== null && (
        <div className="commit-velocity-footer">
          <span className="commit-velocity-average">Average: {average.toFixed(1)} commits/wk</span>
        </div>
      )}
    </div>
  )
}

export default CommitVelocityChart
