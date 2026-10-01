import type { DayActivityInfo } from '@current-client-bindings/app'

// Anywhere in the range, not necessarily ending today.
export function longestStreak(activity: DayActivityInfo[]): number {
  let longest = 0
  let current = 0
  for (const day of activity) {
    if (day.commits > 0) {
      current += 1
      longest = Math.max(longest, current)
    } else {
      current = 0
    }
  }
  return longest
}

// Zero if the last day has no commits.
export function currentStreak(activity: DayActivityInfo[]): number {
  let streak = 0
  for (let i = activity.length - 1; i >= 0; i--) {
    if (activity[i].commits === 0) break
    streak += 1
  }
  return streak
}

export function peakDayCommits(activity: DayActivityInfo[]): number {
  return activity.reduce((max, day) => Math.max(max, day.commits), 0)
}
