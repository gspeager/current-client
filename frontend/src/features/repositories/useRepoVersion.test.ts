import { act, renderHook } from '@testing-library/react'
import { useRepoVersion } from './useRepoVersion'

describe('useRepoVersion', () => {
  afterEach(() => vi.useRealTimers())

  it('says it was just bumped for a second after each bump', () => {
    vi.useFakeTimers()
    const { result } = renderHook(() => useRepoVersion())
    expect(result.current.justBumped()).toBe(false)

    act(() => result.current.bumpRepoVersion())
    expect(result.current.repoVersion).toBe(1)
    vi.advanceTimersByTime(500)
    expect(result.current.justBumped()).toBe(true)

    vi.advanceTimersByTime(600)
    expect(result.current.justBumped()).toBe(false)
  })
})
