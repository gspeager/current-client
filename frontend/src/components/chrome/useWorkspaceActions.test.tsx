import { renderHook } from '@testing-library/react'
import type { ReactNode } from 'react'
import { Links } from '@current-client-bindings/core/hostlink'
import { DialogProvider } from './DialogProvider'
import { useWorkspaceActions } from './useWorkspaceActions'

const wrapper = ({ children }: { children: ReactNode }) => <DialogProvider>{children}</DialogProvider>

describe('useWorkspaceActions', () => {
  it('offers the repository on the host only when a remote points at one', () => {
    const base = { repoPath: '/repos/app', onBranchChanged: vi.fn(), onOpenCommit: vi.fn(), onFailure: vi.fn() }
    const without = renderHook(() => useWorkspaceActions(base), { wrapper })
    expect(without.result.current.actions.map((a) => a.id)).not.toContain('open-on-host')

    const withHost = renderHook(
      () => useWorkspaceActions({ ...base, hostLinks: new Links({ project: 'https://gitlab.com/g/r' }) }),
      { wrapper },
    )
    const action = withHost.result.current.actions.find((a) => a.id === 'open-on-host')
    expect(action?.label).toBe('Open repository on host')
  })
})

describe('useWorkspaceActions during a merge', () => {
  it('disables undo and bisect with the reason, and keeps the rest live', () => {
    const { result } = renderHook(
      () =>
        useWorkspaceActions({
          repoPath: '/repos/app',
          operation: 'rebase',
          onBranchChanged: vi.fn(),
          onOpenCommit: vi.fn(),
          onFailure: vi.fn(),
        }),
      { wrapper },
    )
    const byId = Object.fromEntries(result.current.actions.map((a) => [a.id, a]))
    expect(byId.undo.disabled).toBe(true)
    expect(byId.undo.reason).toBe('Finish the rebase first')
    expect(byId.bisect.disabled).toBe(true)
    expect(byId.reflog.disabled).toBeUndefined()
    expect(byId['open-terminal'].disabled).toBeUndefined()
  })
})
