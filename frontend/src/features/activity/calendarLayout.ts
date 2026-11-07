import type { DayActivityInfo } from '@current-client-bindings/app'

const COUNT_BANDS: { maxCount: number; intensity: number }[] = [
  { maxCount: 0, intensity: 0 },
  { maxCount: 1, intensity: 52 },
  { maxCount: 3, intensity: 68 },
  { maxCount: 6, intensity: 84 },
  { maxCount: Infinity, intensity: 100 },
]

export function activityIntensity(count: number): number {
  return COUNT_BANDS.find((band) => count <= band.maxCount)!.intensity
}

// A YYYY-MM-DD day at local midnight; a bare date string parses as UTC.
export function localDay(date: string): Date {
  return new Date(`${date}T00:00:00`)
}

export const ACTIVITY_INTENSITY_STEPS = COUNT_BANDS.map((band) => band.intensity)

interface CalendarCell extends DayActivityInfo {
  weekIndex: number
  dayOfWeek: number // 0 = Sunday .. 6 = Saturday
}

// Columns are weeks, rows are weekdays; the front is padded so day one lands on its weekday.
export function layoutActivityCalendar(activity: DayActivityInfo[]): CalendarCell[] {
  if (activity.length === 0) return []
  const padDays = localDay(activity[0].date).getDay()
  return activity.map((d, i) => ({
    ...d,
    weekIndex: Math.floor((i + padDays) / 7),
    dayOfWeek: localDay(d.date).getDay(),
  }))
}
