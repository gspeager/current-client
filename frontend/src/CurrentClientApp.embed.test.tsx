import { createRef } from 'react'
import { act, render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import {
  BranchService,
  GitService,
  HistoryService,
  HostService,
  RemoteService,
  RepositoryService,
  SettingsService,
  StatusService,
  WorktreeService,
  type OpenTabsInfo,
  type Settings,
} from '@current-client-bindings/app'
import { giveElementsLayout } from './test/bindings'
import CurrentClientApp, { type CurrentClientAppHandle } from './CurrentClientApp'

function stubStartup(tabs: string[], focused = '') {
  vi.mocked(GitService.GetGitVersion).mockResolvedValue('2.53.0')
  vi.mocked(SettingsService.GetSettings).mockResolvedValue({ theme: 'dark' } as Settings)
  vi.mocked(RepositoryService.GetRecentRepositories).mockResolvedValue([])
  vi.mocked(RepositoryService.GetOpenTabs).mockResolvedValue({ tabs, focused } as OpenTabsInfo)
  vi.mocked(RepositoryService.GetTabSummaries).mockResolvedValue([])
  vi.mocked(RepositoryService.SetOpenTabs).mockResolvedValue()
  vi.mocked(RepositoryService.OpenRepository).mockImplementation(
    (path) => Promise.resolve(path) as ReturnType<typeof RepositoryService.OpenRepository>,
  )
  vi.mocked(HistoryService.FollowsConventionalCommits).mockResolvedValue(false)
}

const toggle = <button type="button">Friendly view</button>

beforeEach(() => giveElementsLayout())

describe('CurrentClientApp embedding', () => {
  it('shows the host accessory on the welcome screen and in the header', async () => {
    stubStartup([])
    const { rerender } = render(<CurrentClientApp headerAccessory={toggle} />)
    expect(await screen.findByRole('button', { name: 'Friendly view' })).toBeInTheDocument()

    rerender(<CurrentClientApp headerAccessory={toggle} activeRepoPath="/repos/app" />)
    await waitFor(() =>
      expect(document.querySelector('.header-bar')).toContainElement(screen.getByText('Friendly view')),
    )
  })

  it('opens the host repository instead of the saved tab and reports changes back', async () => {
    stubStartup(['/repos/a', '/repos/b'], '/repos/a')
    const onActiveRepoChange = vi.fn()
    const { rerender } = render(<CurrentClientApp activeRepoPath="/repos/b" onActiveRepoChange={onActiveRepoChange} />)

    await waitFor(() => expect(onActiveRepoChange).toHaveBeenLastCalledWith('/repos/b'))
    expect(RepositoryService.OpenRepository).not.toHaveBeenCalledWith('/repos/a')

    rerender(<CurrentClientApp activeRepoPath="/repos/a" onActiveRepoChange={onActiveRepoChange} />)
    await waitFor(() => expect(onActiveRepoChange).toHaveBeenLastCalledWith('/repos/a'))
  })

  it('opens a branch in History through the handle', async () => {
    stubStartup(['/repos/app'], '/repos/app')
    vi.mocked(HistoryService.ResolveCommit).mockResolvedValue('a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0')
    vi.mocked(HistoryService.GetHistory).mockResolvedValue([])
    const handle = createRef<CurrentClientAppHandle>()
    render(<CurrentClientApp ref={handle} />)
    await waitFor(() => expect(RepositoryService.OpenRepository).toHaveBeenCalledWith('/repos/app'))
    await waitFor(() => expect(document.querySelector('.header-bar')).not.toBeNull())

    await handle.current!.openBranch('feature/login')

    expect(HistoryService.ResolveCommit).toHaveBeenCalledWith('/repos/app', 'feature/login')
    await waitFor(() =>
      expect(HistoryService.GetHistory).toHaveBeenCalledWith(
        '/repos/app',
        expect.any(Number),
        0,
        expect.objectContaining({ ref: 'feature/login' }),
      ),
    )
  })

  it('opens a repository on Changes, the first view', async () => {
    stubStartup(['/repos/app'], '/repos/app')
    render(<CurrentClientApp />)
    const workingCopy = await screen.findByRole('button', { name: /^Changes/ })

    const views = [...document.querySelectorAll('.tab-bar button')].map((b) => b.textContent)
    expect(views).toEqual([expect.stringMatching(/^Changes/), 'History', 'Activity', 'Releases', 'Explore'])
    expect(workingCopy).toHaveClass('tab-active')
  })

  it('leaves Explore out when the host app hides it', async () => {
    stubStartup(['/repos/app'], '/repos/app')
    render(<CurrentClientApp hideExplore />)
    await screen.findByRole('button', { name: /^Changes/ })
    expect(screen.queryByRole('button', { name: 'Explore' })).toBeNull()
  })

  it('does not reload a tab that is visited again with no new commits', async () => {
    stubStartup(['/repos/app'], '/repos/app')
    vi.mocked(HistoryService.GetHistory).mockResolvedValue([])
    render(<CurrentClientApp />)
    const historyTab = await screen.findByRole('button', { name: 'History' })

    await userEvent.click(historyTab)
    await waitFor(() => expect(HistoryService.GetHistory).toHaveBeenCalled())
    await new Promise((resolve) => setTimeout(resolve, 50))
    // The sidebar's HEAD row reads one commit whenever it reappears; the list reads pages.
    const listLoads = () => vi.mocked(HistoryService.GetHistory).mock.calls.filter(([, limit]) => limit !== 1).length
    const loads = listLoads()

    await userEvent.click(screen.getByRole('button', { name: 'Activity' }))
    await userEvent.click(historyTab)
    await new Promise((resolve) => setTimeout(resolve, 50))

    expect(listLoads()).toBe(loads)
  })

  describe('while hidden', () => {
    const github = [{ hostname: 'github.com', provider: 'github', enabled: false, dismissed: false, managed: false }]

    it('opens no dialog, tells the host one waits, and asks once shown', async () => {
      stubStartup([])
      vi.mocked(HostService.Detect).mockResolvedValue(github as never)
      const onNeedsAttention = vi.fn()
      const host = (active: boolean, repoPath: string | null) => (
        <>
          <button type="button">The host's own view</button>
          <CurrentClientApp active={active} activeRepoPath={repoPath} onNeedsAttention={onNeedsAttention} />
        </>
      )
      const { rerender } = render(host(true, null))
      await screen.findByText('Current Client', { selector: 'h1' })
      // Hidden again, the host opens a repository on github.com.
      rerender(host(false, '/repos/app'))

      await waitFor(() => expect(onNeedsAttention).toHaveBeenCalled())
      expect(screen.queryByRole('alertdialog', { hidden: true })).toBeNull()
      expect(document.body.style.pointerEvents).not.toBe('none')
      expect(screen.getByRole('button', { name: "The host's own view" }).closest('[aria-hidden]')).toBeNull()

      rerender(host(true, '/repos/app'))
      expect(await screen.findByRole('alertdialog', { name: 'Turn on GitHub integration?' })).toBeInTheDocument()
    })

    it('asks nothing about a host when the host app asks in its own words', async () => {
      stubStartup(['/repos/app'], '/repos/app')
      vi.mocked(HostService.Detect).mockResolvedValue(github as never)
      render(<CurrentClientApp hostPrompt={false} />)
      await waitFor(() => expect(RepositoryService.OpenRepository).toHaveBeenCalledWith('/repos/app'))
      await waitFor(() => expect(document.querySelector('.header-bar')).not.toBeNull())
      await waitFor(() => expect(HostService.Detect).toHaveBeenCalled())
      await new Promise((resolve) => setTimeout(resolve, 50))
      expect(screen.queryByRole('alertdialog')).toBeNull()
    })
  })

  describe('hidden from the start', () => {
    afterEach(() => vi.useRealTimers())

    it('reads nothing for the screen, but restores, reports and fetches, then loads when first shown', async () => {
      vi.useFakeTimers({ shouldAdvanceTime: true })
      stubStartup(['/repos/app', '/repos/site'], '/repos/app')
      vi.mocked(SettingsService.GetSettings).mockResolvedValue({
        theme: 'dark',
        autoFetchIntervalMinutes: 5,
      } as Settings)
      vi.mocked(RemoteService.FetchAll).mockResolvedValue()
      vi.mocked(BranchService.ListLocal).mockResolvedValue([])
      const onActiveRepoChange = vi.fn()
      const app = (active: boolean) => <CurrentClientApp active={active} onActiveRepoChange={onActiveRepoChange} />
      const { rerender } = render(app(false))

      await waitFor(() => expect(onActiveRepoChange).toHaveBeenLastCalledWith('/repos/app'))
      await act(() => vi.advanceTimersByTimeAsync(5 * 60 * 1000))
      expect(RemoteService.FetchAll).toHaveBeenCalledWith('/repos/app', null)
      for (const read of [
        StatusService.GetStatus,
        BranchService.ListLocal,
        BranchService.CurrentBranchStatus,
        WorktreeService.List,
        RepositoryService.GetTabSummaries,
        RemoteService.LastFetchTime,
        RepositoryService.GetRepoSummaries,
      ]) {
        expect(read).not.toHaveBeenCalled()
      }

      rerender(app(true))
      await waitFor(() => expect(BranchService.ListLocal).toHaveBeenCalledWith('/repos/app'))
      await waitFor(() => expect(RepositoryService.GetTabSummaries).toHaveBeenCalledWith(['/repos/site']))
      expect(StatusService.GetStatus).toHaveBeenCalledWith('/repos/app')

      const reads = vi.mocked(BranchService.ListLocal).mock.calls.length
      rerender(app(false))
      rerender(app(true))
      await act(() => vi.advanceTimersByTimeAsync(100))
      expect(vi.mocked(BranchService.ListLocal).mock.calls.length).toBe(reads)
    })
  })
})
