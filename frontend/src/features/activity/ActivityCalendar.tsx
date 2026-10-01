import { ActivityService } from '@current-client-bindings/app'
import { ACTIVITY_INTENSITY_STEPS, activityIntensity, layoutActivityCalendar } from './calendarLayout'
import { currentStreak, longestStreak, peakDayCommits } from './activityStats'
import { useAsyncData } from '../../lib/useAsyncData'
import './ActivityCalendar.scss'

interface ActivityCalendarProps {
  repoPath: string
  repoVersion: number
}

const DAYS = 371 // 53 full weeks, GitHub-contribution-graph shape

function formatDays(days: number): string {
  return `${days} ${days === 1 ? 'day' : 'days'}`
}

function formatCommits(commits: number): string {
  return `${commits} ${commits === 1 ? 'commit' : 'commits'}`
}

function ActivityCalendar({ repoPath, repoVersion }: ActivityCalendarProps) {
  const { data: activity, error } = useAsyncData(() => ActivityService.GetCommitActivity(repoPath, DAYS), [repoPath], {
    refreshKey: repoVersion,
  })

  const cells = activity ? layoutActivityCalendar(activity) : []
  const weekCount = cells.length > 0 ? cells[cells.length - 1].weekIndex + 1 : 0
  const totalCommits = activity?.reduce((sum, d) => sum + d.commits, 0) ?? 0

  return (
    <div className="activity-calendar">
      <div className="activity-header">
        <span className="activity-title">Activity</span>
        {activity && (
          <span className="activity-badge">
            {totalCommits} commit{totalCommits === 1 ? '' : 's'} in the last year
          </span>
        )}
        <div className="activity-header-meta">
          <div className="activity-legend">
            <span className="activity-legend-label">Less</span>
            {ACTIVITY_INTENSITY_STEPS.map((intensity) => (
              <span
                key={intensity}
                className="activity-legend-swatch"
                style={{ background: `color-mix(in srgb, var(--tertiary) ${intensity}%, var(--surface-container))` }}
              />
            ))}
            <span className="activity-legend-label">More</span>
          </div>
          <span className="activity-range">Last 365 days</span>
        </div>
      </div>

      <div className="activity-body">
        {error ? (
          <p className="activity-hint">Could not load activity: {error}</p>
        ) : activity === null ? (
          <p className="activity-hint">Loading activity…</p>
        ) : (
          <div className="activity-grid" style={{ gridTemplateColumns: `repeat(${weekCount}, 1fr)` }}>
            {cells.map((cell) => (
              <span
                key={cell.date}
                className="activity-cell"
                style={{
                  gridColumn: cell.weekIndex + 1,
                  gridRow: cell.dayOfWeek + 1,
                  background: `color-mix(in srgb, var(--tertiary) ${activityIntensity(cell.commits)}%, var(--surface-container))`,
                }}
                title={`${cell.date}: ${cell.commits} commit${cell.commits === 1 ? '' : 's'}`}
              />
            ))}
          </div>
        )}
      </div>

      {activity && (
        <div className="activity-footer">
          <div className="activity-footer-stats">
            <span className="activity-footer-stat">
              <span className="activity-footer-dot activity-footer-dot-longest" />
              Longest streak: <strong>{formatDays(longestStreak(activity))}</strong>
            </span>
            <span className="activity-footer-stat">
              <span className="activity-footer-dot activity-footer-dot-current" />
              Current streak: <strong>{formatDays(currentStreak(activity))}</strong>
            </span>
            <span className="activity-footer-stat">
              <span className="activity-footer-dot activity-footer-dot-peak" />
              Peak day: <strong>{formatCommits(peakDayCommits(activity))}</strong>
            </span>
          </div>
          <span className="activity-footer-hint">Hover over any square for daily counts</span>
        </div>
      )}
    </div>
  )
}

export default ActivityCalendar
