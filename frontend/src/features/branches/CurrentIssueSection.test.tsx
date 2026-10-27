import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { BranchService, HostService, PlatformService, RemoteService } from '@current-client-bindings/app'
import { Links } from '@current-client-bindings/core/hostlink'
import { IssuePanelContext } from '../../lib/issuePanel'
import CurrentIssueSection from './CurrentIssueSection'

const links = new Links({
  root: 'https://gitlab.example.com/',
  issue: 'https://gitlab.example.com/g/r/-/issues/',
  issuePath: '/-/issues/',
})

describe('CurrentIssueSection', () => {
  it('shows the issue the checked-out branch is named after, with Details and Open on host', async () => {
    vi.mocked(BranchService.CurrentBranchStatus).mockResolvedValue({
      current: '12-fix-login',
      ahead: 0,
      behind: 0,
    } as never)
    vi.mocked(RemoteService.HostLinks).mockResolvedValue(links)
    vi.mocked(HostService.FetchIssueTitles).mockResolvedValue({
      12: { number: 12, title: 'Login fails on Windows', state: 'open', webUrl: '', known: true },
    })
    vi.mocked(PlatformService.OpenWebPage).mockResolvedValue()
    const openIssue = vi.fn()
    render(
      <IssuePanelContext.Provider value={{ open: openIssue }}>
        <CurrentIssueSection repoPath="/repos/app" refreshKey={0} />
      </IssuePanelContext.Provider>,
    )

    const card = await screen.findByRole('region', { name: 'Issue for 12-fix-login' })
    expect(card).toHaveTextContent('#12')
    expect(card).toHaveTextContent('Login fails on Windows')
    expect(card).toHaveTextContent('open')
    await userEvent.click(screen.getByRole('button', { name: 'Details' }))
    expect(openIssue).toHaveBeenCalledWith('12')
    await userEvent.click(screen.getByRole('button', { name: 'Open on host' }))
    expect(PlatformService.OpenWebPage).toHaveBeenCalledWith('https://gitlab.example.com/g/r/-/issues/12')
  })

  it('shows nothing for a branch without an issue, or before the host answered', async () => {
    vi.mocked(BranchService.CurrentBranchStatus).mockResolvedValue({ current: 'main', ahead: 0, behind: 0 } as never)
    vi.mocked(RemoteService.HostLinks).mockResolvedValue(links)
    const { rerender } = render(<CurrentIssueSection repoPath="/repos/app" refreshKey={0} />)
    await new Promise((r) => setTimeout(r, 0))
    expect(screen.queryByRole('region')).not.toBeInTheDocument()

    vi.mocked(BranchService.CurrentBranchStatus).mockResolvedValue({ current: '7-x', ahead: 0, behind: 0 } as never)
    vi.mocked(HostService.FetchIssueTitles).mockResolvedValue({
      7: { number: 7, title: '', state: '', webUrl: '', known: false },
    })
    rerender(<CurrentIssueSection repoPath="/repos/other" refreshKey={0} />)
    await new Promise((r) => setTimeout(r, 300))
    expect(screen.queryByRole('region')).not.toBeInTheDocument()
  })
})
