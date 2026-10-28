import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { HostService, IssueDetails, IssueList } from '@current-client-bindings/app'
import { IssuePanelContext } from '../../lib/issuePanel'
import IssuesView from './IssuesView'

const REPO = '/repos/app'

function details(key: string, title: string) {
  return new IssueDetails({
    reference: /^\d+$/.test(key) ? `#${key}` : key,
    descriptionHtml: '',
    issue: {
      key,
      number: Number(key) || 0,
      title,
      state: 'open',
      webUrl: '',
      description: '',
      labels: [],
      milestone: '',
      author: { name: 'Ada', username: 'ada' },
      createdAt: '2026-10-20T10:00:00Z',
      updatedAt: '2026-10-20T10:00:00Z',
    },
    comments: [],
  } as never)
}

const fetched = new Date().toISOString()

function list(over: Partial<IssueList> = {}) {
  return new IssueList({
    enabled: true,
    sources: [
      { name: 'github.com', fetchedAt: fetched },
      { name: 'acme.atlassian.net', fetchedAt: fetched },
    ],
    issues: [
      {
        issue: { key: '18', number: 18, title: 'Coupon codes', state: 'open', webUrl: '' },
        source: 'github.com',
        branch: '18-coupon-codes',
      },
      {
        issue: { key: 'WEB-57', number: 0, title: 'Yearly billing', state: 'open', webUrl: '' },
        source: 'acme.atlassian.net',
        branch: '',
      },
    ],
    ...over,
  } as never)
}

function renderView(selectedKey: string | null = null, create = vi.fn()) {
  vi.mocked(HostService.IssueDetails).mockImplementation(
    (_repo, key) => Promise.resolve(details(key, key === '18' ? 'Coupon codes' : 'Yearly billing')) as never,
  )
  render(
    <IssuePanelContext.Provider value={{ open: vi.fn(), create }}>
      <IssuesView repoPath={REPO} selectedKey={selectedKey} refreshKey={0} onBranchChanged={vi.fn()} />
    </IssuePanelContext.Provider>,
  )
  return create
}

describe('IssuesView', () => {
  it('lists the host’s and the tracker’s issues assigned to you, the first in the pane', async () => {
    vi.mocked(HostService.MyIssues).mockResolvedValue(list())
    renderView()

    const rows = await screen.findByRole('list', { name: 'Issues list' })
    expect(within(rows).getByText('#18')).toBeInTheDocument()
    expect(within(rows).getByText('WEB-57')).toBeInTheDocument()
    expect(within(rows).getByText('18-coupon-codes')).toBeInTheDocument()
    expect(screen.getByText(/github\.com · acme\.atlassian\.net · fetched/)).toBeInTheDocument()
    expect(await screen.findByRole('article', { name: '#18' })).toBeInTheDocument()
    expect(HostService.Refresh).not.toHaveBeenCalled()

    await userEvent.click(within(rows).getByRole('button', { name: /Yearly billing/ }))
    expect(await screen.findByRole('article', { name: 'WEB-57' })).toBeInTheDocument()
  })

  it('opens on the issue it is pointed at, and creates an issue from its header', async () => {
    vi.mocked(HostService.MyIssues).mockResolvedValue(list())
    const create = renderView('WEB-57')

    expect(await screen.findByRole('article', { name: 'WEB-57' })).toBeInTheDocument()
    await userEvent.click(screen.getByRole('button', { name: 'New issue' }))
    expect(create).toHaveBeenCalled()
  })

  it('refreshes once when nothing was read yet', async () => {
    vi.mocked(HostService.MyIssues)
      .mockResolvedValueOnce(list({ sources: [{ name: 'github.com', fetchedAt: null }], issues: [] } as never))
      .mockResolvedValue(list())
    vi.mocked(HostService.Refresh).mockResolvedValue()
    renderView()

    await waitFor(() => expect(HostService.Refresh).toHaveBeenCalledTimes(1))
    expect(await screen.findByText('Coupon codes', { selector: '.host-change-row-title' })).toBeInTheDocument()
  })

  it('asks for All and Closed only when chosen, once each', async () => {
    vi.mocked(HostService.MyIssues).mockResolvedValue(list())
    vi.mocked(HostService.ProjectIssues).mockImplementation(
      (_repo, open) =>
        Promise.resolve(
          list({
            issues: [
              {
                issue: {
                  key: open ? '30' : '7',
                  number: open ? 30 : 7,
                  title: open ? 'Someone else’s' : 'Long done',
                  state: open ? 'open' : 'closed',
                  webUrl: '',
                },
                source: 'github.com',
                branch: '',
              },
            ],
          } as never),
        ) as never,
    )
    renderView()

    await screen.findByRole('list', { name: 'Issues list' })
    expect(HostService.ProjectIssues).not.toHaveBeenCalled()
    await userEvent.click(screen.getByRole('button', { name: 'Closed' }))
    expect(await screen.findByText('Long done', { selector: '.host-change-row-title' })).toBeInTheDocument()
    expect(HostService.ProjectIssues).toHaveBeenCalledWith(REPO, false)
    await userEvent.click(screen.getByRole('button', { name: 'All' }))
    expect(await screen.findByText('Someone else’s', { selector: '.host-change-row-title' })).toBeInTheDocument()
    expect(HostService.ProjectIssues).toHaveBeenCalledWith(REPO, true)
    await userEvent.click(screen.getByRole('button', { name: 'Mine 2' }))
    expect(await screen.findByText('Coupon codes', { selector: '.host-change-row-title' })).toBeInTheDocument()
  })

  it('renders nothing while no host or tracker lists issues', async () => {
    vi.mocked(HostService.MyIssues).mockResolvedValue(new IssueList({ enabled: false, sources: [], issues: [] }))
    renderView()
    await waitFor(() => expect(HostService.MyIssues).toHaveBeenCalled())
    expect(screen.queryByRole('region', { name: 'Issues' })).not.toBeInTheDocument()
  })
})
