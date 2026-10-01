import { relativeTime, sinceDaysAgo } from './relativeTime'

const now = new Date('2026-01-10T12:00:00Z')

describe('relativeTime', () => {
  it('uses the largest whole unit', () => {
    expect(relativeTime(new Date('2026-01-10T11:59:55Z'), now)).toBe('just now')
    expect(relativeTime(new Date('2026-01-10T11:59:30Z'), now)).toBe('30s ago')
    expect(relativeTime(new Date('2026-01-10T11:58:00Z'), now)).toBe('2m ago')
    expect(relativeTime(new Date('2026-01-10T09:00:00Z'), now)).toBe('3h ago')
    expect(relativeTime(new Date('2026-01-07T12:00:00Z'), now)).toBe('3d ago')
  })
})

describe('sinceDaysAgo', () => {
  it('returns no bound for zero days', () => {
    expect(sinceDaysAgo(0, now)).toBe('')
  })

  it('returns an ISO timestamp the given number of days back', () => {
    expect(sinceDaysAgo(7, now)).toBe('2026-01-03T12:00:00.000Z')
  })
})
