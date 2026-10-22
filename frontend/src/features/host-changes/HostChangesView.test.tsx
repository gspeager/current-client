import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { ChangeList, HostService, IdentityService, PlatformService } from '@current-client-bindings/app'
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

function renderView(onBranchChanged = vi.fn()) {
  vi.mocked(IdentityService.GetIdentities).mockResolvedValue([{ initials: 'AL', color: '#123456' }])
  render(
    <DialogProvider>
      <HostChangesView repoPath={REPO} label="Merge requests" onBranchChanged={onBranchChanged} />
    </DialogProvider>,
  )
  return onBranchChanged
}

describe('HostChangesView', () => {
  it('lists the open changes from the cache and filters to mine', async () => {
    vi.mocked(HostService.Changes).mockResolvedValue(list())
    vi.mocked(HostService.ChangeDetails).mockResolvedValue({ change: change({}), descriptionHtml: '<p>Body</p>' })
    renderView()

    const rows = await screen.findByRole('list', { name: 'Merge requests list' })
    expect(within(rows).getByText('Add the thing')).toBeInTheDocument()
    expect(within(rows).getByText('Fix a typo')).toBeInTheDocument()
    expect(within(rows).getByText('backend')).toBeInTheDocument()
    expect(HostService.Refresh).not.toHaveBeenCalled()

    await userEvent.click(screen.getByRole('checkbox', { name: 'Authored by me' }))
    expect(within(rows).queryByText('Fix a typo')).not.toBeInTheDocument()
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
    await userEvent.click(within(detail).getByRole('button', { name: 'Open on host' }))
    expect(PlatformService.OpenWebPage).toHaveBeenCalledWith('https://gitlab.example.com/g/r/-/merge_requests/138')
    expect(HostService.ChangeDetails).toHaveBeenCalledWith(REPO, 138)
  })

  it('checks a change out locally and tells the app the branch changed', async () => {
    vi.mocked(HostService.Changes).mockResolvedValue(list())
    vi.mocked(HostService.ChangeDetails).mockResolvedValue({ change: change({}), descriptionHtml: '' })
    vi.mocked(HostService.CheckoutChange).mockResolvedValue({ branch: 'feature', existed: false })
    const onBranchChanged = renderView()

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
