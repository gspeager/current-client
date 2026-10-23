import { toggled } from './sets'

describe('toggled', () => {
  it('flips membership without touching the original', () => {
    const original = new Set(['a'])
    expect([...toggled(original, 'a')]).toEqual([])
    expect([...toggled(original, 'b')]).toEqual(['a', 'b'])
    expect([...original]).toEqual(['a'])
  })

  it('sets membership when told which way', () => {
    expect(toggled(new Set(['a']), 'a', true).has('a')).toBe(true)
    expect(toggled(new Set<string>(), 'a', false).has('a')).toBe(false)
  })
})
