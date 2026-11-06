import { render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { BranchService, type BranchReviewRowInfo } from '@current-client-bindings/app'
import { DialogProvider } from '../../components/chrome/DialogProvider'
import BranchReviewDialog from './BranchReviewDialog'

function row(name: string, overrides: Partial<BranchReviewRowInfo> = {}): BranchReviewRowInfo {
  return {
    name,
    current: false,
    upstream: `origin/${name}`,
    ahead: 0,
    behind: 0,
    aheadBase: 0,
    behindBase: 0,
    merged: false,
    mergedChange: '',
    openChange: '',
    stale: false,
    unpushed: false,
    lastCommitDate: '2026-01-02T00:00:00Z',
    authorName: 'Ada',
    authorEmail: 'ada@example.com',
    author: { initials: 'A', color: '#111111' },
    deletable: true,
    ...overrides,
  }
}

const rows = [
  row('main', { current: true, deletable: false, lastCommitDate: '2026-03-01T00:00:00Z' }),
  row('old', { stale: true, merged: true, lastCommitDate: '2025-01-01T00:00:00Z' }),
  row('local-only', { unpushed: true, upstream: '', aheadBase: 2, lastCommitDate: '2026-02-01T00:00:00Z' }),
  row('in-worktree', { deletable: false, lastCommitDate: '2026-01-15T00:00:00Z' }),
]

function renderDialog(onDeleted = vi.fn()) {
  vi.mocked(BranchService.ReviewBranches).mockResolvedValue({ base: 'origin/main', rows })
  render(
    <DialogProvider>
      <BranchReviewDialog repoPath="/repo" onDeleted={onDeleted} onClose={vi.fn()} />
    </DialogProvider>,
  )
  return onDeleted
}

const shownNames = () =>
  within(screen.getByRole('table'))
    .getAllByRole('row')
    .slice(1)
    .map((tr) => within(tr).getAllByRole('cell')[1].firstChild?.textContent)

describe('BranchReviewDialog', () => {
  it('sorts newest first, and by any column', async () => {
    const user = userEvent.setup()
    renderDialog()
    await screen.findByRole('table')
    expect(shownNames()).toEqual(['main', 'local-only', 'in-worktree', 'old'])

    await user.click(screen.getByRole('button', { name: 'Branch' }))
    expect(shownNames()).toEqual(['in-worktree', 'local-only', 'main', 'old'])

    await user.click(screen.getByRole('button', { name: /^Branch/ }))
    expect(shownNames()).toEqual(['old', 'main', 'local-only', 'in-worktree'])
  })

  it('filters to merged, stale and unpushed', async () => {
    const user = userEvent.setup()
    renderDialog()
    await screen.findByRole('table')

    await user.click(screen.getByRole('button', { name: 'Unpushed' }))
    expect(shownNames()).toEqual(['local-only'])

    await user.click(screen.getByRole('button', { name: 'Stale' }))
    expect(shownNames()).toEqual(['old'])

    await user.click(screen.getAllByRole('button', { name: 'Merged' })[0])
    expect(shownNames()).toEqual(['old'])
  })

  it("won't select the current branch or one in a worktree", async () => {
    renderDialog()
    expect(await screen.findByRole('checkbox', { name: 'main' })).toBeDisabled()
    expect(screen.getByRole('checkbox', { name: 'in-worktree' })).toBeDisabled()
    expect(screen.getByRole('checkbox', { name: 'old' })).toBeEnabled()
  })

  it('deletes merged branches without asking and hands back their tips for undo', async () => {
    const user = userEvent.setup()
    vi.mocked(BranchService.DeleteBranch).mockResolvedValue('abc')
    const onDeleted = renderDialog()

    await user.click(await screen.findByRole('checkbox', { name: 'old' }))
    await user.click(screen.getByRole('button', { name: 'Delete 1 branch' }))

    expect(BranchService.DeleteBranch).toHaveBeenCalledWith('/repo', 'old', true)
    expect(onDeleted).toHaveBeenCalledWith([{ name: 'old', tip: 'abc' }])
  })

  it('asks first before deleting an unmerged branch', async () => {
    const user = userEvent.setup()
    vi.mocked(BranchService.DeleteBranch).mockResolvedValue('def')
    const onDeleted = renderDialog()

    await user.click(await screen.findByRole('checkbox', { name: 'local-only' }))
    await user.click(screen.getByRole('button', { name: 'Delete 1 branch' }))
    expect(BranchService.DeleteBranch).not.toHaveBeenCalled()

    await user.click(
      within(await screen.findByRole('alertdialog', { name: 'Delete unmerged branches' })).getByRole('button', {
        name: 'Delete',
      }),
    )
    expect(onDeleted).toHaveBeenCalledWith([{ name: 'local-only', tip: 'def' }])
  })

  it("selects all shown, skipping what can't be deleted", async () => {
    const user = userEvent.setup()
    renderDialog()
    await user.click(await screen.findByRole('checkbox', { name: 'Select all shown' }))
    expect(screen.getByRole('button', { name: 'Delete 2 branches' })).toBeEnabled()
  })
})
