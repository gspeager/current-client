import { describe, expect, it } from 'vitest'
import { ageIntensity } from './blameAge'

describe('ageIntensity', () => {
  const now = new Date('2026-01-15T12:00:00Z')

  it('returns the highest band for a commit from today', () => {
    expect(ageIntensity(new Date('2026-01-15T06:00:00Z'), now)).toBe(100)
  })

  it('returns a mid band for a commit a couple weeks old', () => {
    expect(ageIntensity(new Date('2026-01-01T12:00:00Z'), now)).toBe(50)
  })

  it('returns the lowest band for a commit over a year old', () => {
    expect(ageIntensity(new Date('2020-01-15T12:00:00Z'), now)).toBe(15)
  })

  it('steps rather than interpolates between bands', () => {
    const justUnderOneDay = ageIntensity(new Date('2026-01-14T13:00:00Z'), now)
    const justOverOneDay = ageIntensity(new Date('2026-01-14T11:00:00Z'), now)
    expect(justUnderOneDay).toBe(100)
    expect(justOverOneDay).toBe(75)
  })
})
