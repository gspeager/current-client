import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { ChangedFile, DiffService, FileContent, FileDiff, StashService } from '@current-client-bindings/app'
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

  it('shows a changed image from the commit the stash was made on and the stash', async () => {
    vi.mocked(StashService.GetChangedFiles).mockResolvedValue([new ChangedFile({ status: 'M', path: 'logo.png' })])
    vi.mocked(DiffService.GetFileContent).mockResolvedValue(new FileContent({ found: true, data: btoa('png') }))
    render(<StashModal repoPath={REPO} stash={stash} onClose={vi.fn()} />)

    await userEvent.click(await screen.findByRole('button', { name: /logo\.png/ }))

    expect(await screen.findByRole('img', { name: 'After' })).toBeInTheDocument()
    expect(DiffService.GetFileContent).toHaveBeenCalledWith(REPO, 'logo.png', { kind: 'commit', rev: 'stash@{1}^1' })
    expect(DiffService.GetFileContent).toHaveBeenCalledWith(REPO, 'logo.png', { kind: 'commit', rev: 'stash@{1}' })
  })

  it('shows an untracked image as added, from the third parent', async () => {
    vi.mocked(StashService.GetChangedFiles).mockResolvedValue([new ChangedFile({ status: '?', path: 'new.png' })])
    vi.mocked(DiffService.GetFileContent).mockImplementation(
      (_repo, _path, source) =>
        Promise.resolve(new FileContent({ found: source.rev.endsWith('^3'), data: btoa('png') })) as never,
    )
    render(<StashModal repoPath={REPO} stash={stash} onClose={vi.fn()} />)

    await userEvent.click(await screen.findByRole('button', { name: /new\.png/ }))

    expect(await screen.findByRole('img', { name: 'Added' })).toBeInTheDocument()
    expect(DiffService.GetFileContent).toHaveBeenCalledWith(REPO, 'new.png', { kind: 'commit', rev: EMPTY_TREE_SHA })
    expect(DiffService.GetFileContent).toHaveBeenCalledWith(REPO, 'new.png', { kind: 'commit', rev: 'stash@{1}^3' })
  })
})
