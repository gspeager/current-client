import { plural } from './plural'

describe('plural', () => {
  it('adds an s past one', () => {
    expect(plural(1, 'commit')).toBe('1 commit')
    expect(plural(0, 'commit')).toBe('0 commits')
    expect(plural(3, 'commit')).toBe('3 commits')
  })

  it('takes an irregular plural', () => {
    expect(plural(2, 'branch', 'branches')).toBe('2 branches')
  })
})
