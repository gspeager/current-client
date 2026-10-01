import { render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { BranchService, OverlapService, type BranchInfo } from '@current-client-bindings/app'
import { DialogProvider } from '../../components/chrome/DialogProvider'
import BranchSidebar from './BranchSidebar'

const REPO = '/repos/app'

function branch(name: string, current = false): BranchInfo {
  return { name, current, upstream: '', ahead: 0, behind: 0, lastCommitDate: '' }
}

function renderSidebar(onBranchChanged = vi.fn()) {
  render(
    <DialogProvider>
      <BranchSidebar repoPath={REPO} dirty={false} refreshKey={0} onBranchChanged={onBranchChanged} />
    </DialogProvider>,
  )
  return onBranchChanged
}

describe('BranchSidebar', () => {
  it('checks out a branch when its row is clicked', async () => {
    vi.mocked(BranchService.ListLocal).mockResolvedValue([branch('main', true), branch('feature/login')])
    vi.mocked(BranchService.CheckoutBranch).mockResolvedValue()
    const onBranchChanged = renderSidebar()

    await userEvent.click(await screen.findByRole('button', { name: /^feature\/login/ }))

    expect(BranchService.CheckoutBranch).toHaveBeenCalledWith(REPO, 'feature/login')
    expect(onBranchChanged).toHaveBeenCalled()
  })

  it('creates a branch with spaces turned into dashes, and shows the name first', async () => {
    vi.mocked(BranchService.ListLocal).mockResolvedValue([branch('main', true)])
    vi.mocked(BranchService.CreateBranch).mockResolvedValue()
    vi.mocked(BranchService.CheckoutBranch).mockResolvedValue()
    renderSidebar()

    await userEvent.click(await screen.findByRole('button', { name: 'New branch' }))
    await userEvent.type(screen.getByRole('textbox', { name: 'new branch name' }), 'fix login bug')
    expect(screen.getByRole('status')).toHaveTextContent('Will be created as fix-login-bug')
    await userEvent.click(screen.getByRole('button', { name: 'Create' }))

    expect(BranchService.CreateBranch).toHaveBeenCalledWith(REPO, 'fix-login-bug')
    expect(BranchService.CheckoutBranch).toHaveBeenCalledWith(REPO, 'fix-login-bug')
  })

  it('ignores a click on the current branch', async () => {
    vi.mocked(BranchService.ListLocal).mockResolvedValue([branch('main', true)])
    renderSidebar()

    await userEvent.click(await screen.findByRole('button', { name: /^main/ }))

    expect(BranchService.CheckoutBranch).not.toHaveBeenCalled()
  })

  it('offers to restore a deleted branch at its old tip', async () => {
    vi.mocked(BranchService.ListLocal).mockResolvedValue([branch('main', true), branch('spike')])
    vi.mocked(BranchService.DeleteBranch).mockResolvedValue('abc1234def')
    vi.mocked(BranchService.CreateBranchAt).mockResolvedValue()
    renderSidebar()

    await userEvent.click(await screen.findByRole('button', { name: 'Delete spike' }))
    const notice = await screen.findByRole('status')
    expect(notice).toHaveTextContent('Deleted spike')
    await userEvent.click(within(notice).getByRole('button', { name: 'Undo' }))

    expect(BranchService.CreateBranchAt).toHaveBeenCalledWith(REPO, 'spike', 'abc1234def')
    expect(screen.queryByRole('status')).not.toBeInTheDocument()
  })

  it('asks before force-deleting a branch with unmerged changes', async () => {
    vi.mocked(BranchService.ListLocal).mockResolvedValue([branch('main', true), branch('spike')])
    vi.mocked(BranchService.DeleteBranch)
      .mockRejectedValueOnce(new Error('This branch has unmerged changes.'))
      .mockResolvedValueOnce('abc1234def')
    renderSidebar()

    await userEvent.click(await screen.findByRole('button', { name: 'Delete spike' }))
    const dialog = await screen.findByRole('alertdialog', { name: 'Delete branch' })
    await userEvent.click(within(dialog).getByRole('button', { name: 'Delete' }))

    expect(BranchService.DeleteBranch).toHaveBeenNthCalledWith(1, REPO, 'spike', false)
    expect(BranchService.DeleteBranch).toHaveBeenNthCalledWith(2, REPO, 'spike', true)
  })

  it('merges the default branch into the current branch from its context menu', async () => {
    vi.mocked(BranchService.ListLocal).mockResolvedValue([branch('main'), branch('feature/login', true)])
    vi.mocked(BranchService.DefaultBranch).mockResolvedValue('main')
    vi.mocked(BranchService.MergeBranch).mockResolvedValue()
    const onBranchChanged = renderSidebar()
    const user = userEvent.setup()

    await user.pointer({ keys: '[MouseRight]', target: await screen.findByRole('button', { name: /^feature\/login/ }) })
    await user.click(await screen.findByRole('button', { name: 'Merge main into current' }))

    expect(BranchService.MergeBranch).toHaveBeenCalledWith(REPO, 'main')
    await vi.waitFor(() => expect(onBranchChanged).toHaveBeenCalled())
  })

  it('does not offer to merge the default branch into itself', async () => {
    vi.mocked(BranchService.ListLocal).mockResolvedValue([branch('main', true), branch('feature/login')])
    vi.mocked(BranchService.DefaultBranch).mockResolvedValue('main')
    renderSidebar()
    const user = userEvent.setup()

    await vi.waitFor(() => expect(BranchService.DefaultBranch).toHaveBeenCalled())
    await user.pointer({ keys: '[MouseRight]', target: await screen.findByRole('button', { name: /^main/ }) })

    expect(screen.getByRole('button', { name: 'Merge into current' })).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Merge main into current' })).not.toBeInTheDocument()
  })

  it('asks before merging a default branch that is behind its upstream', async () => {
    const main = { ...branch('main'), upstream: 'origin/main', behind: 3 }
    vi.mocked(BranchService.ListLocal).mockResolvedValue([main, branch('feature/login', true)])
    vi.mocked(BranchService.DefaultBranch).mockResolvedValue('main')
    vi.mocked(BranchService.MergeBranch).mockResolvedValue()
    renderSidebar()
    const user = userEvent.setup()

    await user.pointer({ keys: '[MouseRight]', target: await screen.findByRole('button', { name: /^feature\/login/ }) })
    await user.click(await screen.findByRole('button', { name: 'Merge main into current' }))
    const dialog = await screen.findByRole('alertdialog', { name: 'Merge main' })
    expect(dialog).toHaveTextContent('main is 3 commits behind origin/main.')
    await user.click(within(dialog).getByRole('button', { name: 'Merge anyway' }))

    expect(BranchService.MergeBranch).toHaveBeenCalledWith(REPO, 'main')
  })

  it('does not merge a stale default branch when the warning is cancelled', async () => {
    const main = { ...branch('main'), upstream: 'origin/main', behind: 1 }
    vi.mocked(BranchService.ListLocal).mockResolvedValue([main, branch('feature/login', true)])
    vi.mocked(BranchService.DefaultBranch).mockResolvedValue('main')
    renderSidebar()
    const user = userEvent.setup()

    await user.pointer({ keys: '[MouseRight]', target: await screen.findByRole('button', { name: /^feature\/login/ }) })
    await user.click(await screen.findByRole('button', { name: 'Merge main into current' }))
    const dialog = await screen.findByRole('alertdialog', { name: 'Merge main' })
    await user.click(within(dialog).getByRole('button', { name: 'Cancel' }))

    expect(BranchService.MergeBranch).not.toHaveBeenCalled()
  })

  it('shows why branches could not load', async () => {
    vi.mocked(BranchService.ListLocal).mockRejectedValue(new Error('Git command failed.'))
    renderSidebar()

    expect(await screen.findByText('Git command failed.')).toBeInTheDocument()
  })
})

describe('BranchSidebar overlap prediction', () => {
  const report = (supported: boolean) => ({
    supported,
    overlaps: [
      { branch: 'spike', remote: false, files: ['src/app.ts', 'README.md'] },
      { branch: 'origin/login', remote: true, files: ['src/app.ts'] },
    ],
  })

  it('marks local and remote branches that would conflict', async () => {
    vi.mocked(BranchService.ListLocal).mockResolvedValue([branch('main', true), branch('spike'), branch('docs')])
    vi.mocked(OverlapService.Predict).mockResolvedValue(report(true))
    renderSidebar()

    const marks = await screen.findAllByRole('img', { name: /^Would conflict with the current branch/ })
    expect(marks.map((m) => m.getAttribute('aria-label'))).toEqual([
      'Would conflict with the current branch: src/app.ts, README.md',
      'Would conflict with the current branch: src/app.ts',
    ])
    expect(screen.getByRole('button', { name: /^spike/ })).toContainElement(marks[0])
    expect(screen.getByText('Remote branches that would conflict').parentElement).toHaveTextContent('origin/login')
  })

  it('shows nothing when Git is too old to predict', async () => {
    vi.mocked(BranchService.ListLocal).mockResolvedValue([branch('main', true), branch('spike')])
    vi.mocked(OverlapService.Predict).mockResolvedValue({ supported: false, overlaps: [] })
    renderSidebar()

    await screen.findByRole('button', { name: /^spike/ })
    expect(screen.queryByRole('img', { name: /Would conflict/ })).not.toBeInTheDocument()
    expect(screen.queryByText('Remote branches that would conflict')).not.toBeInTheDocument()
  })
})
