import type { DayActivityInfo } from '@current-client-bindings/app'
const COUNT_BANDS: { maxCount: number; intensity: number }[] = [
  { maxCount: 0, intensity: 0 },
  { maxCount: 1, intensity: 30 },
  { maxCount: 3, intensity: 55 },
  { maxCount: 6, intensity: 75 },
  { maxCount: Infinity, intensity: 100 },
]

export function activityIntensity(count: number): number {
  for (const band of COUNT_BANDS) {
    if (count <= band.maxCount) return band.intensity
  }
  return COUNT_BANDS[COUNT_BANDS.length - 1].intensity
}

export const ACTIVITY_INTENSITY_STEPS = COUNT_BANDS.map((band) => band.intensity)

interface CalendarCell extends DayActivityInfo {
  weekIndex: number
  dayOfWeek: number // 0 = Sunday .. 6 = Saturday
}

// Columns are weeks, rows are weekdays; the front is padded so day one lands on its weekday.
export function layoutActivityCalendar(activity: DayActivityInfo[]): CalendarCell[] {
  if (activity.length === 0) return []
  const padDays = new Date(activity[0].date + 'T00:00:00').getDay()
  return activity.map((d, i) => ({
    ...d,
    weekIndex: Math.floor((i + padDays) / 7),
    dayOfWeek: new Date(d.date + 'T00:00:00').getDay(),
  }))
}
