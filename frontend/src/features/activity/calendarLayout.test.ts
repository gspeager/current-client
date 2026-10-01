import { describe, expect, it } from 'vitest'
import { activityIntensity, layoutActivityCalendar } from './calendarLayout'

describe('activityIntensity', () => {
  it('returns 0 for no commits', () => {
    expect(activityIntensity(0)).toBe(0)
  })

  it('steps up through fixed bands rather than scaling continuously', () => {
    expect(activityIntensity(1)).toBe(30)
    expect(activityIntensity(2)).toBe(55)
    expect(activityIntensity(3)).toBe(55)
    expect(activityIntensity(4)).toBe(75)
    expect(activityIntensity(10)).toBe(100)
  })
})

describe('layoutActivityCalendar', () => {
  it('returns empty for no data', () => {
    expect(layoutActivityCalendar([])).toEqual([])
  })

  it('places each day in its correct weekday row and pads the first week', () => {
    // 2024-01-01 is a Monday (dayOfWeek 1); the function expects a dense,
    // consecutive-day array, the same shape git.CommitActivity produces.
    const dates = [
      '2024-01-01',
      '2024-01-02',
      '2024-01-03',
      '2024-01-04',
      '2024-01-05',
      '2024-01-06',
      '2024-01-07',
      '2024-01-08',
    ]
    const got = layoutActivityCalendar(dates.map((date) => ({ date, commits: 0 })))
    // A leading pad cell stands in for the Sunday before Jan 1, so Jan 1
    // (Monday) through Jan 6 (Saturday) share week 0; the next Sunday,
    // Jan 7, starts week 1.
    expect(got[0]).toMatchObject({ date: '2024-01-01', dayOfWeek: 1, weekIndex: 0 })
    expect(got[5]).toMatchObject({ date: '2024-01-06', dayOfWeek: 6, weekIndex: 0 })
    expect(got[6]).toMatchObject({ date: '2024-01-07', dayOfWeek: 0, weekIndex: 1 })
    expect(got[7]).toMatchObject({ date: '2024-01-08', dayOfWeek: 1, weekIndex: 1 })
  })
})
