import { render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { BranchService, WorktreeService, type BranchInfo, type WorktreeInfo } from '@current-client-bindings/app'
import { DialogProvider } from '../../components/chrome/DialogProvider'
import WorktreesPanel from './WorktreesPanel'

const REPO = '/projects/app'

function worktree(path: string, overrides: Partial<WorktreeInfo> = {}): WorktreeInfo {
  return {
    path,
    branch: '',
    head: 'abc1234def',
    main: false,
    current: false,
    locked: false,
    missing: false,
    ...overrides,
  }
}

function branch(name: string, worktreePath = ''): BranchInfo {
  return { name, current: false, upstream: '', ahead: 0, behind: 0, lastCommitDate: '', worktreePath }
}

function renderPanel(worktrees: WorktreeInfo[]) {
  vi.mocked(WorktreeService.List).mockResolvedValue(worktrees)
  vi.mocked(BranchService.ListLocal).mockResolvedValue([
    branch('main', '/projects/app'),
    branch('fix/login'),
    branch('review', '/projects/app-review'),
  ])
  const onOpenRepository = vi.fn()
  const onWorktreesChanged = vi.fn()
  render(
    <DialogProvider>
      <WorktreesPanel repoPath={REPO} onOpenRepository={onOpenRepository} onWorktreesChanged={onWorktreesChanged} />
    </DialogProvider>,
  )
  return { onOpenRepository, onWorktreesChanged }
}

const listed = [
  worktree('/projects/app', { branch: 'main', main: true, current: true }),
  worktree('/projects/app-review', { branch: 'review' }),
  worktree('/projects/app-old', { missing: true }),
  worktree('/projects/app-usb', { branch: 'usb', locked: true }),
]

describe('WorktreesPanel', () => {
  it('lists worktrees with their branch and state, and opens one as a tab', async () => {
    const { onOpenRepository } = renderPanel(listed)

    const review = (await screen.findByText('app-review')).closest('li')!
    expect(within(review).getByText('review')).toBeInTheDocument()
    expect(within(screen.getByText('app-old').closest('li')!).getByText('detached abc1234')).toBeInTheDocument()
    expect(screen.getByText('missing')).toBeInTheDocument()
    expect(screen.getByText('locked')).toBeInTheDocument()
    // Not the current one or the main one; not a locked one either.
    expect(screen.queryByRole('button', { name: 'Remove worktree app' })).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Remove worktree app-usb' })).not.toBeInTheDocument()

    await userEvent.click(screen.getByRole('button', { name: 'Open worktree app-review' }))
    expect(onOpenRepository).toHaveBeenCalledWith('/projects/app-review')
  })

  it('adds a worktree for a new branch next to the repository, then offers to open it', async () => {
    vi.mocked(WorktreeService.Add).mockResolvedValue()
    const { onOpenRepository, onWorktreesChanged } = renderPanel(listed)

    await userEvent.click(await screen.findByRole('button', { name: 'Add worktree' }))
    const branchSelect = screen.getByRole('combobox', { name: 'Branch for the new worktree' })
    // Branches already checked out somewhere aren't offered.
    expect(within(branchSelect).queryByRole('option', { name: 'review' })).not.toBeInTheDocument()
    await userEvent.type(screen.getByRole('textbox', { name: 'New branch name' }), 'hotfix login')
    expect(screen.getByText('/projects/app-hotfix-login')).toBeInTheDocument()
    await userEvent.click(screen.getAllByRole('button', { name: 'Add worktree' })[1])

    expect(WorktreeService.Add).toHaveBeenCalledWith(REPO, '/projects/app-hotfix-login', 'hotfix-login', true)
    await vi.waitFor(() => expect(onWorktreesChanged).toHaveBeenCalled())
    await userEvent.click(within(screen.getByRole('status')).getByRole('button', { name: 'Open' }))
    expect(onOpenRepository).toHaveBeenCalledWith('/projects/app-hotfix-login')
  })

  it('adds a worktree for an existing branch', async () => {
    vi.mocked(WorktreeService.Add).mockResolvedValue()
    renderPanel(listed)

    await userEvent.click(await screen.findByRole('button', { name: 'Add worktree' }))
    await userEvent.selectOptions(screen.getByRole('combobox', { name: 'Branch for the new worktree' }), 'fix/login')
    await userEvent.click(screen.getAllByRole('button', { name: 'Add worktree' })[1])

    expect(WorktreeService.Add).toHaveBeenCalledWith(REPO, '/projects/app-fix-login', 'fix/login', false)
  })

  it('asks again before removing a worktree that has changes', async () => {
    vi.mocked(WorktreeService.Remove)
      .mockRejectedValueOnce(new Error('This worktree has changes.'))
      .mockResolvedValueOnce()
    const { onWorktreesChanged } = renderPanel(listed)

    await userEvent.click(await screen.findByRole('button', { name: 'Remove worktree app-review' }))
    await userEvent.click(within(await screen.findByRole('alertdialog')).getByRole('button', { name: 'Remove' }))
    const second = await screen.findByRole('alertdialog')
    expect(second).toHaveTextContent('app-review has uncommitted changes')
    await userEvent.click(within(second).getByRole('button', { name: 'Remove' }))

    expect(WorktreeService.Remove).toHaveBeenNthCalledWith(1, REPO, '/projects/app-review', false)
    expect(WorktreeService.Remove).toHaveBeenNthCalledWith(2, REPO, '/projects/app-review', true)
    await vi.waitFor(() => expect(onWorktreesChanged).toHaveBeenCalled())
  })

  it('lists nothing while the repository is its only worktree', async () => {
    renderPanel([worktree('/projects/app', { branch: 'main', main: true, current: true })])

    expect(await screen.findByText('No other worktrees.')).toBeInTheDocument()
    expect(screen.queryByText('current')).not.toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Add worktree' })).toBeInTheDocument()
  })
})
