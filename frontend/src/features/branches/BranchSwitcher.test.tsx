import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { BranchService, RemoteService, StatusService, type BranchStatusInfo } from '@current-client-bindings/app'
import { DialogProvider } from '../../components/chrome/DialogProvider'
import BranchSwitcher from './BranchSwitcher'

const REPO = '/repos/app'

describe('BranchSwitcher', () => {
  it('checks out a remote branch from a remote whose name contains a slash', async () => {
    vi.mocked(BranchService.ListLocal).mockResolvedValue([
      { name: 'main', current: true, upstream: '', ahead: 0, behind: 0, lastCommitDate: '', worktreePath: '' },
    ])
    vi.mocked(BranchService.ListRemote).mockResolvedValue(['team/origin/fix/login'])
    vi.mocked(BranchService.CurrentBranchStatus).mockResolvedValue({ current: 'main', ahead: 0 } as BranchStatusInfo)
    vi.mocked(StatusService.GetStatus).mockResolvedValue([])
    vi.mocked(RemoteService.List).mockResolvedValue([{ name: 'team/origin', fetchUrl: '', pushUrl: '' }])
    vi.mocked(BranchService.CheckoutBranch).mockResolvedValue()
    render(
      <DialogProvider>
        <BranchSwitcher repoPath={REPO} />
      </DialogProvider>,
    )

    await userEvent.click(await screen.findByRole('button', { name: /team\/origin\/fix\/login/ }))

    expect(BranchService.CheckoutBranch).toHaveBeenCalledWith(REPO, 'fix/login')
  })
})
