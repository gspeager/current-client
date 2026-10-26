import { cleanup, render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { BranchService, RemoteService, RepositoryService } from '@current-client-bindings/app'
import type { useRepositoryLifecycle } from '../../features/repositories/useRepositoryLifecycle'
import { DialogProvider } from './DialogProvider'
import HeaderBar from './HeaderBar'

const REPO = '/repos/app'
const repo = {
  tabs: [REPO],
  tabSummaries: {},
  recentRepos: [],
  openRecent: vi.fn(),
  closeTab: vi.fn(),
  reorderTabs: vi.fn(),
} as unknown as ReturnType<typeof useRepositoryLifecycle>

function renderHeader(overrides: Partial<Parameters<typeof HeaderBar>[0]> = {}) {
  const handlers = {
    onPull: vi.fn(),
    onPullMerge: vi.fn(),
    onPullRebase: vi.fn(),
    onPush: vi.fn(),
    onForcePush: vi.fn(),
    onPushTags: vi.fn(),
    onPushSetUpstream: vi.fn(),
  }
  render(
    <DialogProvider>
      <HeaderBar
        repoPath={REPO}
        repoVersion={0}
        lastFetchedAt={new Date(Date.now() - 26 * 60 * 1000)}
        pruneOnFetch={false}
        repo={repo}
        dirty={false}
        workspaceActions={[]}
        onOpenSettings={vi.fn()}
        onOpenCommit={vi.fn()}
        syncing={false}
        {...handlers}
        {...overrides}
      />
    </DialogProvider>,
  )
  return handlers
}

const menu = () => vi.waitFor(() => document.querySelector<HTMLElement>('.context-menu')!)

beforeEach(() => {
  vi.mocked(BranchService.CurrentBranchStatus).mockResolvedValue({ current: 'main', ahead: 2, behind: 1 } as never)
})

describe('HeaderBar sync cluster', () => {
  it('shows Fetch, Pull and Push with their counts, each with a menu, and no Prune checkbox', async () => {
    renderHeader()

    const cluster = document.querySelector('.header-bar-sync-cluster')!
    await vi.waitFor(() => expect(cluster).toHaveTextContent('FetchPull 1Push 2'))
    expect(screen.queryByRole('checkbox', { name: 'Prune' })).not.toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Fetch' })).toHaveAttribute('title', 'Fetched 26m ago. Fetch')
    expect(screen.queryByText(/fetched 26m ago/)).not.toBeInTheDocument()
  })

  it('colors Pull only while behind and Push only while ahead, on both segments', async () => {
    renderHeader()
    await screen.findByRole('button', { name: 'Pull 1' })
    for (const name of ['Pull 1', 'Pull options', 'Push 2', 'Push options']) {
      expect(screen.getByRole('button', { name })).toHaveClass('header-bar-sync-due')
    }
    expect(screen.getByRole('button', { name: 'Fetch' })).not.toHaveClass('header-bar-sync-due')
    cleanup()

    vi.mocked(BranchService.CurrentBranchStatus).mockResolvedValue({ current: 'main', ahead: 0, behind: 1 } as never)
    renderHeader()
    await screen.findByRole('button', { name: 'Pull 1' })
    expect(screen.getByRole('button', { name: 'Pull options' })).toHaveClass('header-bar-sync-due')
    expect(screen.getByRole('button', { name: 'Push' })).not.toHaveClass('header-bar-sync-due')
    expect(screen.getByRole('button', { name: 'Push options' })).not.toHaveClass('header-bar-sync-due')
  })

  it('opens a chevron’s menu with its right edge at the chevron’s right edge', async () => {
    const rect = { left: 500, right: 540, top: 10, bottom: 30, width: 40, height: 20 }
    vi.spyOn(Element.prototype, 'getBoundingClientRect').mockReturnValue({ ...rect, x: 500, y: 10, toJSON: () => rect })
    renderHeader()
    await userEvent.click(await screen.findByRole('button', { name: 'Push options' }))
    const items = await menu()
    expect(items.style.left).toBe(`${540 - 40}px`)
  })

  it('fetches origin from the button, and prunes or fetches every remote from the menu', async () => {
    vi.mocked(RemoteService.List).mockResolvedValue([{ name: 'origin' }] as never)
    vi.mocked(RemoteService.Fetch).mockResolvedValue()
    vi.mocked(RemoteService.FetchPrune).mockResolvedValue()
    vi.mocked(RemoteService.FetchAll).mockResolvedValue()
    renderHeader()

    await userEvent.click(screen.getByRole('button', { name: 'Fetch' }))
    await vi.waitFor(() => expect(RemoteService.Fetch).toHaveBeenCalledWith(REPO, 'origin', null))

    await userEvent.click(screen.getByRole('button', { name: 'Fetch options' }))
    expect([...(await menu()).querySelectorAll('button')].map((b) => b.textContent)).toEqual([
      'Fetch and prune',
      'Fetch all remotes',
    ])
    await userEvent.click(screen.getByRole('button', { name: 'Fetch and prune' }))
    await vi.waitFor(() => expect(RemoteService.FetchPrune).toHaveBeenCalledWith(REPO, 'origin', null))

    await userEvent.click(screen.getByRole('button', { name: 'Fetch options' }))
    await userEvent.click(await screen.findByRole('button', { name: 'Fetch all remotes' }))
    await vi.waitFor(() => expect(RemoteService.FetchAll).toHaveBeenCalledWith(REPO, null))
  })

  it('offers tags, upstream and force-push from the Push menu', async () => {
    const handlers = renderHeader()

    await userEvent.click(screen.getByRole('button', { name: 'Push options' }))
    const items = await menu()
    expect([...items.querySelectorAll('button')].map((b) => b.textContent)).toEqual([
      'Push tags',
      'Set upstream…',
      'Force-push (with lease)…',
    ])
    expect(within(items).getByRole('button', { name: 'Force-push (with lease)…' })).toHaveClass(
      'context-menu-item-destructive',
    )
    await userEvent.click(within(items).getByRole('button', { name: 'Push tags' }))
    expect(handlers.onPushTags).toHaveBeenCalled()

    await userEvent.click(screen.getByRole('button', { name: 'Push options' }))
    await userEvent.click(await screen.findByRole('button', { name: 'Set upstream…' }))
    expect(handlers.onPushSetUpstream).toHaveBeenCalled()
  })

  it('pulls, rebasing on shift-click, and merges or rebases from the menu', async () => {
    const handlers = renderHeader()
    const user = userEvent.setup()

    await user.click(screen.getByRole('button', { name: /^Pull( [0-9]+)?$/ }))
    expect(handlers.onPull).toHaveBeenCalled()
    await user.keyboard('{Shift>}')
    await user.click(screen.getByRole('button', { name: /^Pull( [0-9]+)?$/ }))
    await user.keyboard('{/Shift}')
    expect(handlers.onPullRebase).toHaveBeenCalledTimes(1)

    await user.click(screen.getByRole('button', { name: 'Pull options' }))
    await user.click(await screen.findByRole('button', { name: 'Pull (merge)' }))
    expect(handlers.onPullMerge).toHaveBeenCalled()
  })
})

describe('HeaderBar during a merge', () => {
  it('badges the repository chip with the operation', async () => {
    renderHeader({ operation: 'merge' })

    expect(await screen.findByText('merging')).toHaveClass('header-bar-mode')
  })
})

describe('HeaderBar sync cluster during a merge', () => {
  it('disables Fetch, Pull and Push with the reason', async () => {
    renderHeader({ operation: 'merge' })

    const fetch = screen.getByRole('button', { name: 'Fetch' })
    expect(fetch).toBeDisabled()
    expect(fetch).toHaveAttribute('title', 'Finish the merge first')
    expect(screen.getByRole('button', { name: 'Push options' })).toBeDisabled()
  })
})

describe('HeaderBar repository pane', () => {
  it('switches between recent repositories and opens the dialog for anything else', async () => {
    vi.mocked(RepositoryService.GetRepoSummaries).mockResolvedValue([])
    renderHeader()

    await userEvent.click(screen.getByText('app', { selector: '.header-bar-repo-name' }))
    await userEvent.click(await screen.findByRole('button', { name: 'Open another…' }))

    expect(await screen.findByRole('dialog', { name: 'Open repository' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Clone' })).toBeInTheDocument()
  })
})
