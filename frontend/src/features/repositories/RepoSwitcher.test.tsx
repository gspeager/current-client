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
      tabs: [],
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
        tabs: [],
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

describe('RepoSwitcher clone', () => {
  it('shows the folder the clone will create inside the chosen destination', async () => {
    vi.mocked(RepositoryService.GetRepoSummaries).mockResolvedValue([])
    const repo = {
      recentRepos: [],
      tabs: [],
      cloneUrl: 'https://github.com/gspeager/current-client.git',
      cloneDest: '/Users/dev/projects',
      cloneFolder: 'current-client',
      setCloneFolder: vi.fn(),
      cloneRepository: vi.fn(),
      cloneOp: { running: false, error: null },
    } as unknown as ReturnType<typeof useRepositoryLifecycle>
    render(<RepoSwitcher repo={repo} />)

    expect(screen.queryByRole('textbox', { name: 'folder name' })).not.toBeInTheDocument()
    await userEvent.click(screen.getByRole('button', { name: 'Clone' }))
    expect(screen.getByRole('textbox', { name: 'folder name' })).toHaveValue('current-client')
    expect(screen.getByText('Clones into').parentElement).toHaveTextContent('/Users/dev/projects/current-client')
    expect(screen.getByRole('button', { name: 'Clone repository' })).toBeEnabled()
  })

  it('cannot clone without a folder name', async () => {
    vi.mocked(RepositoryService.GetRepoSummaries).mockResolvedValue([])
    const repo = {
      recentRepos: [],
      tabs: [],
      cloneUrl: 'https://example.com/',
      cloneDest: '/Users/dev/projects',
      cloneFolder: '',
      setCloneFolder: vi.fn(),
      cloneOp: { running: false, error: null },
    } as unknown as ReturnType<typeof useRepositoryLifecycle>
    render(<RepoSwitcher repo={repo} />)
    await userEvent.click(screen.getByRole('button', { name: 'Clone' }))

    expect(screen.queryByText('Clones into')).not.toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Clone repository' })).toBeDisabled()
  })
})

describe('RepoSwitcher Open tab', () => {
  const repo = () =>
    ({
      recentRepos: ['/repos/app', '/repos/site'],
      tabs: ['/repos/app'],
      openRecent: vi.fn(),
      removeRecent: vi.fn(),
      openRepository: vi.fn(),
      initRepository: vi.fn(),
      cloneOp: { running: false, error: null },
    }) as unknown as ReturnType<typeof useRepositoryLifecycle>

  it('filters the recent cards, badges the open ones, and browses from a button', async () => {
    vi.mocked(RepositoryService.GetRepoSummaries).mockResolvedValue([
      summary('/repos/app', 'app', true),
      summary('/repos/site', 'site', true),
    ])
    const r = repo()
    render(<RepoSwitcher repo={r} />)

    const app = (await screen.findByText('app')).closest('.repo-card')!
    expect(app).toHaveTextContent('open')
    expect(screen.getByText('site').closest('.repo-card')).not.toHaveTextContent('open')

    await userEvent.type(screen.getByRole('textbox', { name: 'Filter recent repositories' }), 'sit')
    expect(screen.queryByText('app')).not.toBeInTheDocument()
    expect(screen.getByText('site')).toBeInTheDocument()

    await userEvent.click(screen.getByRole('button', { name: 'Browse…' }))
    expect(r.openRepository).toHaveBeenCalled()
  })

  it('shows the default location in the footer with a way to Settings, and New initializes a folder', async () => {
    vi.mocked(RepositoryService.GetRepoSummaries).mockResolvedValue([])
    const r = repo()
    const onOpenSettings = vi.fn()
    render(<RepoSwitcher repo={r} defaultLocation="/Users/dev/projects" onOpenSettings={onOpenSettings} />)

    expect(screen.getByText('Dialogs start in').parentElement).toHaveTextContent('/Users/dev/projects')
    await userEvent.click(screen.getByRole('button', { name: 'Change in Settings' }))
    expect(onOpenSettings).toHaveBeenCalled()

    await userEvent.click(screen.getByRole('button', { name: 'New' }))
    await userEvent.click(screen.getByRole('button', { name: 'Choose a folder and initialize…' }))
    expect(r.initRepository).toHaveBeenCalled()
  })
})
