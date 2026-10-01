import { describe, expect, it } from 'vitest'
import { countHealthIssues, isStaleBranch } from './repoHealth'

describe('isStaleBranch', () => {
  const now = new Date('2024-06-01T00:00:00Z')

  it('flags a branch untouched for 30+ days', () => {
    expect(isStaleBranch('2024-04-01T00:00:00Z', now)).toBe(true)
  })

  it('does not flag a recently-touched branch', () => {
    expect(isStaleBranch('2024-05-25T00:00:00Z', now)).toBe(false)
  })
})

describe('countHealthIssues', () => {
  it('is 0 when every check passes', () => {
    expect(countHealthIssues({ staleBranchCount: 0, unpushedCount: 0, workingTreeClean: true })).toBe(0)
  })

  it('counts one point per failing check, not per unit', () => {
    expect(countHealthIssues({ staleBranchCount: 5, unpushedCount: 3, workingTreeClean: false })).toBe(3)
  })

  it('counts a single failing check', () => {
    expect(countHealthIssues({ staleBranchCount: 0, unpushedCount: 0, workingTreeClean: false })).toBe(1)
  })
})
