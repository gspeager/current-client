import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { BranchService, HostInfo, PlatformService } from '@current-client-bindings/app'
import StatusBar from './StatusBar'

describe('StatusBar', () => {
  it('shows which API host integration is on for, when it was last asked, and opens Settings', async () => {
    vi.mocked(PlatformService.AppVersion).mockResolvedValue('0.1.0')
    vi.mocked(BranchService.CurrentBranchStatus).mockResolvedValue({ current: 'main', upstream: '' } as never)
    const onOpenSettings = vi.fn()
    const integration = {
      host: new HostInfo({
        hostname: 'github.com',
        provider: 'github',
        apiUrl: 'https://api.github.com',
        enabled: true,
      }),
      apiHost: 'api.github.com',
      status: { lastRequest: null, inFlight: false, remaining: 0, limit: 0, resetAt: null },
    }

    render(
      <StatusBar
        repoPath="/repos/app"
        repoVersion={0}
        gitVersion="2.47.0"
        integration={integration}
        onOpenSettings={onOpenSettings}
      />,
    )

    const indicator = await screen.findByRole('button', { name: /api\.github\.com/ })
    expect(indicator).toHaveTextContent('no requests yet')
    await userEvent.click(indicator)
    expect(onOpenSettings).toHaveBeenCalled()
  })

  it('shows the budget the host last reported', async () => {
    vi.mocked(PlatformService.AppVersion).mockResolvedValue('0.1.0')
    vi.mocked(BranchService.CurrentBranchStatus).mockResolvedValue({ current: 'main', upstream: '' } as never)
    const integration = {
      host: new HostInfo({
        hostname: 'github.com',
        provider: 'github',
        apiUrl: 'https://api.github.com',
        enabled: true,
      }),
      apiHost: 'api.github.com',
      status: { lastRequest: new Date().toISOString(), inFlight: false, remaining: 42, limit: 60, resetAt: null },
    }

    render(<StatusBar repoPath="/repos/app" repoVersion={0} gitVersion="2.47.0" integration={integration} />)

    const indicator = await screen.findByRole('button', { name: /api\.github\.com/ })
    expect(indicator).toHaveTextContent('42 of 60 requests left · last request just now')
  })

  it('shows nothing about hosts while integration is off', async () => {
    vi.mocked(PlatformService.AppVersion).mockResolvedValue('0.1.0')
    vi.mocked(BranchService.CurrentBranchStatus).mockResolvedValue({ current: 'main', upstream: '' } as never)

    render(<StatusBar repoPath="/repos/app" repoVersion={0} gitVersion="2.47.0" />)

    expect(await screen.findByText('Current Client 0.1.0')).toBeInTheDocument()
    expect(screen.queryByText(/requests/)).not.toBeInTheDocument()
  })

  it('shows the app version next to the Git version', async () => {
    vi.mocked(PlatformService.AppVersion).mockResolvedValue('0.1.0')
    vi.mocked(BranchService.CurrentBranchStatus).mockResolvedValue({ current: 'main', upstream: '' } as never)

    render(<StatusBar repoPath="/repos/app" repoVersion={0} gitVersion="2.47.0" />)

    expect(await screen.findByText('Current Client 0.1.0')).toBeInTheDocument()
    expect(screen.getByText('git 2.47.0')).toBeInTheDocument()
  })

  it('says where HEAD is when it is detached', async () => {
    vi.mocked(PlatformService.AppVersion).mockResolvedValue('0.1.0')
    vi.mocked(BranchService.CurrentBranchStatus).mockResolvedValue({ current: 'HEAD', detachedAt: 'abc1234' } as never)

    render(<StatusBar repoPath="/repos/app" repoVersion={0} gitVersion="2.47.0" />)

    expect(await screen.findByText('HEAD detached at abc1234')).toBeInTheDocument()
  })

  it('shows the repository path with ~ for home, in full on hover, and copies it', async () => {
    vi.mocked(PlatformService.HomeDir).mockResolvedValue('/Users/sam')
    const writeText = vi.fn().mockResolvedValue(undefined)
    Object.assign(navigator, { clipboard: { writeText } })

    render(<StatusBar repoPath="/Users/sam/projects/acme" repoVersion={0} gitVersion="2.47.0" />)

    const path = await screen.findByRole('button', { name: 'Copy /Users/sam/projects/acme' })
    await vi.waitFor(() => expect(path).toHaveTextContent('~/projects/acme'))
    expect(path).toHaveAttribute('title', '/Users/sam/projects/acme')
    await userEvent.click(path)
    expect(writeText).toHaveBeenCalledWith('/Users/sam/projects/acme')
    expect(await screen.findByText('Copied')).toBeInTheDocument()
  })
})
