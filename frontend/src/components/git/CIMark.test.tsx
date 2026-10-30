import { render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { BranchChange, CIMark as CIMarkInfo, HostService, PlatformService } from '@current-client-bindings/app'
import { CIStatus, Job } from '@current-client-bindings/core/hostapi'
import CIMark, { ciLabel, formatDuration, PipelineJobList } from './CIMark'
import ChangeMarker from './ChangeMarker'

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
    expect(mark.querySelector('svg')).toHaveClass('ci-icon-failed')
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

  // Each status has its own icon, so results read without color. The mark,
  // the PR marker and the job rows share the map.
  it('shows one icon and label per status everywhere a result appears', async () => {
    const icons: Record<string, string> = {
      success: 'circle-check',
      failed: 'circle-x',
      running: 'circle-dashed',
      pending: 'clock',
      canceled: 'circle-slash',
      skipped: 'circle-minus',
    }
    const statuses = Object.keys(icons)
    vi.mocked(HostService.PipelineJobs).mockResolvedValue(
      statuses.map((status, i) => new Job({ id: i, name: `job-${status}`, stage: 's', status: status as CIStatus })),
    )
    render(
      <>
        {statuses.map((status) => (
          <CIMark key={status} repoPath="/repos/app" mark={new CIMarkInfo({ status, pipelineId: 1 })} />
        ))}
        {statuses.map((status) => (
          <ChangeMarker key={status} change={new BranchChange({ number: 1, prefix: '#', state: 'open', ci: status })} />
        ))}
        <PipelineJobList repoPath="/repos/app" pipelineId={1} />
      </>,
    )
    const jobs = await screen.findByRole('list', { name: 'Jobs' })
    for (const status of statuses) {
      expect(screen.getByRole('button', { name: ciLabel(status) }).querySelector('svg')).toHaveClass(
        `lucide-${icons[status]}`,
        `ci-icon-${status}`,
      )
      expect(screen.getByRole('button', { name: `#1 · open · ${ciLabel(status)}` }).querySelector('svg')).toHaveClass(
        `lucide-${icons[status]}`,
      )
      expect(within(jobs).getByText(`job-${status}`).parentElement!.querySelector('svg')).toHaveClass(
        `lucide-${icons[status]}`,
      )
    }
    expect(new Set(statuses.map((s) => icons[s])).size).toBe(statuses.length)
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

  it('fetches a failed job’s log only when the job is clicked, then the full log on request', async () => {
    vi.mocked(HostService.PipelineJobs).mockResolvedValue([
      new Job({
        id: 5,
        name: 'test',
        stage: 'unit',
        status: CIStatus.CIFailed,
        webUrl: 'https://gitlab.example.com/j/5',
      }),
    ])
    vi.mocked(HostService.JobLog)
      .mockResolvedValueOnce({ text: 'line 49\nERROR: Job failed', truncated: true, full: false })
      .mockResolvedValueOnce({ text: 'line 1\n…\nERROR: Job failed', truncated: false, full: true })
    render(<PipelineJobList repoPath="/repos/app" pipelineId={12} />)

    await screen.findByRole('list', { name: 'Jobs' })
    expect(HostService.JobLog).not.toHaveBeenCalled()

    await userEvent.click(screen.getByRole('button', { name: /^test/ }))
    const log = await screen.findByLabelText('Log of test')
    expect(await within(log).findByText(/ERROR: Job failed/)).toBeInTheDocument()
    expect(HostService.JobLog).toHaveBeenCalledWith('/repos/app', 5, false)

    await userEvent.click(within(log).getByRole('button', { name: 'Show full log' }))
    expect(await within(log).findByText(/line 1/)).toBeInTheDocument()
    expect(HostService.JobLog).toHaveBeenLastCalledWith('/repos/app', 5, true)
  })

  it('re-runs a failed job or every failed job on a click, then shows the jobs after it', async () => {
    const failed = new Job({ id: 5, name: 'test', stage: 'unit', status: CIStatus.CIFailed, retryable: true })
    const passed = new Job({ id: 6, name: 'build', stage: 'compile', status: CIStatus.CISuccess })
    vi.mocked(HostService.PipelineJobs).mockResolvedValue([failed, passed])
    vi.mocked(HostService.JobLog).mockResolvedValue({ text: 'boom', truncated: false, full: false } as never)
    const queued = new Job({ id: 7, name: 'test', stage: 'unit', status: CIStatus.CIPending })
    vi.mocked(HostService.RetryJob).mockResolvedValue([failed, passed, queued])
    vi.mocked(HostService.RetryPipeline).mockResolvedValue([passed, queued])
    render(<PipelineJobList repoPath="/repos/app" pipelineId={12} />)

    // A failed job's actions are on its row, not under its log.
    await screen.findByRole('list', { name: 'Jobs' })
    expect(screen.getByRole('button', { name: 'Re-run test' })).toHaveAttribute('title', 'Re-run job')
    expect(screen.getByRole('button', { name: 'Open test on host' })).toBeInTheDocument()
    await userEvent.click(screen.getByRole('button', { name: 'Re-run test' }))
    expect(HostService.RetryJob).toHaveBeenCalledWith('/repos/app', 12, 5)
    expect(await screen.findAllByRole('listitem')).toHaveLength(3)

    await userEvent.click(screen.getByRole('button', { name: 'Re-run failed jobs' }))
    expect(HostService.RetryPipeline).toHaveBeenCalledWith('/repos/app', 12)
    await vi.waitFor(() => expect(screen.getAllByRole('listitem')).toHaveLength(2))
    expect(screen.queryByRole('button', { name: 'Re-run failed jobs' })).not.toBeInTheDocument()
  })

  it('lists jobs at a glance from the mark, and opens a failed one in the commit’s details', async () => {
    vi.mocked(HostService.PipelineJobs).mockResolvedValue([
      new Job({ id: 5, name: 'test', stage: 'unit', status: CIStatus.CIFailed, retryable: true }),
      new Job({ id: 6, name: 'build', stage: 'compile', status: CIStatus.CISuccess }),
    ])
    const onOpenJobs = vi.fn()
    render(
      <CIMark
        repoPath="/repos/app"
        mark={new CIMarkInfo({ status: 'failed', pipelineId: 12, webUrl: 'https://gitlab.example.com/p/12' })}
        onOpenJobs={onOpenJobs}
      />,
    )
    await userEvent.click(screen.getByRole('button', { name: 'CI failed' }))
    const dialog = await screen.findByRole('dialog', { name: 'CI failed' })
    await within(dialog).findByRole('list', { name: 'Jobs' })
    // One row of actions, under the list; none on the jobs themselves.
    expect(
      within(dialog)
        .getAllByRole('button', { name: /Re-run|Open on host/ })
        .map((b) => b.textContent),
    ).toEqual(['Re-run failed jobs', 'Open on host'])
    await userEvent.click(within(dialog).getByRole('button', { name: /^test/ }))
    expect(onOpenJobs).toHaveBeenCalled()
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
    expect(HostService.JobLog).not.toHaveBeenCalled()
  })

  it('offers no re-run where nothing can be re-run', async () => {
    vi.mocked(HostService.PipelineJobs).mockResolvedValue([
      new Job({ id: 7, name: 'ci/circleci', stage: 'status', status: CIStatus.CIFailed }),
    ])
    render(<PipelineJobList repoPath="/repos/app" pipelineId={12} />)
    await screen.findByRole('list', { name: 'Jobs' })
    expect(screen.queryByRole('button', { name: 'Re-run failed jobs' })).not.toBeInTheDocument()
  })
})
