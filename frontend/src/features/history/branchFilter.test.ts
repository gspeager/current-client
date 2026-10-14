import { ALL_BRANCHES, isBranchName, refFromSaved, savedFromRef } from './branchFilter'

describe('branch filter setting', () => {
  it('defaults to all branches', () => {
    expect(refFromSaved(undefined)).toBe(ALL_BRANCHES)
    expect(refFromSaved('')).toBe(ALL_BRANCHES)
  })

  it('round-trips each choice', () => {
    for (const ref of [ALL_BRANCHES, '', 'feature/login']) {
      expect(refFromSaved(savedFromRef(ref))).toBe(ref)
    }
    expect(savedFromRef('')).toBe('HEAD')
  })

  it('tells a branch name from the two fixed choices', () => {
    expect(isBranchName('main')).toBe(true)
    expect(isBranchName(ALL_BRANCHES)).toBe(false)
    expect(isBranchName('')).toBe(false)
  })
})
