import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { BranchService, PlatformService } from '@current-client-bindings/app'
import StatusBar from './StatusBar'

describe('StatusBar', () => {
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
