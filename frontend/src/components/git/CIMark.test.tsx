import { render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { CIMark as CIMarkInfo, HostService, PlatformService } from '@current-client-bindings/app'
import { CIStatus, Job } from '@current-client-bindings/core/hostapi'
import CIMark, { formatDuration } from './CIMark'

describe('CIMark', () => {
  it('shows the status and lists the jobs on click, fetched only then', async () => {
    vi.mocked(HostService.PipelineJobs).mockResolvedValue([
      new Job({ id: 5, name: 'test', stage: 'unit', status: CIStatus.CIFailed, duration: 42.5e9 as Job['duration'] }),
      new Job({
        id: 6,
        name: 'build',
        stage: 'compile',
        status: CIStatus.CISuccess,
        duration: 825e9 as Job['duration'],
      }),
    ])
    vi.mocked(PlatformService.OpenWebPage).mockResolvedValue()
    render(
      <CIMark
        repoPath="/repos/app"
        mark={new CIMarkInfo({ status: 'failed', pipelineId: 12, webUrl: 'https://gitlab.example.com/p/12' })}
      />,
    )

    const mark = screen.getByRole('button', { name: 'CI failed' })
    expect(mark).toHaveClass('ci-mark-failed')
    expect(HostService.PipelineJobs).not.toHaveBeenCalled()

    await userEvent.click(mark)
    const jobs = await screen.findByRole('list', { name: 'Jobs' })
    expect(within(jobs).getByText('test')).toBeInTheDocument()
    expect(within(jobs).getByText('43s')).toBeInTheDocument()
    expect(within(jobs).getByText('13m 45s')).toBeInTheDocument()
    expect(HostService.PipelineJobs).toHaveBeenCalledWith('/repos/app', 12)
    await userEvent.click(screen.getByRole('button', { name: 'Open on host' }))
    expect(PlatformService.OpenWebPage).toHaveBeenCalledWith('https://gitlab.example.com/p/12')
  })

  it('renders nothing without a pipeline', () => {
    render(<CIMark repoPath="/repos/app" mark={undefined} />)
    expect(screen.queryByRole('button')).not.toBeInTheDocument()
  })

  it('formats durations from nanoseconds', () => {
    expect(formatDuration(0)).toBe('')
    expect(formatDuration(59e9)).toBe('59s')
    expect(formatDuration(3_725e9)).toBe('1h 2m')
  })
})
