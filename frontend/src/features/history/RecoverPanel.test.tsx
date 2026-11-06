import { render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { BranchService, RecoverService, type SnapshotInfo } from '@current-client-bindings/app'
import { DialogProvider } from '../../components/chrome/DialogProvider'
import RecoverPanel from './RecoverPanel'

const tree: SnapshotInfo = {
  id: '2',
  kind: 'tree',
  tip: '',
  date: new Date().toISOString(),
  action: 'discard',
  branch: 'main',
  skipped: 1,
  files: ['a.txt', 'src/b.txt'],
}
const tip: SnapshotInfo = {
  ...tree,
  id: '1',
  kind: 'tip',
  tip: 'abc1234def',
  action: 'branch delete',
  branch: 'doomed',
  skipped: 0,
  files: [],
}

function renderPanel(snapshots = [tree, tip]) {
  vi.mocked(RecoverService.List).mockResolvedValue(snapshots)
  render(
    <DialogProvider>
      <RecoverPanel repoPath="/repo" onClose={vi.fn()} />
    </DialogProvider>,
  )
}

describe('RecoverPanel', () => {
  it('lists each snapshot with its time, branch, action and files', async () => {
    const user = userEvent.setup()
    renderPanel()

    expect(await screen.findByText('Before discard')).toBeInTheDocument()
    expect(screen.getByText('Before branch delete')).toBeInTheDocument()
    expect(screen.getByText('1 untracked file over 10 MB left out.')).toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: '2 files' }))
    expect(screen.getByText('src/b.txt')).toBeInTheDocument()
  })

  it('restores a snapshot after asking, and one file straight away', async () => {
    const user = userEvent.setup()
    vi.mocked(RecoverService.Restore).mockResolvedValue()
    renderPanel()

    await user.click(await screen.findByRole('button', { name: 'Restore' }))
    const dialog = await screen.findByRole('alertdialog', { name: 'Restore snapshot' })
    await user.click(within(dialog).getByRole('button', { name: 'Restore' }))
    expect(RecoverService.Restore).toHaveBeenCalledWith('/repo', '2', [])
    expect(await screen.findByText('Restored.')).toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: '2 files' }))
    await user.click(screen.getAllByRole('button', { name: 'Restore one file' })[1])
    expect(RecoverService.Restore).toHaveBeenLastCalledWith('/repo', '2', ['src/b.txt'])
  })

  it('makes a branch from a saved tip', async () => {
    const user = userEvent.setup()
    vi.mocked(BranchService.CreateBranchAt).mockResolvedValue()
    renderPanel()

    await user.click(await screen.findByRole('button', { name: 'Create branch' }))
    const dialog = await screen.findByRole('dialog', { name: 'Recover commits' })
    await user.type(within(dialog).getByRole('textbox'), 'recovered/doomed')
    await user.click(within(dialog).getByRole('button', { name: 'Create branch' }))
    await vi.waitFor(() =>
      expect(BranchService.CreateBranchAt).toHaveBeenCalledWith('/repo', 'recovered/doomed', 'abc1234def'),
    )
  })

  it('says when nothing is saved', async () => {
    renderPanel([])
    expect(await screen.findByText('Nothing saved yet.')).toBeInTheDocument()
  })
})
