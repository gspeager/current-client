import type { DayActivityInfo } from '@current-client-bindings/app'

export function peakDayCommits(activity: DayActivityInfo[]): number {
  return activity.reduce((max, day) => Math.max(max, day.commits), 0)
}
