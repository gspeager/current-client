import { render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import {
  type BranchInfo,
  BranchService,
  ChangedFile,
  type CommitInfo,
  DiffService,
  FileDiff,
  GraphNode,
  HistoryService,
  IdentityInfo,
  IdentityService,
  HostService,
  PlatformService,
  RemoteService,
  ReviewThread,
  ReviewThreads,
} from '@current-client-bindings/app'
import { Links } from '@current-client-bindings/core/hostlink'
import { DialogProvider } from '../../components/chrome/DialogProvider'
import { giveElementsLayout } from '../../test/bindings'
import HistoryView from './HistoryView'

const REPO = '/repos/app'

function commit(sha: string, subject: string, parentShas: string[]): CommitInfo {
  return {
    sha,
    parentShas,
    authorName: 'Ada Lovelace',
    authorEmail: 'ada@example.com',
    date: '2026-01-10T12:00:00Z',
    subject,
    body: '',
    signature: 'none',
    coAuthors: [],
  }
}

const history = [
  commit('b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0c1', 'Fix login redirect', [
    'a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0',
  ]),
  commit('a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0', 'Initial commit', []),
]

function renderHistory(props: Partial<Parameters<typeof HistoryView>[0]> = {}) {
  render(
    <DialogProvider>
      <HistoryView repoPath={REPO} {...props} />
    </DialogProvider>,
  )
}

beforeEach(() => giveElementsLayout())

describe('HistoryView', () => {
  it('opens the inspector for a clicked commit and reports the selection', async () => {
    vi.mocked(HistoryService.GetHistory).mockResolvedValue(history)
    const onSelectedShaChange = vi.fn()
    renderHistory({ onSelectedShaChange })

    await userEvent.click(await screen.findByRole('button', { name: /Initial commit/ }))

    expect(onSelectedShaChange).toHaveBeenCalledWith(history[1].sha)
    expect(HistoryService.GetChangedFiles).toHaveBeenCalledWith(REPO, history[1].sha)
    expect(screen.getByRole('button', { name: 'Copy full commit hash' }).parentElement).toHaveTextContent('a1b2c3d')
  })

  it('links issue references in the row and the inspector to the host', async () => {
    vi.mocked(HistoryService.GetHistory).mockResolvedValue([{ ...history[0], subject: 'Fix login (#12)' }, history[1]])
    vi.mocked(RemoteService.HostLinks).mockResolvedValue(
      new Links({ issue: 'https://github.com/o/r/issues/', root: 'https://github.com/', issuePath: '/issues/' }),
    )
    vi.mocked(PlatformService.OpenWebPage).mockResolvedValue()
    const onSelectedShaChange = vi.fn()
    renderHistory({ onSelectedShaChange })

    const rowLink = await screen.findByRole('link', { name: '#12' })
    expect(rowLink).toHaveAttribute('href', 'https://github.com/o/r/issues/12')
    await userEvent.click(rowLink)
    expect(PlatformService.OpenWebPage).toHaveBeenCalledWith('https://github.com/o/r/issues/12')
    expect(onSelectedShaChange).not.toHaveBeenCalled()

    await userEvent.click(screen.getByRole('button', { name: /Fix login/ }))
    expect(screen.getAllByRole('link', { name: '#12' })).toHaveLength(2)
  })

  it('lists co-authors in the inspector', async () => {
    const paired = {
      ...history[1],
      coAuthors: [
        { name: 'Grace Hopper', email: 'grace@example.com' },
        { name: 'Alan Turing', email: '' },
      ],
    }
    vi.mocked(HistoryService.GetHistory).mockResolvedValue([history[0], paired])
    renderHistory({ initialSelectedSha: paired.sha })

    const label = await screen.findByText('Co-authors')
    expect(label.parentElement).toHaveTextContent('Grace Hopper, Alan Turing')
    expect(screen.getByText('Grace Hopper')).toHaveAttribute('title', 'grace@example.com')
  })

  it('shows no co-authors row for a commit without them', async () => {
    vi.mocked(HistoryService.GetHistory).mockResolvedValue(history)
    renderHistory({ initialSelectedSha: history[0].sha })

    await screen.findByRole('button', { name: 'Copy full commit hash' })
    expect(screen.queryByText('Co-authors')).not.toBeInTheDocument()
  })

  it('starts with the requested commit selected', async () => {
    vi.mocked(HistoryService.GetHistory).mockResolvedValue(history)
    renderHistory({ initialSelectedSha: history[0].sha })

    await screen.findByRole('button', { name: /Fix login redirect/ })
    expect(HistoryService.GetChangedFiles).toHaveBeenCalledWith(REPO, history[0].sha)
  })

  it('opens a commit menu with Shift+F10 on the focused row', async () => {
    vi.mocked(HistoryService.GetHistory).mockResolvedValue(history)
    renderHistory()
    const user = userEvent.setup()

    ;(await screen.findByRole('button', { name: /Initial commit/ })).focus()
    await user.keyboard('{Shift>}{F10}{/Shift}')

    const menu = await vi.waitFor(() => document.querySelector<HTMLElement>('.context-menu')!)
    expect(within(menu).getByRole('button', { name: 'Check out' })).toHaveFocus()
    expect([...menu.querySelectorAll('button')].map((b) => b.textContent)).toEqual([
      'Check out',
      'Branch here',
      'Cherry-pick',
      'Revert',
      'Reset to here…',
      'Export patch…',
      'Copy SHA',
      'Highlight author’s commits',
    ])
  })

  it('follows only first parents when the toggle is on, saving the choice and drawing one lane', async () => {
    const merge = commit('c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0c1d2', 'Merge feature', [history[0].sha, 'side'])
    vi.mocked(HistoryService.GetHistory).mockResolvedValue([merge, ...history])
    vi.mocked(HistoryService.ComputeGraphLayout).mockResolvedValue([])
    const onFirstParentChange = vi.fn()
    renderHistory({ onFirstParentChange })

    await screen.findByRole('button', { name: /Merge feature/ })
    expect(vi.mocked(HistoryService.GetHistory).mock.lastCall?.[3]).toMatchObject({ firstParent: false })
    expect(vi.mocked(HistoryService.ComputeGraphLayout).mock.lastCall?.[0][0].parentShas).toHaveLength(2)

    await userEvent.click(screen.getByRole('checkbox', { name: 'First parent only' }))
    expect(onFirstParentChange).toHaveBeenCalledWith(true)
    await vi.waitFor(() =>
      expect(vi.mocked(HistoryService.GetHistory).mock.lastCall?.[3]).toMatchObject({ firstParent: true }),
    )
    await vi.waitFor(() =>
      expect(vi.mocked(HistoryService.ComputeGraphLayout).mock.lastCall?.[0][0].parentShas).toEqual([history[0].sha]),
    )
  })

  it('highlights one author’s commits, dimming the rest with the graph unchanged, and clears again', async () => {
    const byGrace = {
      ...commit('c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0c1d2', 'Add tests', [history[0].sha]),
      authorName: 'Grace Hopper',
      authorEmail: 'grace@example.com',
    }
    const commits = [byGrace, ...history]
    vi.mocked(HistoryService.GetHistory).mockResolvedValue(commits)
    vi.mocked(HistoryService.ComputeGraphLayout).mockResolvedValue(
      commits.map(
        (c) => new GraphNode({ sha: c.sha, lane: 0, parentLanes: [0], convergingLanes: [], passThrough: [] }),
      ),
    )
    vi.mocked(IdentityService.GetIdentities).mockResolvedValue(
      ['G', 'A'].map((initials) => new IdentityInfo({ initials, color: 'red' })),
    )
    renderHistory()
    const user = userEvent.setup()

    const rows = {
      grace: await screen.findByRole('button', { name: /Add tests/ }),
      ada: await screen.findByRole('button', { name: /Fix login redirect/ }),
    }
    const graph = () => [...document.querySelectorAll('.commit-row-graph svg')].map((s) => s.outerHTML)
    const before = graph()
    expect(before).toHaveLength(3)

    await user.click(await within(rows.ada).findByRole('button', { name: 'Highlight commits by Ada Lovelace' }))
    expect(rows.grace).toHaveStyle({ opacity: '0.4' })
    expect(rows.ada).toHaveStyle({ opacity: '1' })
    expect(graph()).toEqual(before)
    expect(HistoryService.GetHistory).toHaveBeenCalledTimes(1)

    await user.click(within(rows.ada).getByRole('button', { name: 'Clear the highlight of Ada Lovelace' }))
    expect(rows.grace).toHaveStyle({ opacity: '1' })

    await user.click(within(rows.grace).getByRole('button', { name: 'Highlight commits by Grace Hopper' }))
    expect(rows.ada).toHaveStyle({ opacity: '0.4' })
    await user.keyboard('{Escape}')
    expect(rows.ada).toHaveStyle({ opacity: '1' })
  })

  it("shows a file's diff in place of the graph, with a way back", async () => {
    vi.mocked(HistoryService.GetHistory).mockResolvedValue(history)
    vi.mocked(HistoryService.GetChangedFiles).mockResolvedValue([new ChangedFile({ status: 'M', path: 'src/app.ts' })])
    vi.mocked(DiffService.GetRefDiff).mockResolvedValue(new FileDiff({ binary: true }))
    renderHistory()

    await userEvent.click(await screen.findByRole('button', { name: /Fix login redirect/ }))
    await userEvent.click(await screen.findByRole('button', { name: /src\/app\.ts/ }))

    expect(await screen.findByText('Binary file changed.')).toBeInTheDocument()
    expect(DiffService.GetRefDiff).toHaveBeenCalledWith(
      REPO,
      'src/app.ts',
      'a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0',
      'b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0c1',
      false,
      false,
    )
    expect(screen.queryByPlaceholderText('Search commits, authors, hashes…')).not.toBeInTheDocument()
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
    expect(document.querySelector('.history-inspector')).not.toBeNull()

    await userEvent.click(screen.getByRole('button', { name: 'History' }))
    expect(screen.getByPlaceholderText('Search commits, authors, hashes…')).toBeInTheDocument()
  })

  it("counts the branch's review threads on the commit's files, as they are at that commit", async () => {
    vi.mocked(HistoryService.GetHistory).mockResolvedValue(history)
    vi.mocked(HistoryService.GetChangedFiles).mockResolvedValue([new ChangedFile({ status: 'M', path: 'src/app.ts' })])
    vi.mocked(HostService.ReviewThreadsAt).mockResolvedValue(
      new ReviewThreads({
        enabled: true,
        number: 7,
        threads: [new ReviewThread({ id: 't1', path: 'src/app.ts', line: 3 })],
        unresolved: { 'src/app.ts': 1 },
        total: 1,
      }),
    )
    renderHistory()

    await userEvent.click(await screen.findByRole('button', { name: /Fix login redirect/ }))

    expect(await screen.findByTitle('1 unresolved review thread')).toBeInTheDocument()
    expect(HostService.ReviewThreadsAt).toHaveBeenCalledWith(REPO, 'b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0c1')
  })

  it('opens a fixed inspector whose actions match the menu, and says how many commits there are', async () => {
    vi.mocked(HistoryService.GetHistory).mockResolvedValue(history)
    renderHistory()

    await userEvent.click(await screen.findByRole('button', { name: /Initial commit/ }))

    const inspector = document.querySelector('.history-inspector')!
    expect(inspector).not.toHaveAttribute('style')
    expect(screen.queryByRole('separator', { name: 'Resize commit inspector' })).not.toBeInTheDocument()
    expect([...inspector.querySelectorAll('.commit-detail-actions > button')].map((b) => b.textContent)).toEqual([
      'Check out',
      'Branch here',
      'Cherry-pick',
      'Revert',
    ])
    expect(inspector.querySelector('.commit-detail-more-toggle')).toHaveTextContent('More')
    expect(await screen.findByText('2 commits · stashes hidden')).toBeInTheDocument()
  })

  it('offers Open on host for a commit when a remote points at a host', async () => {
    vi.mocked(HistoryService.GetHistory).mockResolvedValue(history)
    vi.mocked(RemoteService.HostLinks).mockResolvedValue(new Links({ commit: 'https://github.com/o/r/commit/' }))
    vi.mocked(PlatformService.OpenWebPage).mockResolvedValue()
    renderHistory()
    const user = userEvent.setup()

    ;(await screen.findByRole('button', { name: /Initial commit/ })).focus()
    await user.keyboard('{Shift>}{F10}{/Shift}')
    const menu = await vi.waitFor(() => document.querySelector<HTMLElement>('.context-menu')!)
    await user.click(within(menu).getByRole('button', { name: 'Open on host' }))

    expect(PlatformService.OpenWebPage).toHaveBeenCalledWith(
      expect.stringMatching(/^https:\/\/github.com\/o\/r\/commit\/[0-9a-f]+$/),
    )
  })

  it('shows why history could not load', async () => {
    vi.mocked(HistoryService.GetHistory).mockRejectedValue(new Error('Not a git repository.'))
    renderHistory()

    expect(await screen.findByText('Could not load history: Not a git repository.')).toBeInTheDocument()
  })
  describe('Conventional Commits', () => {
    const typed = [commit('c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0c1d2', 'feat(history): filter by type', [])]

    it('mutes the type prefix and filters by type', async () => {
      vi.mocked(HistoryService.GetHistory).mockResolvedValue(typed)
      renderHistory({ conventional: true })

      const prefix = await screen.findByText('feat(history):', { exact: false, selector: '.commit-row-type' })
      expect(prefix.parentElement).toHaveTextContent('feat(history): filter by type')

      await userEvent.click(screen.getByText('Filters'))
      await userEvent.selectOptions(screen.getByRole('combobox', { name: 'Type filter' }), 'Breaking changes')
      expect(screen.getByText('Filters').closest('summary')).toHaveAttribute('aria-label', 'Filters, 1 on')
      expect(HistoryService.GetHistory).toHaveBeenLastCalledWith(
        REPO,
        expect.any(Number),
        0,
        expect.objectContaining({ type: '!' }),
      )
    })

    it('shows plain subjects and no type filter without the format', async () => {
      vi.mocked(HistoryService.GetHistory).mockResolvedValue(typed)
      renderHistory()

      await screen.findByText('feat(history): filter by type')
      expect(document.querySelector('.commit-row-type')).toBeNull()
      await userEvent.click(screen.getByText('Filters'))
      expect(screen.queryByRole('combobox', { name: 'Type filter' })).toBeNull()
      expect(screen.getByRole('combobox', { name: 'Date filter' })).toBeInTheDocument()
    })
  })
})

describe('HistoryView branch filter', () => {
  const localBranches = (...names: string[]) =>
    vi.mocked(BranchService.ListLocal).mockResolvedValue(names.map((name) => ({ name }) as BranchInfo))
  const loadedRefs = () => vi.mocked(HistoryService.GetHistory).mock.calls.map(([, , , filter]) => filter.ref)

  it('shows all branches by default, and reports a change to the filter', async () => {
    vi.mocked(HistoryService.GetHistory).mockResolvedValue(history)
    localBranches('main', 'feature')
    const onRefChange = vi.fn()
    renderHistory({ onRefChange })

    const select = await screen.findByRole('combobox', { name: 'Branch filter' })
    expect(select).toHaveValue('--all')
    await vi.waitFor(() => expect(loadedRefs()).toContain('--all'))

    await screen.findByRole('option', { name: 'feature' })
    await userEvent.selectOptions(select, 'feature')
    expect(onRefChange).toHaveBeenCalledWith('feature')
  })

  it('falls back to all branches when the saved branch no longer exists', async () => {
    vi.mocked(HistoryService.GetHistory).mockResolvedValue(history)
    localBranches('main')
    renderHistory({ defaultRef: 'deleted-branch' })

    await vi.waitFor(() => expect(screen.getByRole('combobox', { name: 'Branch filter' })).toHaveValue('--all'))
  })

  it('keeps a saved branch that still exists', async () => {
    vi.mocked(HistoryService.GetHistory).mockResolvedValue(history)
    localBranches('main', 'feature')
    renderHistory({ defaultRef: 'feature' })

    await screen.findByRole('option', { name: 'feature' })
    expect(screen.getByRole('combobox', { name: 'Branch filter' })).toHaveValue('feature')
  })
})
