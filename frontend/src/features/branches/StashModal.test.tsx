import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { ChangedFile, DiffService, FileDiff, StashService } from '@current-client-bindings/app'
import { EMPTY_TREE_SHA } from '../diff/diffMapping'
import StashModal from './StashModal'

const REPO = '/repos/app'
const stash = { index: 1, message: 'On main: wip', date: '2026-10-01T10:00:00Z' }

describe('StashModal', () => {
  beforeEach(() => {
    vi.mocked(StashService.GetChangedFiles).mockResolvedValue([
      new ChangedFile({ status: 'M', path: 'tracked.txt', added: 1 }),
      new ChangedFile({ status: '?', path: 'new.txt', added: 3 }),
    ])
    vi.mocked(DiffService.GetRefDiff).mockResolvedValue(new FileDiff({ binary: true }))
  })

  it("lists the stash's files under its message", async () => {
    render(<StashModal repoPath={REPO} stash={stash} onClose={vi.fn()} />)

    expect(screen.getByText('On main: wip')).toBeInTheDocument()
    expect(await screen.findByRole('button', { name: /tracked\.txt/ })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /new\.txt/ })).toBeInTheDocument()
    expect(StashService.GetChangedFiles).toHaveBeenCalledWith(REPO, 1)
  })

  it('diffs a tracked file against the commit the stash was made on', async () => {
    render(<StashModal repoPath={REPO} stash={stash} onClose={vi.fn()} />)

    await userEvent.click(await screen.findByRole('button', { name: /tracked\.txt/ }))

    await screen.findByText('Binary file changed.')
    expect(DiffService.GetRefDiff).toHaveBeenCalledWith(REPO, 'tracked.txt', 'stash@{1}^1', 'stash@{1}', false)
  })

  it('diffs an untracked file from the stash third parent', async () => {
    render(<StashModal repoPath={REPO} stash={stash} onClose={vi.fn()} />)

    await userEvent.click(await screen.findByRole('button', { name: /new\.txt/ }))

    await screen.findByText('Binary file changed.')
    expect(DiffService.GetRefDiff).toHaveBeenCalledWith(REPO, 'new.txt', EMPTY_TREE_SHA, 'stash@{1}^3', false)
  })
})
