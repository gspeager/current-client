import { act, renderHook, waitFor } from '@testing-library/react'
import { useAsyncData } from './useAsyncData'

function deferred<T>() {
  let resolve!: (value: T) => void
  let reject!: (reason: unknown) => void
  const promise = new Promise<T>((res, rej) => {
    resolve = res
    reject = rej
  })
  return { promise, resolve, reject }
}

describe('useAsyncData', () => {
  it('loads data and reports loading while in flight', async () => {
    const request = deferred<string>()
    const { result } = renderHook(() => useAsyncData(() => request.promise, []))
    expect(result.current.loading).toBe(true)
    expect(result.current.data).toBeNull()

    await act(async () => request.resolve('main'))
    expect(result.current).toMatchObject({ data: 'main', error: null, loading: false })
  })

  it('keeps the newest result when responses arrive out of order', async () => {
    const requests: Record<string, ReturnType<typeof deferred<string>>> = { a: deferred(), b: deferred() }
    const { result, rerender } = renderHook(({ path }) => useAsyncData(() => requests[path].promise, [path]), {
      initialProps: { path: 'a' },
    })
    rerender({ path: 'b' })

    await act(async () => requests.b.resolve('diff of b'))
    await act(async () => requests.a.resolve('diff of a'))
    expect(result.current.data).toBe('diff of b')
  })

  it('clears data when deps change but keeps it across a refresh key change', async () => {
    let next = deferred<string>()
    const { result, rerender } = renderHook(
      ({ repo, refreshKey }) => useAsyncData(() => (repo ? next.promise : null), [repo], { refreshKey }),
      { initialProps: { repo: 'one', refreshKey: 0 } },
    )
    await act(async () => next.resolve('first'))

    next = deferred()
    rerender({ repo: 'one', refreshKey: 1 })
    expect(result.current).toMatchObject({ data: 'first', loading: true })

    await act(async () => next.resolve('refreshed'))
    next = deferred()
    rerender({ repo: 'two', refreshKey: 1 })
    expect(result.current).toMatchObject({ data: null, loading: true })
  })

  it('keeps data across a deps change when keepData is set', async () => {
    let next = deferred<number>()
    const { result, rerender } = renderHook(
      ({ commits }) => useAsyncData(() => (commits.length > 0 ? next.promise : null), [commits], { keepData: true }),
      { initialProps: { commits: ['a'] } },
    )
    await act(async () => next.resolve(1))

    next = deferred()
    rerender({ commits: ['a', 'b'] })
    expect(result.current).toMatchObject({ data: 1, loading: true })
  })

  it('reload refetches and keeps the current data meanwhile', async () => {
    let calls = 0
    const { result } = renderHook(() => useAsyncData(() => Promise.resolve(++calls), []))
    await waitFor(() => expect(result.current.data).toBe(1))

    act(() => result.current.reload())
    expect(result.current.data).toBe(1)
    await waitFor(() => expect(result.current.data).toBe(2))
  })

  it('surfaces a rejection as its message and drops the data', async () => {
    const { result } = renderHook(() => useAsyncData(() => Promise.reject(new Error('Git command failed.')), []))
    await waitFor(() => expect(result.current.error).toBe('Git command failed.'))
    expect(result.current.data).toBeNull()
  })

  it('stays idle when there is nothing to load', () => {
    const { result } = renderHook(() => useAsyncData<string>(() => null, []))
    expect(result.current).toMatchObject({ data: null, error: null, loading: false })
  })

  it('ignores a response that arrives after unmount', async () => {
    const request = deferred<string>()
    const { result, unmount } = renderHook(() => useAsyncData(() => request.promise, []))
    const before = result.current
    unmount()
    await act(async () => request.resolve('late'))
    expect(result.current).toBe(before)
  })
})
