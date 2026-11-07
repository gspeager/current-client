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
          <h3 className="commit-velocity-title">Commits per week</h3>
          <span className="commit-velocity-subtitle">
            Last {WEEK_COUNT} weeks{average !== null && ` · average ${average.toFixed(1)}`}
            {busiest && ` · most on ${busiest.day}`}
          </span>
        </div>
      </div>

      <div className="commit-velocity-body">
        {error ? (
          <p className="commit-velocity-hint">Could not load commit velocity: {error}</p>
        ) : activity === null ? (
          <p className="commit-velocity-hint">Loading…</p>
        ) : (
          <div className="commit-velocity-bars">
            {average !== null && average > 0 && (
              <span
                className="commit-velocity-average"
                title={`Average ${average.toFixed(1)} a week`}
                style={{ bottom: `calc(20px + (100% - 20px) * ${average / maxCommits})` }}
              />
            )}
            {weeks.map((week, i) => (
              <div
                key={week.weekStart}
                className="commit-velocity-bar-wrap"
                title={`Week of ${week.weekStart}: ${plural(week.commits, 'commit')}`}
              >
                <span className="commit-velocity-bar-area">
                  <span
                    className={
                      i === weeks.length - 1 ? 'commit-velocity-bar commit-velocity-bar-current' : 'commit-velocity-bar'
                    }
                    style={{ height: `${Math.max(4, (week.commits / maxCommits) * 100)}%` }}
                  />
                </span>
                <span
                  className={`commit-velocity-week-label${i === weeks.length - 1 ? ' commit-velocity-week-label-current' : ''}`}
                >
                  {i === weeks.length - 1
                    ? 'Now'
                    : (weeks.length - 1 - i) % 3 === 0
                      ? new Date(`${week.weekStart}T00:00:00`).toLocaleDateString(undefined, {
                          month: 'short',
                          day: 'numeric',
                        })
                      : ''}
                </span>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  )
}

export default CommitVelocityChart
