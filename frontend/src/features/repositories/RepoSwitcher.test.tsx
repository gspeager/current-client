import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { RepositoryService, type RepoSummaryInfo } from '@current-client-bindings/app'
import type { useRepositoryLifecycle } from './useRepositoryLifecycle'
import RepoSwitcher from './RepoSwitcher'

function summary(path: string, name: string, available: boolean): RepoSummaryInfo {
  return { path, name, available, currentBranch: 'main', ahead: 0, behind: 0, dirty: 0, activity: [] }
}

describe('RepoSwitcher recent repositories', () => {
  it('removes a missing repository from the recent list without opening it', async () => {
    vi.mocked(RepositoryService.GetRepoSummaries).mockResolvedValue([
      summary('/repos/app', 'app', true),
      summary('/tmp/gone', 'gone', false),
    ])
    const repo = {
      recentRepos: ['/repos/app', '/tmp/gone'],
      openRecent: vi.fn(),
      removeRecent: vi.fn(),
      cloneOp: { running: false, error: null },
    } as unknown as ReturnType<typeof useRepositoryLifecycle>
    render(<RepoSwitcher repo={repo} />)

    await userEvent.click(await screen.findByRole('button', { name: 'Remove gone from recent' }))

    expect(repo.removeRecent).toHaveBeenCalledWith('/tmp/gone')
    expect(repo.openRecent).not.toHaveBeenCalled()
  })
})

describe('RepoSwitcher after a removal', () => {
  it('drops just that card and keeps the rest on screen while the list refreshes', async () => {
    vi.mocked(RepositoryService.GetRepoSummaries)
      .mockResolvedValueOnce([summary('/repos/app', 'app', true), summary('/tmp/gone', 'gone', false)])
      .mockReturnValueOnce(new Promise(() => {}) as ReturnType<typeof RepositoryService.GetRepoSummaries>)
    const repoWith = (recentRepos: string[]) =>
      ({
        recentRepos,
        openRecent: vi.fn(),
        removeRecent: vi.fn(),
        cloneOp: { running: false, error: null },
      }) as unknown as ReturnType<typeof useRepositoryLifecycle>
    const { rerender } = render(<RepoSwitcher repo={repoWith(['/repos/app', '/tmp/gone'])} />)
    await screen.findByText('gone')

    rerender(<RepoSwitcher repo={repoWith(['/repos/app'])} />)

    expect(screen.queryByText('gone')).not.toBeInTheDocument()
    expect(screen.getByText('app')).toBeInTheDocument()
    expect(screen.queryByText('Loading…')).not.toBeInTheDocument()
  })
})
