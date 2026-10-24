import { render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { BranchService, MergedBranches, RemoteService } from '@current-client-bindings/app'
import { DialogProvider } from '../../components/chrome/DialogProvider'
import CleanUpBranchesDialog from './CleanUpBranchesDialog'

const REPO = '/repos/app'

const report = new MergedBranches({
  default: 'main',
  base: 'origin/main',
  branches: [
    {
      name: 'feat/login',
      current: false,
      upstream: 'origin/feat/login',
      tip: 'aaa',
      newer: false,
      number: 0,
      prefix: '',
    },
    { name: 'fix/typo', current: true, upstream: 'origin/fix/typo', tip: 'bbb', newer: false, number: 12, prefix: '!' },
    { name: 'spike', current: false, upstream: 'origin/spike', tip: 'ccc', newer: true, number: 0, prefix: '' },
  ],
})

function renderDialog(onDeleted = vi.fn()) {
  render(
    <DialogProvider>
      <CleanUpBranchesDialog repoPath={REPO} onDeleted={onDeleted} onClose={vi.fn()} />
    </DialogProvider>,
  )
  return onDeleted
}

describe('CleanUpBranchesDialog', () => {
  it('lists merged branches with why, leaves one with newer commits unselected, and deletes the rest', async () => {
    vi.mocked(BranchService.MergedBranches).mockResolvedValue(report)
    vi.mocked(BranchService.CheckoutBranch).mockResolvedValue()
    vi.mocked(RemoteService.Pull).mockResolvedValue()
    vi.mocked(BranchService.DeleteBranch).mockImplementation(
      (_repo, name) => Promise.resolve(`tip-of-${name}`) as never,
    )
    const onDeleted = renderDialog()

    const dialog = await screen.findByRole('dialog', { name: 'Clean up merged branches' })
    expect(await within(dialog).findAllByText('Merged into origin/main')).toHaveLength(2)
    expect(within(dialog).getByText('!12 merged on the host')).toBeInTheDocument()
    expect(within(dialog).getByText('Newer commits')).toBeInTheDocument()
    expect(within(dialog).getByRole('checkbox', { name: 'feat/login' })).toBeChecked()
    expect(within(dialog).getByRole('checkbox', { name: 'spike' })).not.toBeChecked()
    expect(within(dialog).getByText(/Switches to main and pulls first/)).toBeInTheDocument()

    await userEvent.click(within(dialog).getByRole('button', { name: 'Delete 2 branches' }))

    expect(BranchService.CheckoutBranch).toHaveBeenCalledWith(REPO, 'main')
    expect(RemoteService.Pull).toHaveBeenCalledWith(REPO, null)
    expect(vi.mocked(BranchService.DeleteBranch).mock.calls).toEqual([
      [REPO, 'feat/login', true],
      [REPO, 'fix/typo', true],
    ])
    expect(onDeleted).toHaveBeenCalledWith([
      { name: 'feat/login', tip: 'tip-of-feat/login' },
      { name: 'fix/typo', tip: 'tip-of-fix/typo' },
    ])
  })

  it('deletes without switching when the current branch is unselected, and reports a failure', async () => {
    vi.mocked(BranchService.MergedBranches).mockResolvedValue(report)
    vi.mocked(BranchService.DeleteBranch).mockRejectedValue(new Error('Could not delete feat/login.'))
    const onDeleted = renderDialog()

    const dialog = await screen.findByRole('dialog', { name: 'Clean up merged branches' })
    await userEvent.click(await within(dialog).findByRole('checkbox', { name: 'fix/typo' }))
    await userEvent.click(within(dialog).getByRole('button', { name: 'Delete branch' }))

    expect(await within(dialog).findByRole('alert')).toHaveTextContent('Could not delete feat/login.')
    expect(BranchService.CheckoutBranch).not.toHaveBeenCalled()
    expect(onDeleted).not.toHaveBeenCalled()
  })

  it('says when there is nothing to clean up', async () => {
    vi.mocked(BranchService.MergedBranches).mockResolvedValue(
      new MergedBranches({ default: 'main', base: 'main', branches: [] }),
    )
    renderDialog()

    expect(await screen.findByText('No merged branches to clean up.')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Delete branch' })).toBeDisabled()
  })
})
