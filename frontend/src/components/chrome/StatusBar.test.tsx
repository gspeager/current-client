import { render, screen } from '@testing-library/react'
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
})
