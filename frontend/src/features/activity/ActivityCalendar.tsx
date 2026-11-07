import { ActivityService } from '@current-client-bindings/app'
import { ACTIVITY_INTENSITY_STEPS, activityIntensity, layoutActivityCalendar } from './calendarLayout'
import { peakDayCommits } from './activityStats'
import { plural } from '../../lib/plural'
import { useAsyncData } from '../../lib/useAsyncData'
import './ActivityCalendar.scss'

interface ActivityCalendarProps {
  repoPath: string
  repoVersion: number
}

const DAYS = 371 // 53 full weeks, GitHub-contribution-graph shape
const WEEKDAY_LABELS = [
  { day: 1, label: 'Mon' },
  { day: 3, label: 'Wed' },
  { day: 5, label: 'Fri' },
]

// A label above the first week of each month, skipping a month too short to fit one.
function monthLabels(cells: { date: string; weekIndex: number }[]) {
  const labels: { weekIndex: number; label: string }[] = []
  let lastMonth = -1
  for (const cell of cells) {
    const date = new Date(cell.date + 'T00:00:00')
    if (date.getMonth() === lastMonth) continue
    lastMonth = date.getMonth()
    labels.push({ weekIndex: cell.weekIndex, label: date.toLocaleDateString(undefined, { month: 'short' }) })
  }
  return labels.filter((l, i) => (labels[i + 1]?.weekIndex ?? Infinity) - l.weekIndex >= 3)
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
        <span className="activity-title">Commits by day</span>
        {activity && <span className="activity-badge">{plural(totalCommits, 'commit')} in the last year</span>}
        <div className="activity-header-meta">
          <div className="activity-legend">
            <span className="activity-legend-label">Less</span>
            {ACTIVITY_INTENSITY_STEPS.map((intensity) => (
              <span
                key={intensity}
                className="activity-legend-swatch"
                style={{ background: `color-mix(in srgb, var(--tertiary) ${intensity}%, var(--surface-high))` }}
              />
            ))}
            <span className="activity-legend-label">More</span>
          </div>
          {activity && <span className="activity-range">Peak day: {plural(peakDayCommits(activity), 'commit')}</span>}
        </div>
      </div>

      <div className="activity-body">
        {error ? (
          <p className="activity-hint">Could not load activity: {error}</p>
        ) : activity === null ? (
          <p className="activity-hint">Loading activity…</p>
        ) : (
          <div
            className="activity-grid"
            role="img"
            aria-label={`Commits by day: ${plural(
              cells.reduce((sum, cell) => sum + cell.commits, 0),
              'commit',
            )} in the last year`}
            style={{ gridTemplateColumns: `auto repeat(${weekCount}, minmax(9px, 18px))` }}
          >
            {monthLabels(cells).map(({ weekIndex, label }) => (
              <span
                key={`${weekIndex}-${label}`}
                className="activity-month"
                style={{ gridColumn: `${weekIndex + 2} / span 4` }}
              >
                {label}
              </span>
            ))}
            {WEEKDAY_LABELS.map(({ day, label }) => (
              <span key={label} className="activity-weekday" style={{ gridRow: day + 2 }}>
                {label}
              </span>
            ))}
            {cells.map((cell) => (
              <span
                key={cell.date}
                className="activity-cell"
                style={{
                  gridColumn: cell.weekIndex + 2,
                  gridRow: cell.dayOfWeek + 2,
                  background: `color-mix(in srgb, var(--tertiary) ${activityIntensity(cell.commits)}%, var(--surface-high))`,
                }}
                title={`${cell.date}: ${plural(cell.commits, 'commit')}`}
              />
            ))}
          </div>
        )}
      </div>
    </div>
  )
}

export default ActivityCalendar
