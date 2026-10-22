import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { BranchChange, CIMark, CurrentChange, HostService, PlatformService } from '@current-client-bindings/app'
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
  it('shows the checked-out branch’s merge request with CI, failed jobs and actions', async () => {
    vi.mocked(HostService.CurrentChange).mockResolvedValue(
      new CurrentChange({
        branch: 'feature',
        upstream: 'origin/feature',
        enabled: true,
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

  it('is absent when the branch has no known merge request or the host is off', async () => {
    vi.mocked(HostService.CurrentChange).mockResolvedValue(
      new CurrentChange({ branch: 'main', enabled: true, change: null }),
    )
    const { rerender } = render(<CurrentChangeSection repoPath="/repos/app" refreshKey={0} />)
    await waitFor(() => expect(HostService.CurrentChange).toHaveBeenCalled())
    expect(screen.queryByRole('region')).not.toBeInTheDocument()

    vi.mocked(HostService.CurrentChange).mockResolvedValue(
      new CurrentChange({ branch: 'feature', enabled: false, change }),
    )
    rerender(<CurrentChangeSection repoPath="/repos/other" refreshKey={0} />)
    await waitFor(() => expect(HostService.CurrentChange).toHaveBeenCalledWith('/repos/other'))
    expect(screen.queryByRole('region')).not.toBeInTheDocument()
  })
})
