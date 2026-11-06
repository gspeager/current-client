import { describe, expect, it } from 'vitest'
import { formatAge } from './repoAge'

describe('formatAge', () => {
  it('formats days into a terse label per band', () => {
    expect(formatAge(0)).toBe('Today')
    expect(formatAge(5)).toBe('5d')
    expect(formatAge(60)).toBe('2mo')
    expect(formatAge(400)).toBe('1y')
  })
})

