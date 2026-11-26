import { screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { BranchService, RemoteService, StatusService, type BranchStatusInfo } from '@current-client-bindings/app'
import BranchSwitcher from './BranchSwitcher'
import { renderWithDialogs } from '../../test/render'
import { REPO, branchInfo } from '../../test/factories'

describe('BranchSwitcher', () => {
  it('checks out a remote branch from a remote whose name contains a slash', async () => {
    vi.mocked(BranchService.ListLocal).mockResolvedValue([branchInfo({ name: 'main', current: true })])
    vi.mocked(BranchService.ListRemote).mockResolvedValue(['team/origin/fix/login'])
    vi.mocked(BranchService.CurrentBranchStatus).mockResolvedValue({ current: 'main', ahead: 0 } as BranchStatusInfo)
    vi.mocked(StatusService.GetStatus).mockResolvedValue([])
    vi.mocked(RemoteService.List).mockResolvedValue([{ name: 'team/origin', fetchUrl: '', pushUrl: '' }])
    vi.mocked(BranchService.CheckoutBranch).mockResolvedValue()
    renderWithDialogs(<BranchSwitcher repoPath={REPO} />)

    await userEvent.click(await screen.findByRole('button', { name: /team\/origin\/fix\/login/ }))

    expect(BranchService.CheckoutBranch).toHaveBeenCalledWith(REPO, 'fix/login')
  })
})
