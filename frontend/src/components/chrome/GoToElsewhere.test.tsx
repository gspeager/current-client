import { useState } from 'react'
import type { Mock } from 'vitest'
import type { CancellablePromise } from '@wailsio/runtime'
import { act, fireEvent, render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { BranchService, HistoryService, RepositoryMatches, SearchService } from '@current-client-bindings/app'
import type { useRepositoryLifecycle } from '../../features/repositories/useRepositoryLifecycle'
import { DialogProvider } from './DialogProvider'
import HeaderBar from './HeaderBar'
import { NEXT_AFTER_MS } from './useSearchElsewhere'

const APP = '/repos/app'
const SITE = '/repos/site'
const DOCS = '/repos/docs'

function cancellable<T>(promise: Promise<T>) {
  return Object.assign(promise, { cancel: vi.fn() }) as never as CancellablePromise<T> & { cancel: Mock }
}

const matches = (found: Partial<RepositoryMatches>) =>
  new RepositoryMatches({ files: [], branches: [], commits: [], notes: [], ...found })

function Shell({ onOpenCommit }: { onOpenCommit: (sha: string, position: number) => void }) {
  const [repoPath, setRepoPath] = useState(APP)
  const repo = {
    tabs: [APP, SITE, DOCS],
    tabSummaries: {},
    recentRepos: [],
    openRecent: (path: string) => Promise.resolve(setRepoPath(path)),
    closeTab: vi.fn(),
    reorderTabs: vi.fn(),
  } as unknown as ReturnType<typeof useRepositoryLifecycle>
  return (
    <DialogProvider>
      <span data-testid="open">{repoPath}</span>
      <HeaderBar
        repoPath={repoPath}
        repoVersion={0}
        lastFetchedAt={null}
        pruneOnFetch={false}
        repo={repo}
        dirty={false}
        workspaceActions={[]}
        onOpenSettings={vi.fn()}
        onOpenCommit={onOpenCommit}
        syncing={false}
        onPull={vi.fn()}
        onPullMerge={vi.fn()}
        onPullRebase={vi.fn()}
        onPush={vi.fn()}
        onForcePush={vi.fn()}
        onPushTags={vi.fn()}
        onPushSetUpstream={vi.fn()}
      />
    </DialogProvider>
  )
}

beforeEach(() => {
  vi.stubGlobal(
    'ResizeObserver',
    class {
      observe() {}
      unobserve() {}
      disconnect() {}
    },
  )
  Element.prototype.scrollIntoView = vi.fn()
  vi.mocked(BranchService.CurrentBranchStatus).mockResolvedValue({ current: 'main', ahead: 0, behind: 0 } as never)
})
afterEach(() => {
  vi.unstubAllGlobals()
  vi.useRealTimers()
})

async function searchOpenRepositories(query: string) {
  fireEvent.keyDown(window, { key: 'k', ctrlKey: true })
  const palette = within(await screen.findByRole('dialog', { name: 'Go to anything' }))
  await userEvent.click(palette.getByRole('button', { name: 'Open repositories (3)' }))
  await userEvent.type(palette.getByRole('combobox'), query)
  return palette
}

it('names each result by its repository, and opens it there', async () => {
  vi.mocked(SearchService.InRepository).mockImplementation((path: string) =>
    cancellable(
      Promise.resolve(
        path === SITE
          ? matches({ commits: [{ sha: 'abc1234def', subject: 'Fix the login redirect', date: '' } as never] })
          : matches({}),
      ),
    ),
  )
  vi.mocked(HistoryService.GetCommitPosition).mockResolvedValue(7)
  const onOpenCommit = vi.fn()
  render(<Shell onOpenCommit={onOpenCommit} />)
  const palette = await searchOpenRepositories('login')

  expect(await palette.findByText('In site')).toBeInTheDocument()
  expect(SearchService.InRepository).not.toHaveBeenCalledWith(APP, expect.anything())
  await userEvent.click(palette.getByRole('option', { name: 'Fix the login redirect, commit in site' }))

  expect(screen.getByTestId('open')).toHaveTextContent(SITE)
  await vi.waitFor(() => expect(onOpenCommit).toHaveBeenCalledWith('abc1234def', 7))
  expect(HistoryService.GetCommitPosition).toHaveBeenCalledWith(SITE, 'abc1234def')
})

it('cancels a search when the query changes, and moves on past a slow repository', async () => {
  const searches: { path: string; query: string; cancel: Mock }[] = []
  vi.mocked(SearchService.InRepository).mockImplementation((path: string, query: string) => {
    const search = cancellable(new Promise<RepositoryMatches>(() => {}))
    searches.push({ path, query, cancel: search.cancel })
    return search
  })
  vi.useFakeTimers({ shouldAdvanceTime: true })
  render(<Shell onOpenCommit={vi.fn()} />)
  const palette = await searchOpenRepositories('log')
  await vi.waitFor(() => expect(searches).toHaveLength(1))
  expect(palette.getByRole('status')).toHaveTextContent('Searching 2 more repositories…')

  await act(() => vi.advanceTimersByTimeAsync(NEXT_AFTER_MS))
  expect(searches.map((s) => s.path)).toEqual([SITE, DOCS])

  fireEvent.change(palette.getByRole('combobox'), { target: { value: 'login' } })
  await vi.waitFor(() => expect(searches.some((s) => s.query === 'login')).toBe(true))
  expect(searches.filter((s) => s.query === 'log').every((s) => s.cancel.mock.calls.length === 1)).toBe(true)
})
