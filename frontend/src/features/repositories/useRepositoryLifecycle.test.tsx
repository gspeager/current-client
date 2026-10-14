import type { ReactNode } from 'react'
import { act, renderHook } from '@testing-library/react'
import { RepositoryService, type OpenTabsInfo, type RepoSummaryInfo } from '@current-client-bindings/app'
import { DialogProvider } from '../../components/chrome/DialogProvider'
import { useRepositoryLifecycle } from './useRepositoryLifecycle'

function wrapper({ children }: { children: ReactNode }) {
  return <DialogProvider>{children}</DialogProvider>
}

describe('useRepositoryLifecycle', () => {
  beforeEach(() => {
    vi.mocked(RepositoryService.GetRecentRepositories).mockResolvedValue([])
    vi.mocked(RepositoryService.SetOpenTabs).mockResolvedValue()
    vi.mocked(RepositoryService.GetTabSummaries).mockImplementation(
      (paths) =>
        Promise.resolve(paths.map((path) => ({ path, available: true }) as RepoSummaryInfo)) as ReturnType<
          typeof RepositoryService.GetTabSummaries
        >,
    )
    vi.mocked(RepositoryService.OpenRepository).mockImplementation(
      (path) => Promise.resolve(path) as ReturnType<typeof RepositoryService.OpenRepository>,
    )
  })

  it('summarizes the other tabs after the focused one opens, and a tab again on leaving it', async () => {
    vi.mocked(RepositoryService.GetOpenTabs).mockResolvedValue({
      tabs: ['/a', '/b', '/c'],
      focused: '/b',
    } as OpenTabsInfo)
    const { result } = renderHook(() => useRepositoryLifecycle(), { wrapper })

    await vi.waitFor(() => expect(result.current.tabSummaries['/c']).toBeDefined())
    expect(RepositoryService.GetTabSummaries).toHaveBeenCalledTimes(1)
    expect(RepositoryService.GetTabSummaries).toHaveBeenCalledWith(['/a', '/c'])
    expect(vi.mocked(RepositoryService.OpenRepository).mock.invocationCallOrder[0]).toBeLessThan(
      vi.mocked(RepositoryService.GetTabSummaries).mock.invocationCallOrder[0],
    )

    await act(async () => result.current.openRecent('/a'))

    await vi.waitFor(() => expect(RepositoryService.GetTabSummaries).toHaveBeenLastCalledWith(['/b']))
    expect(RepositoryService.GetRepoSummaries).not.toHaveBeenCalled()
  })
})
