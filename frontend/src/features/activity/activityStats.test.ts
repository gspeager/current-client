import type { DayActivityInfo } from '@current-client-bindings/app'
import { describe, expect, it } from 'vitest'
import { peakDayCommits } from './activityStats'

function days(counts: number[]): DayActivityInfo[] {
  return counts.map((commits, i) => ({ date: `2024-01-${String(i + 1).padStart(2, '0')}`, commits }))
}

describe('peakDayCommits', () => {
  it('returns the highest single-day count', () => {
    expect(peakDayCommits(days([1, 5, 2]))).toBe(5)
  })

  it('returns 0 for no activity', () => {
    expect(peakDayCommits(days([0, 0]))).toBe(0)
  })
})
