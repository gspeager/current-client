import { act, renderHook } from '@testing-library/react'
import { useTabMemory } from './useTabMemory'

describe('useTabMemory', () => {
  it('starts every repo on Changes with empty memory', () => {
    const { result } = renderHook(() => useTabMemory('/repos/app'))
    expect(result.current.activeTab).toBe('working-copy')
    expect(result.current.restored).toEqual({
      selectedFilePath: null,
      selectedCommitSha: null,
      commitDraft: '',
      historyTopSha: null,
      diffScrollTop: 0,
    })
  })

  describe('per branch', () => {
    const onBranch = (branch: string | null) =>
      renderHook(({ name }) => useTabMemory('/repos/app', name), { initialProps: { name: branch } })

    it('switching branches and back restores the file, scroll position and draft', () => {
      const { result, rerender } = onBranch('main')
      result.current.remember({ selectedFilePath: 'a.ts', commitDraft: 'Fix login', diffScrollTop: 120 })
      rerender({ name: 'side' })
      result.current.remember({ selectedFilePath: 'b.ts', commitDraft: 'Other work', diffScrollTop: 40 })

      rerender({ name: 'main' })
      expect(result.current.restored).toMatchObject({
        selectedFilePath: 'a.ts',
        commitDraft: 'Fix login',
        diffScrollTop: 120,
      })

      rerender({ name: 'side' })
      expect(result.current.restored).toMatchObject({
        selectedFilePath: 'b.ts',
        commitDraft: 'Other work',
        diffScrollTop: 40,
      })
    })

    it('a branch not seen yet starts from the one before it, without changing that one', () => {
      const { result, rerender } = onBranch('main')
      result.current.remember({ selectedFilePath: 'a.ts', commitDraft: 'Fix login', diffScrollTop: 120 })

      rerender({ name: 'fix/login' })
      expect(result.current.restored).toMatchObject({
        selectedFilePath: 'a.ts',
        commitDraft: 'Fix login',
        diffScrollTop: 0,
      })
      result.current.remember({ commitDraft: '' })

      rerender({ name: 'main' })
      expect(result.current.restored.commitDraft).toBe('Fix login')
    })

    it("doesn't carry a rebase's message onto the branch it lands on", () => {
      const { result, rerender } = renderHook(({ name, op }) => useTabMemory('/repos/app', name, op), {
        initialProps: { name: 'topic' as string | null, op: false },
      })
      result.current.remember({ commitDraft: 'A draft on topic' })
      rerender({ name: '', op: true })
      result.current.remember({ commitDraft: 'The commit being replayed' })

      rerender({ name: '', op: false })
      rerender({ name: 'other', op: false })
      expect(result.current.restored.commitDraft).toBe('')

      rerender({ name: 'topic', op: false })
      expect(result.current.restored.commitDraft).toBe('A draft on topic')
    })

    it('carries a draft from a detached HEAD with no operation', () => {
      const { result, rerender } = onBranch('')
      result.current.remember({ commitDraft: 'Save this' })
      rerender({ name: 'saved' })
      expect(result.current.restored.commitDraft).toBe('Save this')
    })

    it('keeps the last branch while the next answer is loading', () => {
      const { result, rerender } = onBranch('main')
      result.current.remember({ commitDraft: 'Fix login' })

      rerender({ name: null })
      expect(result.current.memoryBranch).toBe('main')
      expect(result.current.restored.commitDraft).toBe('Fix login')
    })

    it('keeps what was written before the branch was known', () => {
      const { result, rerender } = onBranch(null)
      result.current.remember({ commitDraft: 'Early' })

      rerender({ name: 'main' })
      expect(result.current.restored.commitDraft).toBe('Early')
    })

    it('drops a deleted branch, so a new one of that name starts clean', () => {
      const { result, rerender } = onBranch('side')
      result.current.remember({ commitDraft: 'Other work' })
      rerender({ name: 'main' })
      result.current.remember({ commitDraft: 'Fix login' })

      result.current.forgetBranchesExcept(['main'])

      rerender({ name: 'side' })
      expect(result.current.restored.commitDraft).toBe('Fix login')
      result.current.remember({ commitDraft: 'New' })
      rerender({ name: 'main' })
      expect(result.current.restored.commitDraft).toBe('Fix login')
    })

    it('keeps history state across branches', () => {
      const { result, rerender } = onBranch('main')
      result.current.remember({ selectedCommitSha: 'abc123' })

      rerender({ name: 'side' })
      expect(result.current.restored.selectedCommitSha).toBe('abc123')
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
    expect(result.current.activeTab).toBe('working-copy')
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
    expect(result.current.activeTab).toBe('working-copy')
    expect(result.current.restored.commitDraft).toBe('')
  })
})
