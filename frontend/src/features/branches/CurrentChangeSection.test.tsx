import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import {
  BranchChange,
  CIMark,
  CurrentChange,
  ForecastReport,
  HostService,
  PlatformService,
  RemoteService,
} from '@current-client-bindings/app'
import CurrentChangeSection from './CurrentChangeSection'

const change = new BranchChange({
  number: 138,
  prefix: '!',
  state: 'open',
  draft: true,
  title: 'Add the thing',
  targetBranch: 'main',
  unresolvedThreads: 2,
  webUrl: 'https://gitlab.example.com/g/r/-/merge_requests/138',
})

describe('CurrentChangeSection', () => {
  beforeEach(() => {
    vi.mocked(HostService.Forecast).mockResolvedValue(new ForecastReport({ enabled: true, conflicts: [] }))
  })

  it('lists the open changes that would conflict with the branch, even without a change of its own', async () => {
    vi.mocked(HostService.CurrentChange).mockResolvedValue(
      new CurrentChange({ branch: 'main', enabled: true, change: null }),
    )
    vi.mocked(HostService.Forecast).mockResolvedValue(
      new ForecastReport({
        enabled: true,
        branch: 'main',
        base: 'main',
        checked: 20,
        total: 34,
        conflicts: [
          {
            number: 138,
            prefix: '!',
            title: 'Add the thing',
            author: 'Ada',
            sourceBranch: 'feature',
            files: ['app/x.go', 'README.md'],
            webUrl: 'https://h/138',
          },
        ],
      }),
    )
    const onOpenChange = vi.fn()
    render(<CurrentChangeSection repoPath="/repos/app" refreshKey={0} onOpenChange={onOpenChange} />)

    const section = await screen.findByRole('region', { name: 'Pull request for main' })
    const conflict = within(section).getByRole('button', { name: /Conflicts with !138 by Ada in app\/x.go, README.md/ })
    expect(within(section).getByText('20 of 34 checked')).toBeInTheDocument()
    await userEvent.click(conflict)
    expect(onOpenChange).toHaveBeenCalledWith(138)
  })

  it('shows the checked-out branch’s merge request with CI, failed jobs and actions', async () => {
    vi.mocked(HostService.CurrentChange).mockResolvedValue(
      new CurrentChange({
        branch: 'feature',
        upstream: 'origin/feature',
        enabled: true,
        kind: 'merge request',
        change,
        ci: new CIMark({ status: 'failed', pipelineId: 12, webUrl: 'https://gitlab.example.com/p/12' }),
        failedJobs: ['test', 'lint'],
      }),
    )
    vi.mocked(PlatformService.OpenWebPage).mockResolvedValue()
    const onOpenChange = vi.fn()
    render(<CurrentChangeSection repoPath="/repos/app" refreshKey={0} onOpenChange={onOpenChange} />)

    const section = await screen.findByRole('region', { name: 'Merge request for feature' })
    expect(within(section).getByText('Add the thing')).toBeInTheDocument()
    expect(within(section).getByText('draft')).toBeInTheDocument()
    expect(within(section).getByText('→ main')).toBeInTheDocument()
    expect(within(section).getByText('failed: test, lint')).toBeInTheDocument()
    expect(within(section).getByText('2 unresolved')).toBeInTheDocument()
    await userEvent.click(within(section).getByRole('button', { name: 'Details' }))
    expect(onOpenChange).toHaveBeenCalledWith(138)
    await userEvent.click(within(section).getByRole('button', { name: 'Open on host' }))
    expect(PlatformService.OpenWebPage).toHaveBeenCalledWith('https://gitlab.example.com/g/r/-/merge_requests/138')
    expect(HostService.CurrentChange).toHaveBeenCalledWith('/repos/app')
  })

  it('is one row, "No merge request · Create…", when the pushed branch has none', async () => {
    vi.mocked(HostService.CurrentChange).mockResolvedValue(
      new CurrentChange({
        branch: 'feature',
        upstream: 'origin/feature',
        enabled: true,
        proposable: true,
        kind: 'merge request',
        change: null,
      }),
    )
    vi.mocked(RemoteService.CreateChangeLink).mockResolvedValue('https://gitlab.com/g/r/-/merge_requests/new?x')
    vi.mocked(PlatformService.OpenWebPage).mockResolvedValue()
    render(<CurrentChangeSection repoPath="/repos/app" refreshKey={0} />)

    const row = await screen.findByRole('button', { name: 'No merge request · Create…' })
    expect(row).toHaveClass('current-change-row')
    expect(screen.queryByText('Merge request')).not.toBeInTheDocument()
    await userEvent.click(row)
    expect(RemoteService.CreateChangeLink).toHaveBeenCalledWith('/repos/app', 'feature')
    await waitFor(() =>
      expect(PlatformService.OpenWebPage).toHaveBeenCalledWith('https://gitlab.com/g/r/-/merge_requests/new?x'),
    )
  })

  it('is one row, "Push to create a pull request", when the branch has no upstream, and pushes from it', async () => {
    vi.mocked(HostService.CurrentChange).mockResolvedValue(
      new CurrentChange({ branch: 'topic', enabled: true, proposable: true, kind: 'pull request', change: null }),
    )
    const onPushToCreate = vi.fn()
    render(<CurrentChangeSection repoPath="/repos/app" refreshKey={0} onPushToCreate={onPushToCreate} />)

    const row = await screen.findByRole('button', { name: 'Push to create a pull request' })
    expect(row).toHaveClass('current-change-row')
    await userEvent.click(row)
    expect(onPushToCreate).toHaveBeenCalled()
  })

  it('is absent on a branch with nothing to propose', async () => {
    vi.mocked(HostService.CurrentChange).mockResolvedValue(
      new CurrentChange({ branch: 'main', upstream: 'origin/main', enabled: true, proposable: false, change: null }),
    )
    const { container } = render(<CurrentChangeSection repoPath="/repos/app" refreshKey={0} />)

    await waitFor(() => expect(HostService.Forecast).toHaveBeenCalled())
    expect(container).toBeEmptyDOMElement()
  })

  it('is absent when the host is off', async () => {
    vi.mocked(HostService.CurrentChange).mockResolvedValue(
      new CurrentChange({ branch: 'feature', enabled: false, change }),
    )
    render(<CurrentChangeSection repoPath="/repos/other" refreshKey={0} />)
    await waitFor(() => expect(HostService.CurrentChange).toHaveBeenCalledWith('/repos/other'))
    expect(screen.queryByRole('region')).not.toBeInTheDocument()
  })
})
