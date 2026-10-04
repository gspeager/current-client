import { localNameFor } from './branches'

describe('localNameFor', () => {
  it('drops the remote, keeping slashes in the branch', () => {
    expect(localNameFor('origin/fix/login', ['origin'])).toBe('fix/login')
  })

  it('handles a remote whose name contains a slash', () => {
    expect(localNameFor('team/origin/fix/login', ['origin', 'team/origin'])).toBe('fix/login')
  })

  it('prefers the longest matching remote', () => {
    expect(localNameFor('team/origin/fix', ['team', 'team/origin'])).toBe('fix')
    expect(localNameFor('team/feature', ['team', 'team/origin'])).toBe('feature')
  })

  it('falls back to the first slash when the remotes are unknown', () => {
    expect(localNameFor('origin/fix/login', [])).toBe('fix/login')
  })
})
