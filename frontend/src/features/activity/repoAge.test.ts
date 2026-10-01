import { describe, expect, it } from 'vitest'
import { ageColor, formatAge } from './repoAge'

describe('formatAge', () => {
  it('formats days into a terse label per band', () => {
    expect(formatAge(0)).toBe('Today')
    expect(formatAge(5)).toBe('5d')
    expect(formatAge(60)).toBe('2mo')
    expect(formatAge(400)).toBe('1y')
  })
})

describe('ageColor', () => {
  it('steps through fixed bands rather than a continuous scale', () => {
    expect(ageColor(0)).toBe('var(--tertiary)')
    expect(ageColor(7)).toBe('var(--tertiary)')
    expect(ageColor(8)).toBe('var(--on-surface)')
    expect(ageColor(365)).toBe('var(--on-surface)')
    expect(ageColor(366)).toBe('var(--outline)')
  })
})
