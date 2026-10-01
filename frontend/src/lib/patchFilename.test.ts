import { describe, expect, it } from 'vitest'
import { suggestedPatchFilename, suggestedWorkingTreePatchFilename } from './patchFilename'

describe('suggestedPatchFilename', () => {
  it('slugifies the subject and prefixes the short SHA', () => {
    expect(suggestedPatchFilename('abc1234567', 'Fix hunk navigation')).toBe('abc1234-fix-hunk-navigation.patch')
  })

  it('collapses punctuation into single dashes', () => {
    expect(suggestedPatchFilename('abc1234567', "Don't crash on empty diff!")).toBe(
      'abc1234-don-t-crash-on-empty-diff.patch',
    )
  })

  it('falls back to "patch" for a subject with no alphanumeric characters', () => {
    expect(suggestedPatchFilename('abc1234567', '...')).toBe('abc1234-patch.patch')
  })
})

describe('suggestedWorkingTreePatchFilename', () => {
  it('formats a zero-padded date-time stamp', () => {
    expect(suggestedWorkingTreePatchFilename(new Date(2026, 0, 5, 9, 3))).toBe('working-tree-2026-01-05-0903.patch')
  })
})
