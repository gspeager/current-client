import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import {
  BranchChange,
  BranchService,
  CIMark as CIMarkInfo,
  ConflictStateInfo,
  CurrentChange,
  HostService,
  MergedBranches,
  OverlapService,
  PlatformService,
  RemoteService,
  type BranchInfo,
  type BranchStatusInfo,
} from '@current-client-bindings/app'
import { CIStatus, Job } from '@current-client-bindings/core/hostapi'
import { Links, Style } from '@current-client-bindings/core/hostlink'
import { DialogProvider } from '../../components/chrome/DialogProvider'
import BranchSidebar from './BranchSidebar'
import { defaultSections } from '../../test/sections'

const REPO = '/repos/app'

function branch(name: string, current = false): BranchInfo {
  return { name, current, upstream: '', ahead: 0, behind: 0, lastCommitDate: '', worktreePath: '', tip: '' }
}

function renderSidebar(onBranchChanged = vi.fn(), sections = defaultSections()) {
  render(
    <DialogProvider>
      <BranchSidebar
        repoPath={REPO}
        dirty={false}
        refreshKey={0}
        sections={sections}
        onBranchChanged={onBranchChanged}
      />
    </DialogProvider>,
  )
  return onBranchChanged
}

// Sections that are all open, so remote rows render.
function openSections() {
  return { ...defaultSections(), isOpen: () => true }
}

describe('BranchSidebar groups', () => {
  beforeEach(() => {
    vi.mocked(BranchService.ListLocal).mockResolvedValue([
      branch('main', true),
      { ...branch('feature'), lastCommitDate: '2026-10-20T00:00:00Z' },
      branch('docs'),
    ])
    vi.mocked(BranchService.ListRemote).mockResolvedValue(['origin/feature', 'origin/login'])
    vi.mocked(RemoteService.List).mockResolvedValue([{ name: 'origin', fetchUrl: 'git@github.com:g/r.git' }] as never)
  })

  it('lists the checked-out branch first, then Local with a count, then Remote closed', async () => {
    renderSidebar()

    const checkedOut = (await screen.findByText('Checked out')).parentElement!
    expect(within(checkedOut).getByRole('button', { name: /^main/ })).toHaveTextContent('HEAD')
    expect(screen.getByRole('button', { name: 'Local, 2' })).toHaveAttribute('aria-expanded', 'true')
    expect(screen.getByRole('button', { name: 'Remote, 2' })).toHaveAttribute('aria-expanded', 'false')
    expect(screen.queryByText('origin/login')).not.toBeInTheDocument()
  })

  it('filters local and remote branches together', async () => {
    renderSidebar(vi.fn(), openSections())

    await screen.findByText('origin/login')
    await userEvent.type(screen.getByRole('textbox', { name: 'Filter branches' }), 'log')

    expect(screen.getByRole('button', { name: 'Local, 0' })).toBeInTheDocument()
    expect(screen.getByText('origin/login')).toBeInTheDocument()
    expect(screen.queryByText('origin/feature')).not.toBeInTheDocument()
    expect(screen.getByRole('button', { name: /^main/ })).toBeInTheDocument()
  })

  it('checks out a remote branch by its local name, and says when it has none yet', async () => {
    vi.mocked(BranchService.CheckoutBranch).mockResolvedValue()
    renderSidebar(vi.fn(), openSections())

    const login = (await screen.findByText('origin/login')).closest('.branch-row')!
    expect(login).toHaveTextContent('not local')
    expect((await screen.findByText('origin/feature')).closest('.branch-row')).not.toHaveTextContent('not local')
    await userEvent.click(login)

    expect(BranchService.CheckoutBranch).toHaveBeenCalledWith(REPO, 'login')
  })

  it('badges the checked-out branch as merging and the other side as incoming', async () => {
    render(
      <DialogProvider>
        <BranchSidebar
          repoPath={REPO}
          dirty={false}
          refreshKey={0}
          sections={defaultSections()}
          conflict={new ConflictStateInfo({ operation: 'merge', ours: 'main', theirs: 'feature' })}
        />
      </DialogProvider>,
    )

    expect(await screen.findByText('merging')).toHaveClass('branch-row-mode')
    expect(screen.getByRole('button', { name: /^feature/ })).toHaveTextContent('incoming')
    expect(screen.getByRole('button', { name: /^docs/ })).not.toHaveTextContent('incoming')
  })

  it('blocks checking out and new branches until the merge is finished', async () => {
    vi.mocked(BranchService.CheckoutBranch).mockResolvedValue()
    render(
      <DialogProvider>
        <BranchSidebar
          repoPath={REPO}
          dirty={false}
          refreshKey={0}
          sections={defaultSections()}
          conflict={new ConflictStateInfo({ operation: 'merge', ours: 'main', theirs: 'feature' })}
        />
      </DialogProvider>,
    )

    const docs = await screen.findByRole('button', { name: /^docs/ })
    expect(docs).toHaveAttribute('aria-disabled', 'true')
    expect(docs).toHaveAttribute('title', 'Finish the merge first')
    await userEvent.click(docs)
    expect(BranchService.CheckoutBranch).not.toHaveBeenCalled()
    expect(screen.getByRole('button', { name: 'New branch' })).toBeDisabled()
    await userEvent.click(screen.getByRole('button', { name: 'More actions for docs' }))
    expect(await screen.findByRole('button', { name: 'Check out' })).toBeDisabled()
  })

  it('badges a remote-tracking branch as incoming when that is what is merging', async () => {
    render(
      <DialogProvider>
        <BranchSidebar
          repoPath={REPO}
          dirty={false}
          refreshKey={0}
          sections={openSections()}
          conflict={new ConflictStateInfo({ operation: 'merge', ours: 'main', theirs: 'origin/login' })}
        />
      </DialogProvider>,
    )

    expect((await screen.findByText('origin/login')).closest('.branch-row')).toHaveTextContent('incoming')
  })

  it('opens Branches when New branch is used on a closed section', async () => {
    const sections = { ...defaultSections(), isOpen: () => false }
    renderSidebar(vi.fn(), sections)

    await userEvent.click(await screen.findByRole('button', { name: 'New branch' }))

    expect(sections.open).toHaveBeenCalledWith('branches')
    expect(screen.getByRole('textbox', { name: 'new branch name' })).toBeInTheDocument()
  })
})

describe('BranchSidebar', () => {
  it('offers to create a merge request for a pushed branch without one, through the host’s form', async () => {
    vi.mocked(BranchService.ListLocal).mockResolvedValue([
      branch('main', true),
      { ...branch('feature'), upstream: 'origin/feature' },
    ])
    vi.mocked(RemoteService.HostLinks).mockResolvedValue(
      new Links({
        root: 'https://gitlab.com/',
        issuePath: '/-/issues/',
        tree: 'https://gitlab.com/g/r/-/tree/',
        style: Style.GitLab,
      }),
    )
    vi.mocked(HostService.BranchChanges).mockResolvedValue({})
    vi.mocked(RemoteService.CreateChangeLink).mockResolvedValue('https://gitlab.com/g/r/-/merge_requests/new?x')
    vi.mocked(PlatformService.OpenWebPage).mockResolvedValue()
    renderSidebar()

    await userEvent.click(await screen.findByRole('button', { name: 'More actions for feature' }))
    await userEvent.click(await screen.findByRole('button', { name: 'Create merge request…' }))
    expect(RemoteService.CreateChangeLink).toHaveBeenCalledWith(REPO, 'feature')
    await waitFor(() =>
      expect(PlatformService.OpenWebPage).toHaveBeenCalledWith('https://gitlab.com/g/r/-/merge_requests/new?x'),
    )
  })

  it('opens a branch on the host, as its merge request when known and its tree otherwise', async () => {
    vi.mocked(BranchService.ListLocal).mockResolvedValue([
      branch('main', true),
      { ...branch('feature'), upstream: 'origin/feature' },
      { ...branch('other'), upstream: 'origin/other' },
    ])
    vi.mocked(RemoteService.HostLinks).mockResolvedValue(
      new Links({ root: 'https://gitlab.com/', issuePath: '/-/issues/', tree: 'https://gitlab.com/g/r/-/tree/' }),
    )
    vi.mocked(HostService.BranchChanges).mockResolvedValue({
      'origin/feature': new BranchChange({
        number: 7,
        prefix: '!',
        state: 'open',
        webUrl: 'https://gitlab.com/g/r/-/merge_requests/7',
      }),
    })
    vi.mocked(PlatformService.OpenWebPage).mockResolvedValue()
    renderSidebar()

    await userEvent.click(await screen.findByRole('button', { name: 'More actions for feature' }))
    await userEvent.click(await screen.findByRole('button', { name: 'Open on host' }))
    expect(PlatformService.OpenWebPage).toHaveBeenLastCalledWith('https://gitlab.com/g/r/-/merge_requests/7')

    await userEvent.click(screen.getByRole('button', { name: 'More actions for other' }))
    await userEvent.click(await screen.findByRole('button', { name: 'Open on host' }))
    expect(PlatformService.OpenWebPage).toHaveBeenLastCalledWith('https://gitlab.com/g/r/-/tree/other')
  })

  it('marks a branch whose upstream has a merge request, from the cache alone', async () => {
    vi.mocked(BranchService.ListLocal).mockResolvedValue([
      branch('main', true),
      { ...branch('feature'), upstream: 'origin/feature' },
      { ...branch('other'), upstream: 'origin/other' },
    ])
    vi.mocked(HostService.BranchChanges).mockResolvedValue({
      'origin/feature': new BranchChange({ number: 138, prefix: '!', state: 'open', title: 'Add the thing' }),
    })
    renderSidebar()

    const row = await screen.findByRole('button', { name: /^feature/ })
    expect(within(row).getByRole('button', { name: '!138 · Add the thing · open' })).toBeInTheDocument()
    expect(within(screen.getByRole('button', { name: /^other/ })).queryByText(/!/)).not.toBeInTheDocument()
    expect(HostService.BranchChanges).toHaveBeenCalledWith(REPO)
  })

  // The job list is portaled out of the row but React still bubbles its
  // clicks up the tree; the modal stops them before they reach the row.
  it('opens a branch’s CI jobs and closes them without checking the branch out', async () => {
    vi.mocked(BranchService.ListLocal).mockResolvedValue([branch('main', true), { ...branch('feature'), tip: 'abc' }])
    vi.mocked(HostService.CIStatus).mockResolvedValue({ abc: new CIMarkInfo({ status: 'failed', pipelineId: 12 }) })
    vi.mocked(HostService.PipelineJobs).mockResolvedValue([
      new Job({
        id: 5,
        name: 'test',
        stage: 'unit',
        status: CIStatus.CIFailed,
        webUrl: 'https://gitlab.example.com/j/5',
      }),
    ])
    vi.mocked(PlatformService.OpenWebPage).mockResolvedValue()
    vi.mocked(BranchService.CheckoutBranch).mockResolvedValue()
    renderSidebar()

    const row = await screen.findByRole('button', { name: /^feature/ })
    await userEvent.click(within(row).getByRole('button', { name: 'CI failed' }))
    const dialog = await screen.findByRole('dialog', { name: 'CI failed' })
    // A sidebar row has no commit details to open, so a failed job opens on the host.
    await userEvent.click(await within(dialog).findByRole('button', { name: /^test/ }))
    expect(PlatformService.OpenWebPage).toHaveBeenCalledWith('https://gitlab.example.com/j/5')
    await userEvent.click(within(dialog).getByRole('button', { name: 'Close' }))
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
    expect(BranchService.CheckoutBranch).not.toHaveBeenCalled()
  })

  it('checks out a branch when its row is clicked', async () => {
    vi.mocked(BranchService.ListLocal).mockResolvedValue([branch('main', true), branch('feature/login')])
    vi.mocked(BranchService.CheckoutBranch).mockResolvedValue()
    const onBranchChanged = renderSidebar()

    await userEvent.click(await screen.findByRole('button', { name: /^feature\/login/ }))

    expect(BranchService.CheckoutBranch).toHaveBeenCalledWith(REPO, 'feature/login')
    expect(onBranchChanged).toHaveBeenCalled()
  })

  it('links a branch named for an issue to that issue instead of checking it out', async () => {
    vi.mocked(BranchService.ListLocal).mockResolvedValue([branch('main', true), branch('123-fix-login')])
    vi.mocked(RemoteService.HostLinks).mockResolvedValue(
      new Links({ issue: 'https://gitlab.com/g/r/-/issues/', root: 'https://gitlab.com/', issuePath: '/-/issues/' }),
    )
    vi.mocked(PlatformService.OpenWebPage).mockResolvedValue()
    renderSidebar()

    const link = await screen.findByRole('link', { name: '123' })
    expect(link).toHaveAttribute('href', 'https://gitlab.com/g/r/-/issues/123')
    expect(screen.getByRole('button', { name: /^123-fix-login/ })).toBeInTheDocument()

    await userEvent.click(link)

    expect(PlatformService.OpenWebPage).toHaveBeenCalledWith('https://gitlab.com/g/r/-/issues/123')
    expect(BranchService.CheckoutBranch).not.toHaveBeenCalled()
  })

  it('creates a branch with spaces turned into dashes, and shows the name first', async () => {
    vi.mocked(BranchService.ListLocal).mockResolvedValue([branch('main', true)])
    vi.mocked(BranchService.CreateBranch).mockResolvedValue()
    vi.mocked(BranchService.CheckoutBranch).mockResolvedValue()
    renderSidebar()

    await userEvent.click(await screen.findByRole('button', { name: 'New branch' }))
    await userEvent.type(screen.getByRole('textbox', { name: 'new branch name' }), 'fix login bug')
    expect(screen.getByRole('status')).toHaveTextContent('Will be created as fix-login-bug')
    await userEvent.click(screen.getByRole('button', { name: 'Create' }))

    expect(BranchService.CreateBranch).toHaveBeenCalledWith(REPO, 'fix-login-bug')
    expect(BranchService.CheckoutBranch).toHaveBeenCalledWith(REPO, 'fix-login-bug')
  })

  it('names a new branch after an assigned issue, asking the host only when the list opens', async () => {
    vi.mocked(BranchService.ListLocal).mockResolvedValue([branch('main', true)])
    vi.mocked(BranchService.CreateBranch).mockResolvedValue()
    vi.mocked(BranchService.CheckoutBranch).mockResolvedValue()
    vi.mocked(HostService.CurrentChange).mockResolvedValue(
      new CurrentChange({ branch: 'main', upstream: '', enabled: true, change: null }),
    )
    vi.mocked(HostService.AssignedIssues).mockResolvedValue([
      { key: '12', number: 12, title: 'Login fails on Windows', state: 'open', webUrl: '' },
    ])
    renderSidebar()

    await userEvent.click(await screen.findByRole('button', { name: 'New branch' }))
    const fromIssue = await screen.findByRole('button', { name: 'From an issue' })
    expect(HostService.AssignedIssues).not.toHaveBeenCalled()
    await userEvent.click(fromIssue)
    const list = await screen.findByRole('list', { name: 'Assigned issues' })
    await userEvent.click(within(list).getByRole('button', { name: /Login fails on Windows/ }))

    expect(screen.getByRole('textbox', { name: 'new branch name' })).toHaveValue('12-login-fails-on-windows')
    expect(screen.queryByRole('list', { name: 'Assigned issues' })).not.toBeInTheDocument()
    await userEvent.click(screen.getByRole('button', { name: 'Create' }))
    expect(BranchService.CreateBranch).toHaveBeenCalledWith(REPO, '12-login-fails-on-windows')
  })

  it('offers no issue picker while the host is off', async () => {
    vi.mocked(BranchService.ListLocal).mockResolvedValue([branch('main', true)])
    vi.mocked(HostService.CurrentChange).mockResolvedValue(
      new CurrentChange({ branch: 'main', upstream: '', enabled: false, change: null }),
    )
    renderSidebar()

    await userEvent.click(await screen.findByRole('button', { name: 'New branch' }))
    await waitFor(() => expect(HostService.CurrentChange).toHaveBeenCalled())
    expect(screen.queryByRole('button', { name: 'From an issue' })).not.toBeInTheDocument()
  })

  it('marks a branch checked out in another worktree', async () => {
    vi.mocked(BranchService.ListLocal).mockResolvedValue([
      branch('main', true),
      { ...branch('review'), worktreePath: '/projects/app-review' },
    ])
    renderSidebar()

    const marker = await screen.findByText('worktree')
    expect(marker).toHaveAttribute('title', 'Checked out in the worktree at /projects/app-review')
    expect(screen.getAllByText('worktree')).toHaveLength(1)
  })

  it('ignores a click on the current branch', async () => {
    vi.mocked(BranchService.ListLocal).mockResolvedValue([branch('main', true)])
    renderSidebar()

    await userEvent.click(await screen.findByRole('button', { name: /^main/ }))

    expect(BranchService.CheckoutBranch).not.toHaveBeenCalled()
  })

  it('offers to restore a deleted branch at its old tip', async () => {
    vi.mocked(BranchService.ListLocal).mockResolvedValue([branch('main', true), branch('spike')])
    vi.mocked(BranchService.DeleteBranch).mockResolvedValue('abc1234def')
    vi.mocked(BranchService.CreateBranchAt).mockResolvedValue()
    renderSidebar()

    await userEvent.click(await screen.findByRole('button', { name: 'Delete spike' }))
    const notice = await screen.findByRole('status')
    expect(notice).toHaveTextContent('Deleted spike')
    await userEvent.click(within(notice).getByRole('button', { name: 'Undo' }))

    expect(BranchService.CreateBranchAt).toHaveBeenCalledWith(REPO, 'spike', 'abc1234def')
    expect(screen.queryByRole('status')).not.toBeInTheDocument()
  })

  it('cleans up merged branches from the header, and restores them all on undo', async () => {
    vi.mocked(BranchService.ListLocal).mockResolvedValue([branch('main', true), branch('a'), branch('b')])
    vi.mocked(BranchService.MergedBranches).mockResolvedValue(
      new MergedBranches({
        default: 'main',
        base: 'origin/main',
        branches: [
          { name: 'a', current: false, upstream: 'origin/a', tip: 'aaa', newer: false, number: 0, prefix: '' },
          { name: 'b', current: false, upstream: 'origin/b', tip: 'bbb', newer: false, number: 0, prefix: '' },
        ],
      }),
    )
    vi.mocked(BranchService.DeleteBranch).mockImplementation((_repo, name) => Promise.resolve(`tip-${name}`) as never)
    vi.mocked(BranchService.CreateBranchAt).mockResolvedValue()
    const onBranchChanged = renderSidebar()

    await userEvent.click(await screen.findByRole('button', { name: 'Clean up merged branches' }))
    const dialog = await screen.findByRole('dialog', { name: 'Clean up merged branches' })
    await userEvent.click(await within(dialog).findByRole('button', { name: 'Delete 2 branches' }))

    const notice = await screen.findByRole('status')
    expect(notice).toHaveTextContent('Deleted 2 branches')
    expect(onBranchChanged).toHaveBeenCalled()
    await userEvent.click(within(notice).getByRole('button', { name: 'Undo' }))

    await waitFor(() => expect(BranchService.CreateBranchAt).toHaveBeenCalledTimes(2))
    expect(BranchService.CreateBranchAt).toHaveBeenCalledWith(REPO, 'a', 'tip-a')
    expect(BranchService.CreateBranchAt).toHaveBeenCalledWith(REPO, 'b', 'tip-b')
  })

  it('asks before force-deleting a branch with unmerged changes', async () => {
    vi.mocked(BranchService.ListLocal).mockResolvedValue([branch('main', true), branch('spike')])
    vi.mocked(BranchService.DeleteBranch)
      .mockRejectedValueOnce(new Error('This branch has unmerged changes.'))
      .mockResolvedValueOnce('abc1234def')
    renderSidebar()

    await userEvent.click(await screen.findByRole('button', { name: 'Delete spike' }))
    const dialog = await screen.findByRole('alertdialog', { name: 'Delete branch' })
    await userEvent.click(within(dialog).getByRole('button', { name: 'Delete' }))

    expect(BranchService.DeleteBranch).toHaveBeenNthCalledWith(1, REPO, 'spike', false)
    expect(BranchService.DeleteBranch).toHaveBeenNthCalledWith(2, REPO, 'spike', true)
  })

  it('merges the default branch into the current branch from its context menu', async () => {
    vi.mocked(BranchService.ListLocal).mockResolvedValue([branch('main'), branch('feature/login', true)])
    vi.mocked(BranchService.DefaultBranch).mockResolvedValue('main')
    vi.mocked(BranchService.MergeBranch).mockResolvedValue()
    const onBranchChanged = renderSidebar()
    const user = userEvent.setup()

    await user.pointer({ keys: '[MouseRight]', target: await screen.findByRole('button', { name: /^feature\/login/ }) })
    await user.click(await screen.findByRole('button', { name: 'Merge main into current' }))

    expect(BranchService.MergeBranch).toHaveBeenCalledWith(REPO, 'main', '')
    await vi.waitFor(() => expect(onBranchChanged).toHaveBeenCalled())
  })

  it('merges with a merge commit when fast-forward is turned off', async () => {
    vi.mocked(BranchService.ListLocal).mockResolvedValue([branch('main', true), branch('feature')])
    vi.mocked(BranchService.MergeBranch).mockResolvedValue()
    const onBranchChanged = renderSidebar()
    const user = userEvent.setup()

    await user.pointer({ keys: '[MouseRight]', target: await screen.findByRole('button', { name: /^feature/ }) })
    await user.click(await screen.findByRole('button', { name: 'Merge into current (no fast-forward)' }))

    expect(BranchService.MergeBranch).toHaveBeenCalledWith(REPO, 'feature', 'no-ff')
    await vi.waitFor(() => expect(onBranchChanged).toHaveBeenCalled())
    expect(screen.queryByRole('status')).not.toBeInTheDocument()
  })

  it('squashes a branch and says to commit the staged changes', async () => {
    vi.mocked(BranchService.ListLocal).mockResolvedValue([branch('main', true), branch('feature')])
    vi.mocked(BranchService.MergeBranch).mockResolvedValue()
    renderSidebar()
    const user = userEvent.setup()

    await user.pointer({ keys: '[MouseRight]', target: await screen.findByRole('button', { name: /^feature/ }) })
    await user.click(await screen.findByRole('button', { name: 'Squash into current' }))

    expect(BranchService.MergeBranch).toHaveBeenCalledWith(REPO, 'feature', 'squash')
    expect(await screen.findByRole('status')).toHaveTextContent(
      'Changes from feature are staged. Commit them to finish the squash.',
    )
  })

  it("deletes a branch's upstream on the remote after confirming", async () => {
    const feature = { ...branch('feature/login'), upstream: 'origin/feature/login' }
    vi.mocked(BranchService.ListLocal).mockResolvedValue([branch('main', true), feature])
    vi.mocked(RemoteService.DeleteRemoteBranch).mockResolvedValue()
    const onBranchChanged = renderSidebar()
    const user = userEvent.setup()

    await user.pointer({ keys: '[MouseRight]', target: await screen.findByRole('button', { name: /^feature\/login/ }) })
    await user.click(await screen.findByRole('button', { name: 'Delete origin/feature/login on remote…' }))
    const dialog = await screen.findByRole('alertdialog', { name: 'Delete on remote' })
    await user.click(within(dialog).getByRole('button', { name: 'Delete' }))

    await vi.waitFor(() => expect(onBranchChanged).toHaveBeenCalled())
    expect(RemoteService.DeleteRemoteBranch).toHaveBeenCalledWith(REPO, 'origin/feature/login', null)
    expect(BranchService.DeleteBranch).not.toHaveBeenCalled()
  })

  it('does not offer to delete on the remote for a branch with no upstream', async () => {
    vi.mocked(BranchService.ListLocal).mockResolvedValue([branch('main', true), branch('spike')])
    renderSidebar()
    const user = userEvent.setup()

    await user.pointer({ keys: '[MouseRight]', target: await screen.findByRole('button', { name: /^spike/ }) })

    expect(await screen.findByRole('button', { name: 'Rename…' })).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: /on remote/ })).not.toBeInTheDocument()
  })

  it("sets a branch's upstream from the remote branches", async () => {
    const feature = { ...branch('feature'), upstream: 'origin/feature' }
    vi.mocked(BranchService.ListLocal).mockResolvedValue([branch('main', true), feature])
    vi.mocked(BranchService.ListRemote).mockResolvedValue(['origin/feature', 'team/origin/feat-x'])
    vi.mocked(BranchService.SetUpstream).mockResolvedValue()
    const onBranchChanged = renderSidebar()
    const user = userEvent.setup()

    await user.pointer({ keys: '[MouseRight]', target: await screen.findByRole('button', { name: /^feature/ }) })
    await user.click(await screen.findByRole('button', { name: 'Set upstream…' }))
    const dialog = await screen.findByRole('dialog', { name: 'Set upstream' })
    const select = await within(dialog).findByRole('combobox', { name: 'Upstream' })
    await vi.waitFor(() => expect(select).toHaveValue('origin/feature'))
    expect(within(dialog).getByRole('button', { name: 'Set upstream' })).toBeDisabled()
    await user.selectOptions(select, 'team/origin/feat-x')
    await user.click(within(dialog).getByRole('button', { name: 'Set upstream' }))

    expect(BranchService.SetUpstream).toHaveBeenCalledWith(REPO, 'feature', 'team/origin/feat-x')
    await vi.waitFor(() => expect(onBranchChanged).toHaveBeenCalled())
    expect(screen.queryByRole('dialog', { name: 'Set upstream' })).not.toBeInTheDocument()
  })

  it("unsets a branch's upstream", async () => {
    const feature = { ...branch('feature'), upstream: 'origin/feature' }
    vi.mocked(BranchService.ListLocal).mockResolvedValue([branch('main', true), feature])
    vi.mocked(BranchService.UnsetUpstream).mockResolvedValue()
    const onBranchChanged = renderSidebar()
    const user = userEvent.setup()

    await user.pointer({ keys: '[MouseRight]', target: await screen.findByRole('button', { name: /^feature/ }) })
    await user.click(await screen.findByRole('button', { name: 'Unset upstream' }))

    expect(BranchService.UnsetUpstream).toHaveBeenCalledWith(REPO, 'feature')
    await vi.waitFor(() => expect(onBranchChanged).toHaveBeenCalled())
  })

  it('offers no Unset upstream for a branch without one', async () => {
    vi.mocked(BranchService.ListLocal).mockResolvedValue([branch('main', true), branch('spike')])
    renderSidebar()
    const user = userEvent.setup()

    await user.pointer({ keys: '[MouseRight]', target: await screen.findByRole('button', { name: /^spike/ }) })

    expect(await screen.findByRole('button', { name: 'Set upstream…' })).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Unset upstream' })).not.toBeInTheDocument()
  })

  it('does not offer to merge the default branch into itself', async () => {
    vi.mocked(BranchService.ListLocal).mockResolvedValue([branch('main', true), branch('feature/login')])
    vi.mocked(BranchService.DefaultBranch).mockResolvedValue('main')
    renderSidebar()
    const user = userEvent.setup()

    await vi.waitFor(() => expect(BranchService.DefaultBranch).toHaveBeenCalled())
    await user.pointer({ keys: '[MouseRight]', target: await screen.findByRole('button', { name: /^main/ }) })

    expect(screen.getByRole('button', { name: 'Merge into current' })).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Merge main into current' })).not.toBeInTheDocument()
  })

  it('asks before merging a default branch that is behind its upstream', async () => {
    const main = { ...branch('main'), upstream: 'origin/main', behind: 3 }
    vi.mocked(BranchService.ListLocal).mockResolvedValue([main, branch('feature/login', true)])
    vi.mocked(BranchService.DefaultBranch).mockResolvedValue('main')
    vi.mocked(BranchService.MergeBranch).mockResolvedValue()
    renderSidebar()
    const user = userEvent.setup()

    await user.pointer({ keys: '[MouseRight]', target: await screen.findByRole('button', { name: /^feature\/login/ }) })
    await user.click(await screen.findByRole('button', { name: 'Merge main into current' }))
    const dialog = await screen.findByRole('alertdialog', { name: 'Merge main' })
    expect(dialog).toHaveTextContent('main is 3 commits behind origin/main.')
    await user.click(within(dialog).getByRole('button', { name: 'Merge anyway' }))

    expect(BranchService.MergeBranch).toHaveBeenCalledWith(REPO, 'main', '')
  })

  it('does not merge a stale default branch when the warning is cancelled', async () => {
    const main = { ...branch('main'), upstream: 'origin/main', behind: 1 }
    vi.mocked(BranchService.ListLocal).mockResolvedValue([main, branch('feature/login', true)])
    vi.mocked(BranchService.DefaultBranch).mockResolvedValue('main')
    renderSidebar()
    const user = userEvent.setup()

    await user.pointer({ keys: '[MouseRight]', target: await screen.findByRole('button', { name: /^feature\/login/ }) })
    await user.click(await screen.findByRole('button', { name: 'Merge main into current' }))
    const dialog = await screen.findByRole('alertdialog', { name: 'Merge main' })
    await user.click(within(dialog).getByRole('button', { name: 'Cancel' }))

    expect(BranchService.MergeBranch).not.toHaveBeenCalled()
  })

  it('opens a branch menu from its More actions button and from Shift+F10', async () => {
    vi.mocked(BranchService.ListLocal).mockResolvedValue([branch('main', true), branch('spike')])
    renderSidebar()
    const user = userEvent.setup()

    await user.click(await screen.findByRole('button', { name: 'More actions for spike' }))
    expect(await screen.findByRole('button', { name: 'Check out' })).toHaveFocus()
    expect(BranchService.CheckoutBranch).not.toHaveBeenCalled()
    await user.keyboard('{Escape}')

    screen.getByRole('button', { name: /^main/ }).focus()
    await user.keyboard('{Shift>}{F10}{/Shift}')
    expect(await screen.findByRole('button', { name: 'Set upstream…' })).toBeInTheDocument()
  })

  it('names the branch a new repository is on instead of saying it has none', async () => {
    vi.mocked(BranchService.ListLocal).mockResolvedValue([])
    vi.mocked(BranchService.CurrentBranchStatus).mockResolvedValue({ current: 'main' } as BranchStatusInfo)
    renderSidebar()

    expect(await screen.findByText('No commits on main yet.')).toBeInTheDocument()
  })

  it('shows why branches could not load', async () => {
    vi.mocked(BranchService.ListLocal).mockRejectedValue(new Error('Git command failed.'))
    renderSidebar()

    expect(await screen.findByText('Git command failed.')).toBeInTheDocument()
  })
})

describe('BranchSidebar overlap prediction', () => {
  const report = (supported: boolean) => ({
    supported,
    overlaps: [
      { branch: 'spike', remote: false, files: ['src/app.ts', 'README.md'] },
      { branch: 'origin/login', remote: true, files: ['src/app.ts'] },
    ],
  })

  it('badges local and remote branches that would conflict', async () => {
    vi.mocked(BranchService.ListLocal).mockResolvedValue([branch('main', true), branch('spike'), branch('docs')])
    vi.mocked(BranchService.ListRemote).mockResolvedValue(['origin/login', 'origin/main'])
    vi.mocked(RemoteService.List).mockResolvedValue([{ name: 'origin', fetchUrl: 'git@github.com:g/r.git' }] as never)
    vi.mocked(OverlapService.Predict).mockResolvedValue(report(true))
    const sections = defaultSections()
    renderSidebar(vi.fn(), sections)

    const badge = await screen.findByTitle('Would conflict with the current branch: src/app.ts, README.md')
    expect(badge).toHaveTextContent('conflicts')
    expect(screen.getByRole('button', { name: /^spike/ })).toContainElement(badge)
    expect(screen.queryByRole('button', { name: /^docs/ })).not.toContainElement(
      screen.queryByText('conflicts', { selector: '.branch-row:nth-child(3) *' }),
    )

    // The Remote group is closed by default, so its header carries the count.
    expect(screen.getByTitle('1 remote branch would conflict with the current branch')).toHaveTextContent('1 conflicts')
    expect(screen.queryByText('origin/login')).not.toBeInTheDocument()
    await userEvent.click(screen.getByRole('button', { name: 'Remote, 2' }))
    expect(sections.toggle).toHaveBeenCalledWith('branches-remote', false)
  })

  it('shows nothing when Git is too old to predict', async () => {
    vi.mocked(BranchService.ListLocal).mockResolvedValue([branch('main', true), branch('spike')])
    vi.mocked(OverlapService.Predict).mockResolvedValue({ supported: false, overlaps: [] })
    renderSidebar()

    await screen.findByRole('button', { name: /^spike/ })
    expect(screen.queryByText('conflicts')).not.toBeInTheDocument()
  })
})
