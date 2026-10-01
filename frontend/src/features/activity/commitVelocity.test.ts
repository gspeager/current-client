import type { DayActivityInfo } from '@current-client-bindings/app'
import { describe, expect, it } from 'vitest'
import {
  averageWeeklyCommits,
  busiestWeekday,
  lastWeekVsAverage,
  weeklyBuckets,
  type WeekBucket,
} from './commitVelocity'

function days(dates: [string, number][]): DayActivityInfo[] {
  return dates.map(([date, commits]) => ({ date, commits }))
}

describe('weeklyBuckets', () => {
  it('sums 7-day chunks starting at index 0', () => {
    const activity = days([
      ['2024-01-01', 1],
      ['2024-01-02', 2],
      ['2024-01-03', 0],
      ['2024-01-04', 0],
      ['2024-01-05', 0],
      ['2024-01-06', 0],
      ['2024-01-07', 0],
      ['2024-01-08', 3],
    ])
    const got = weeklyBuckets(activity)
    expect(got).toEqual([
      { weekStart: '2024-01-01', commits: 3 },
      { weekStart: '2024-01-08', commits: 3 },
    ])
  })

  it('returns an empty list for no data', () => {
    expect(weeklyBuckets([])).toEqual([])
  })
})

describe('busiestWeekday', () => {
  it('returns null when there is no activity at all', () => {
    const activity = days([
      ['2024-01-01', 0],
      ['2024-01-02', 0],
    ])
    expect(busiestWeekday(activity)).toBeNull()
  })

  it('returns the weekday with the most commits summed across the range', () => {
    // 2024-01-01/08 are Mondays (dayOfWeek 1).
    const activity = days([
      ['2024-01-01', 5],
      ['2024-01-02', 1],
      ['2024-01-08', 4],
    ])
    expect(busiestWeekday(activity)).toEqual({ day: 'Monday', commits: 9 })
  })
})

function weeks(commits: number[]): WeekBucket[] {
  return commits.map((c, i) => ({ weekStart: `W${i + 1}`, commits: c }))
}

describe('averageWeeklyCommits', () => {
  it('averages commits across all weeks', () => {
    expect(averageWeeklyCommits(weeks([10, 20, 30]))).toBe(20)
  })

  it('returns 0 for no weeks', () => {
    expect(averageWeeklyCommits([])).toBe(0)
  })
})

describe('lastWeekVsAverage', () => {
  it('reports a sprint burst when the last week is above average', () => {
    expect(lastWeekVsAverage(weeks([10, 10, 10, 20]))).toEqual({ percent: 60, label: 'sprint burst' })
  })

  it('reports below average when the last week trails the average', () => {
    // Average of [20, 20, 20, 10] is 17.5; 10 is 42.9% below that.
    expect(lastWeekVsAverage(weeks([20, 20, 20, 10]))).toEqual({ percent: -43, label: 'below average' })
  })

  it('reports a steady pace when the last week matches the average exactly', () => {
    expect(lastWeekVsAverage(weeks([10, 10, 10]))).toEqual({ percent: 0, label: 'steady pace' })
  })

  it('returns null when the whole window has no commits', () => {
    expect(lastWeekVsAverage(weeks([0, 0, 0]))).toBeNull()
  })

  it('returns null for an empty window', () => {
    expect(lastWeekVsAverage([])).toBeNull()
  })
})
