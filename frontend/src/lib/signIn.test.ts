import { describe, expect, it, vi } from 'vitest'
import { needsSignIn, withSignIn } from './signIn'

const NEEDS = new Error(
  'This remote needs credentials. Set up a credential helper, such as Git Credential Manager, for it.',
)
const OTHER = new Error('Can’t reach the remote host. Check the network connection.')

describe('withSignIn', () => {
  it('runs once without credentials when the remote accepts that', async () => {
    const start = vi.fn().mockResolvedValue('ok')
    const signIn = vi.fn()
    await expect(withSignIn(start, signIn)).resolves.toBe('ok')
    expect(start).toHaveBeenCalledWith(null)
    expect(signIn).not.toHaveBeenCalled()
  })

  it('passes other failures straight through', async () => {
    const signIn = vi.fn()
    await expect(withSignIn(() => Promise.reject(OTHER), signIn)).rejects.toBe(OTHER)
    expect(signIn).not.toHaveBeenCalled()
  })

  it('retries with what the user enters, saying when the last try was rejected', async () => {
    const start = vi.fn().mockRejectedValueOnce(NEEDS).mockRejectedValueOnce(NEEDS).mockResolvedValueOnce('ok')
    const signIn = vi
      .fn()
      .mockResolvedValueOnce({ username: 'a', password: '1' })
      .mockResolvedValueOnce({ username: 'a', password: '2' })

    await expect(withSignIn(start, signIn)).resolves.toBe('ok')
    expect(signIn.mock.calls).toEqual([[{ rejected: false }], [{ rejected: true }]])
    expect(start).toHaveBeenLastCalledWith({ username: 'a', password: '2' })
  })

  it('rethrows the failure when the user cancels', async () => {
    await expect(
      withSignIn(
        () => Promise.reject(NEEDS),
        () => Promise.resolve(null),
      ),
    ).rejects.toBe(NEEDS)
  })
})

describe('needsSignIn', () => {
  it('matches only the credential messages', () => {
    expect(needsSignIn(NEEDS)).toBe(true)
    expect(needsSignIn(new Error('Authentication failed. Check the credentials Git uses for this remote.'))).toBe(true)
    expect(needsSignIn(OTHER)).toBe(false)
  })
})
