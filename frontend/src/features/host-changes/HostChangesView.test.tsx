import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { ChangeList, CurrentChange, ForecastReport, HostService, PlatformService } from '@current-client-bindings/app'
import { Change, ChangeState, CIStatus } from '@current-client-bindings/core/hostapi'
import { DialogProvider } from '../../components/chrome/DialogProvider'
import HostChangesView from './HostChangesView'

const REPO = '/repos/app'

function change(overrides: Partial<Change>): Change {
  return new Change({
    number: 138,
    title: 'Add the thing',
    state: ChangeState.ChangeOpen,
    sourceBranch: 'feature',
    targetBranch: 'main',
    author: { name: 'Ada Lovelace', username: 'ada' },
    labels: ['backend'],
    headSha: 'abc1234def',
    ci: CIStatus.CISuccess,
    updatedAt: new Date().toISOString(),
    webUrl: 'https://gitlab.example.com/g/r/-/merge_requests/138',
    ...overrides,
  })
}

function list(overrides: Partial<ChangeList> = {}): ChangeList {
  return new ChangeList({
    host: 'gitlab.example.com',
    provider: 'gitlab',
    project: 'g/r',
    enabled: true,
    prefix: '!',
    me: 'ada',
    changes: [
      change({}),
      change({ number: 140, title: 'Fix a typo', author: { name: 'Bob', username: 'bob' }, draft: true, labels: [] }),
    ],
    fetchedAt: new Date().toISOString(),
    ...overrides,
  })
}

function renderView({ onBranchChanged = vi.fn(), onCreateChange = vi.fn(), onOpenSettings = vi.fn() } = {}) {
  vi.mocked(HostService.Forecast).mockResolvedValue(
    new ForecastReport({
      enabled: true,
      branch: 'main',
      conflicts: [
        { number: 138, prefix: '!', title: '', author: '', sourceBranch: 'feature', files: ['app/x.go'], webUrl: '' },
      ],
    }),
  )
  vi.mocked(HostService.CIStatus).mockResolvedValue({})
  vi.mocked(HostService.CurrentChange).mockResolvedValue(
    new CurrentChange({ branch: 'topic', upstream: 'origin/topic', enabled: true, change: null }),
  )
  render(
    <DialogProvider>
      <HostChangesView
        repoPath={REPO}
        label="Merge requests"
        onBranchChanged={onBranchChanged}
        onCreateChange={onCreateChange}
        onOpenSettings={onOpenSettings}
      />
    </DialogProvider>,
  )
  return { onBranchChanged, onCreateChange, onOpenSettings }
}

describe('HostChangesView', () => {
  it('lists the open changes from the cache and filters to mine', async () => {
    vi.mocked(HostService.Changes).mockResolvedValue(list())
    vi.mocked(HostService.ChangeDetails).mockResolvedValue({ change: change({}), descriptionHtml: '<p>Body</p>' })
    renderView()

    const rows = await screen.findByRole('list', { name: 'Merge requests list' })
    expect(within(rows).getByText('Add the thing')).toBeInTheDocument()
    expect(within(rows).getByText('Fix a typo')).toBeInTheDocument()
    expect(HostService.Refresh).not.toHaveBeenCalled()
    expect(screen.getByText(/gitlab\.example\.com · fetched/)).toBeInTheDocument()

    await userEvent.click(screen.getByRole('button', { name: 'Mine 1' }))
    expect(within(rows).queryByText('Fix a typo')).not.toBeInTheDocument()
    expect(HostService.ClosedChanges).not.toHaveBeenCalled()
  })

  it('asks for closed changes only when that filter is chosen', async () => {
    vi.mocked(HostService.Changes).mockResolvedValue(list())
    vi.mocked(HostService.ChangeDetails).mockResolvedValue({ change: change({}), descriptionHtml: '' })
    vi.mocked(HostService.ClosedChanges).mockResolvedValue(
      list({ changes: [change({ number: 99, title: 'Shipped it', state: ChangeState.ChangeMerged })] }),
    )
    renderView()

    await screen.findByRole('list', { name: 'Merge requests list' })
    await userEvent.click(screen.getByRole('button', { name: 'Closed' }))
    const rows = screen.getByRole('list', { name: 'Merge requests list' })
    expect(await within(rows).findByText('Shipped it')).toBeInTheDocument()
    expect(HostService.ClosedChanges).toHaveBeenCalledWith(REPO)
    expect(within(rows).queryByText('Add the thing')).not.toBeInTheDocument()
  })

  it('refreshes once when nothing was fetched yet, and on the button', async () => {
    vi.mocked(HostService.Changes)
      .mockResolvedValueOnce(list({ fetchedAt: null, changes: [] }))
      .mockResolvedValue(list())
    vi.mocked(HostService.Refresh).mockResolvedValue()
    vi.mocked(HostService.ChangeDetails).mockResolvedValue({ change: change({}), descriptionHtml: '' })
    renderView()

    await waitFor(() => expect(HostService.Refresh).toHaveBeenCalledTimes(1))
    const rows = await screen.findByRole('list', { name: 'Merge requests list' })
    expect(await within(rows).findByText('Add the thing')).toBeInTheDocument()
    await userEvent.click(screen.getByRole('button', { name: 'Refresh' }))
    expect(HostService.Refresh).toHaveBeenCalledTimes(2)
  })

  it('shows the selected change with its rendered description and opens it on the host', async () => {
    vi.mocked(HostService.Changes).mockResolvedValue(list())
    vi.mocked(HostService.ChangeDetails).mockResolvedValue({
      change: change({}),
      descriptionHtml: '<p>See <a href="https://gitlab.example.com/g/r/-/issues/4">#4</a></p>',
    })
    vi.mocked(PlatformService.OpenWebPage).mockResolvedValue()
    renderView()

    const detail = await screen.findByRole('article', { name: '!138' })
    expect(await within(detail).findByText('#4')).toBeInTheDocument()
    expect(within(detail).getByText('feature → main')).toBeInTheDocument()
    expect(within(detail).getByText(/Would conflict with main in/)).toBeInTheDocument()
    expect(within(detail).getByText('CI passed')).toBeInTheDocument()
    await userEvent.click(within(detail).getByRole('button', { name: 'Open on gitlab.example.com' }))
    expect(PlatformService.OpenWebPage).toHaveBeenCalledWith('https://gitlab.example.com/g/r/-/merge_requests/138')
    expect(HostService.ChangeDetails).toHaveBeenCalledWith(REPO, 138)
  })

  it('fetches threads, files and commits only when their tab is opened', async () => {
    vi.mocked(HostService.Changes).mockResolvedValue(list())
    vi.mocked(HostService.ChangeDetails).mockResolvedValue({ change: change({}), descriptionHtml: '' })
    vi.mocked(HostService.ChangeFiles).mockResolvedValue([
      { path: 'app/x.go', oldPath: '', status: 'modified', additions: 3, deletions: 1 },
      { path: 'app/y.go', oldPath: 'app/old.go', status: 'renamed', additions: 0, deletions: 0 },
    ])
    vi.mocked(HostService.ChangeCommits).mockResolvedValue([
      { sha: 'abc1234def', subject: 'feat: x', author: 'Ada', date: new Date().toISOString() },
    ])
    vi.mocked(HostService.ChangeThreads).mockResolvedValue({
      enabled: true,
      number: 138,
      prefix: '!',
      total: 1,
      unresolved: { 'app/x.go': 1 },
      threads: [
        {
          id: 't1',
          path: 'app/x.go',
          line: 12,
          oldLine: 0,
          resolved: false,
          outdated: false,
          notes: [{ author: 'Bob', bodyHtml: '<p>Why?</p>', createdAt: new Date().toISOString() }],
        },
      ],
    })
    renderView()

    const detail = await screen.findByRole('article', { name: '!138' })
    expect(HostService.ChangeFiles).not.toHaveBeenCalled()

    await userEvent.click(within(detail).getByRole('button', { name: 'Files' }))
    const files = await within(detail).findByRole('list', { name: 'Files' })
    expect(within(files).getByText('app/x.go')).toBeInTheDocument()
    expect(within(files).getByText('from app/old.go')).toBeInTheDocument()
    expect(await within(detail).findByRole('button', { name: /^Files ?2$/ })).toBeInTheDocument()

    await userEvent.click(within(detail).getByRole('button', { name: 'Commits' }))
    expect(await within(detail).findByText('feat: x')).toBeInTheDocument()
    expect(within(detail).getByText('abc1234')).toBeInTheDocument()

    await userEvent.click(within(detail).getByRole('button', { name: 'Threads' }))
    expect(await within(detail).findByText('Why?')).toBeInTheDocument()
    expect(within(detail).getByRole('group', { name: /app\/x\.go:12/ })).toBeInTheDocument()
    expect(HostService.ChangeThreads).toHaveBeenCalledWith(REPO, 138)
    expect(HostService.ChangeFiles).toHaveBeenCalledTimes(1)
  })

  it('starts a new change from the checked-out branch, unless it already has one', async () => {
    vi.mocked(HostService.Changes).mockResolvedValue(list())
    vi.mocked(HostService.ChangeDetails).mockResolvedValue({ change: change({}), descriptionHtml: '' })
    const { onCreateChange } = renderView()

    const create = await screen.findByRole('button', { name: 'New merge request' })
    await waitFor(() => expect(create).toBeEnabled())
    await userEvent.click(create)
    expect(onCreateChange).toHaveBeenCalledWith('topic')
  })

  it('names the host in the privacy line and opens the integration settings from it', async () => {
    vi.mocked(HostService.Changes).mockResolvedValue(list())
    vi.mocked(HostService.ChangeDetails).mockResolvedValue({ change: change({}), descriptionHtml: '' })
    const { onOpenSettings } = renderView()

    expect(await screen.findByText(/gitlab\.example\.com is contacted on Refresh/)).toBeInTheDocument()
    await userEvent.click(screen.getByRole('button', { name: 'Integration settings' }))
    expect(onOpenSettings).toHaveBeenCalled()
  })

  it('checks a change out locally and tells the app the branch changed', async () => {
    vi.mocked(HostService.Changes).mockResolvedValue(list())
    vi.mocked(HostService.ChangeDetails).mockResolvedValue({ change: change({}), descriptionHtml: '' })
    vi.mocked(HostService.CheckoutChange).mockResolvedValue({ branch: 'feature', existed: false })
    const { onBranchChanged } = renderView()

    await userEvent.click(await screen.findByRole('button', { name: 'Check out locally' }))

    expect(HostService.CheckoutChange).toHaveBeenCalledWith(REPO, 138, null)
    expect(await screen.findByRole('status')).toHaveTextContent('Checked out feature.')
    expect(onBranchChanged).toHaveBeenCalled()
  })

  it('renders nothing while the host is off', async () => {
    vi.mocked(HostService.Changes).mockResolvedValue(new ChangeList({ enabled: false }))
    renderView()
    await waitFor(() => expect(HostService.Changes).toHaveBeenCalled())
    expect(screen.queryByRole('region')).not.toBeInTheDocument()
  })
})
