import { act, renderHook } from '@testing-library/react'
import { useTabMemory } from './useTabMemory'

describe('useTabMemory', () => {
  it('starts every repo on the activity view with empty memory', () => {
    const { result } = renderHook(() => useTabMemory('/repos/app'))
    expect(result.current.activeTab).toBe('activity')
    expect(result.current.restored).toEqual({
      selectedFilePath: null,
      selectedCommitSha: null,
      commitDraft: '',
      historyTopSha: null,
    })
  })

  it('remembers the active view and view state per repo', () => {
    const { result, rerender } = renderHook(({ path }) => useTabMemory(path), {
      initialProps: { path: '/repos/app' },
    })
    act(() => result.current.setActiveTab('history'))
    result.current.remember({ commitDraft: 'Fix login' })
    result.current.remember({ selectedCommitSha: 'abc123' })

    rerender({ path: '/repos/server' })
    expect(result.current.activeTab).toBe('activity')
    expect(result.current.restored.commitDraft).toBe('')

    rerender({ path: '/repos/app' })
    expect(result.current.activeTab).toBe('history')
    expect(result.current.restored).toMatchObject({ commitDraft: 'Fix login', selectedCommitSha: 'abc123' })
  })

  it('restores the latest write on the next render, with no re-render in between', () => {
    const { result, rerender } = renderHook(() => useTabMemory('/repos/app'))
    result.current.remember({ commitDraft: 'W' })
    result.current.remember({ selectedFilePath: 'src/app.ts' })
    result.current.remember({ commitDraft: 'WIP' })
    rerender()
    expect(result.current.restored).toMatchObject({ commitDraft: 'WIP', selectedFilePath: 'src/app.ts' })
  })

  it('ignores writes while no repo is open', () => {
    const { result, rerender } = renderHook(({ path }) => useTabMemory(path), {
      initialProps: { path: null as string | null },
    })
    result.current.remember({ commitDraft: 'lost' })
    act(() => result.current.setActiveTab('history'))
    rerender({ path: '/repos/app' })
    expect(result.current.activeTab).toBe('activity')
    expect(result.current.restored.commitDraft).toBe('')
  })
})
