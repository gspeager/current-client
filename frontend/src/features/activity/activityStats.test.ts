import type { DayActivityInfo } from '@current-client-bindings/app'
import { describe, expect, it } from 'vitest'
import { currentStreak, longestStreak, peakDayCommits } from './activityStats'

function days(counts: number[]): DayActivityInfo[] {
  return counts.map((commits, i) => ({ date: `2024-01-${String(i + 1).padStart(2, '0')}`, commits }))
}

describe('longestStreak', () => {
  it('finds the longest run of commit days anywhere in the range', () => {
    expect(longestStreak(days([1, 1, 0, 1, 1, 1, 0]))).toBe(3)
  })

  it('returns 0 for no activity at all', () => {
    expect(longestStreak(days([0, 0, 0]))).toBe(0)
  })

  it('handles a streak that runs to the end of the range', () => {
    expect(longestStreak(days([0, 1, 1, 1]))).toBe(3)
  })
})

describe('currentStreak', () => {
  it('counts back from the last day while commits continue', () => {
    expect(currentStreak(days([1, 0, 1, 1, 1]))).toBe(3)
  })

  it('is 0 when the most recent day has no commits, even after a long streak', () => {
    expect(currentStreak(days([1, 1, 1, 0]))).toBe(0)
  })

  it('is 0 for an empty range', () => {
    expect(currentStreak([])).toBe(0)
  })
})

describe('peakDayCommits', () => {
  it('returns the highest single-day count', () => {
    expect(peakDayCommits(days([1, 5, 2]))).toBe(5)
  })

  it('returns 0 for no activity', () => {
    expect(peakDayCommits(days([0, 0]))).toBe(0)
  })
})
