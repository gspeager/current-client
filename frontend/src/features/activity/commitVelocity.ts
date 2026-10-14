import type { DayActivityInfo } from '@current-client-bindings/app'

export interface WeekBucket {
  weekStart: string
  commits: number
}

// Rolling 7-day buckets from the oldest day, not calendar weeks.
export function weeklyBuckets(activity: DayActivityInfo[]): WeekBucket[] {
  const buckets: WeekBucket[] = []
  for (let i = 0; i < activity.length; i += 7) {
    const week = activity.slice(i, i + 7)
    buckets.push({
      weekStart: week[0].date,
      commits: week.reduce((sum, d) => sum + d.commits, 0),
    })
  }
  return buckets
}

const WEEKDAY_NAMES = ['Sunday', 'Monday', 'Tuesday', 'Wednesday', 'Thursday', 'Friday', 'Saturday']

interface BusiestWeekday {
  day: string
  commits: number
}

export function busiestWeekday(activity: DayActivityInfo[]): BusiestWeekday | null {
  const totals = new Array(7).fill(0)
  for (const day of activity) {
    totals[new Date(day.date + 'T00:00:00').getDay()] += day.commits
  }
  const max = Math.max(...totals)
  if (max === 0) return null
  const dayIndex = totals.indexOf(max)
  return { day: WEEKDAY_NAMES[dayIndex], commits: max }
}

export function averageWeeklyCommits(weeks: WeekBucket[]): number {
  if (weeks.length === 0) return 0
  return weeks.reduce((sum, w) => sum + w.commits, 0) / weeks.length
}
