import { render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { ChangelogService, HostService } from '@current-client-bindings/app'
import ReleaseReadinessDialog from './ReleaseReadinessDialog'

const result = (over = {}) => ({
  ref: 'HEAD',
  sha: 'abc1234def',
  version: '0.5.0',
  previous: 'v0.4.0',
  target: 'main',
  checks: [
    { label: 'Working tree', level: 'ok', value: 'Clean', evidence: [] },
    { label: 'build/config.yml', level: 'warn', value: 'No 0.5.0', evidence: ['line 3: still 0.4.0'] },
    { label: 'CI on the candidate', level: 'unknown', value: 'Host integration is off', evidence: [] },
  ],
  ...over,
})

describe('ReleaseReadinessDialog', () => {
  it('lists each check with its evidence, and checks again with new inputs', async () => {
    vi.mocked(ChangelogService.ReleaseReadiness).mockResolvedValue(result())
    vi.mocked(ChangelogService.VersionFiles).mockResolvedValue(['build/config.yml'])
    render(<ReleaseReadinessDialog repoPath="/r/app" onClose={vi.fn()} />)

    const checks = await screen.findByRole('list', { name: 'Checks' })
    expect(ChangelogService.ReleaseReadiness).toHaveBeenCalledWith('/r/app', '', '', '')
    const config = within(checks).getByText('build/config.yml').closest('li')!
    expect(config).toHaveTextContent('No 0.5.0')
    expect(config).toHaveTextContent('line 3: still 0.4.0')
    expect(within(config).getByRole('img', { name: 'Needs attention' })).toBeInTheDocument()
    expect(screen.getByText(/Against v0\.4\.0 · abc1234/)).toBeInTheDocument()

    const version = screen.getByRole('textbox', { name: 'Version' })
    await userEvent.clear(version)
    await userEvent.type(version, '1.0.0')
    await userEvent.click(screen.getByRole('button', { name: 'Check' }))
    expect(ChangelogService.ReleaseReadiness).toHaveBeenLastCalledWith('/r/app', 'HEAD', '1.0.0', 'main')
    expect(HostService.Refresh).not.toHaveBeenCalled()
  })

  it('saves the version files and checks again', async () => {
    vi.mocked(ChangelogService.ReleaseReadiness).mockResolvedValue(result())
    vi.mocked(ChangelogService.VersionFiles).mockResolvedValue([])
    vi.mocked(ChangelogService.SetVersionFiles).mockResolvedValue()
    render(<ReleaseReadinessDialog repoPath="/r/app" onClose={vi.fn()} />)
    await screen.findByRole('list', { name: 'Checks' })
    await userEvent.click(screen.getByText('Version files'))
    await userEvent.type(
      screen.getByRole('textbox', { name: /One path per line/ }),
      'build/config.yml{enter}package.json',
    )
    await userEvent.click(screen.getByRole('button', { name: 'Save and check' }))
    expect(ChangelogService.SetVersionFiles).toHaveBeenCalledWith('/r/app', ['build/config.yml', 'package.json'])
    await vi.waitFor(() => expect(ChangelogService.ReleaseReadiness).toHaveBeenCalledTimes(2))
  })
})
