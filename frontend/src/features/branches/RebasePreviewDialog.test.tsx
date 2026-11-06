import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { BranchService, type RebasePreviewInfo } from '@current-client-bindings/app'
import RebasePreviewDialog from './RebasePreviewDialog'

const commit = (subject: string, outcome = 'clean', files: string[] = []) => ({
  sha: `${subject}0000000`,
  subject,
  outcome,
  files,
})

function preview(overrides: Partial<RebasePreviewInfo> = {}): RebasePreviewInfo {
  return {
    current: 'topic',
    onto: 'main',
    commits: [commit('one'), commit('two')],
    predicted: true,
    droppedMerges: 0,
    forcePush: false,
    upstream: '',
    dirty: false,
    command: 'git rebase main',
    ...overrides,
  }
}

function renderDialog(info: RebasePreviewInfo) {
  vi.mocked(BranchService.PreviewRebase).mockResolvedValue(info)
  const onRebase = vi.fn()
  render(<RebasePreviewDialog repoPath="/repo" onto="main" onRebase={onRebase} onClose={vi.fn()} />)
  return onRebase
}

describe('RebasePreviewDialog', () => {
  it('lists the commits and the command, and rebases on request', async () => {
    const user = userEvent.setup()
    const onRebase = renderDialog(preview())

    expect(await screen.findByText('2 commits to move, oldest first.')).toBeInTheDocument()
    expect(screen.getByRole('dialog', { name: 'Rebase topic onto main' })).toBeInTheDocument()
    expect(screen.getByText('one')).toBeInTheDocument()
    expect(screen.getByText('git rebase main')).toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: 'Rebase' }))
    expect(onRebase).toHaveBeenCalled()
  })

  it('names a predicted conflict and its files', async () => {
    renderDialog(preview({ commits: [commit('one', 'conflict', ['f.txt']), commit('two', 'unknown')] }))
    expect(await screen.findByText('Conflict')).toBeInTheDocument()
    expect(screen.getByText('f.txt')).toBeInTheDocument()
    expect(screen.getByText(/Stops on the conflict/)).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Rebase' })).toBeEnabled()
  })

  it('says a force push will be needed, with --force-with-lease', async () => {
    renderDialog(preview({ forcePush: true, upstream: 'origin/topic' }))
    expect(await screen.findByText(/origin\/topic has these commits.*--force-with-lease/)).toBeInTheDocument()
  })

  it("says when conflicts can't be predicted", async () => {
    renderDialog(preview({ predicted: false }))
    expect(await screen.findByText(/needs Git 2\.40 or newer/)).toBeInTheDocument()
  })

  it('refuses with changes in the way, and with nothing to do', async () => {
    renderDialog(preview({ dirty: true }))
    expect(await screen.findByText(/Commit or stash/)).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Rebase' })).toBeDisabled()
  })

  it('says when there is nothing to move', async () => {
    renderDialog(preview({ commits: [] }))
    expect(await screen.findByText(/Nothing to move/)).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Rebase' })).toBeDisabled()
  })
})
