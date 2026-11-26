import { act, renderHook } from '@testing-library/react'
import { usePerRepository } from './usePerRepository'

describe('usePerRepository', () => {
  it('starts over in another repository and on coming back', () => {
    const { result, rerender } = renderHook(({ path }) => usePerRepository(path, false), {
      initialProps: { path: '/a' as string | null },
    })
    act(() => result.current[1](true))
    expect(result.current[0]).toBe(true)
    rerender({ path: '/b' })
    expect(result.current[0]).toBe(false)
    rerender({ path: '/a' })
    expect(result.current[0]).toBe(false)
  })

  it('drops a late call from the repository before', () => {
    const { result, rerender } = renderHook(({ path }) => usePerRepository(path, 0), {
      initialProps: { path: '/a' as string | null },
    })
    const setForA = result.current[1]
    rerender({ path: '/b' })
    act(() => result.current[1](2))
    act(() => setForA(1))
    expect(result.current[0]).toBe(2)
  })
})
