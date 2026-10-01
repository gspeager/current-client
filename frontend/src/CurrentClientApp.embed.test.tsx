import { createRef } from 'react'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import {
  GitService,
  HistoryService,
  RepositoryService,
  SettingsService,
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
  vi.mocked(RepositoryService.GetRepoSummaries).mockResolvedValue([])
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

  it('does not reload a tab that is visited again with no new commits', async () => {
    stubStartup(['/repos/app'], '/repos/app')
    vi.mocked(HistoryService.GetHistory).mockResolvedValue([])
    render(<CurrentClientApp />)
    const historyTab = await screen.findByRole('button', { name: 'Branch Graph & History' })

    await userEvent.click(historyTab)
    await waitFor(() => expect(HistoryService.GetHistory).toHaveBeenCalled())
    await new Promise((resolve) => setTimeout(resolve, 50))
    const loads = vi.mocked(HistoryService.GetHistory).mock.calls.length

    await userEvent.click(screen.getByRole('button', { name: 'Activity' }))
    await userEvent.click(historyTab)
    await new Promise((resolve) => setTimeout(resolve, 50))

    expect(HistoryService.GetHistory).toHaveBeenCalledTimes(loads)
  })
})
