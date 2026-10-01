import { render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { DiffService, StatusService, type FileStatus } from '@current-client-bindings/app'
import { DialogProvider } from '../../components/chrome/DialogProvider'
import { isStagedStatus, isUnstagedStatus } from './fileStatus'
import { giveElementsLayout } from '../../test/bindings'
import ChangesView from './ChangesView'

const REPO = '/repos/app'

function file(path: string, overrides: Partial<FileStatus> = {}): FileStatus {
  return {
    path,
    origPath: '',
    indexStatus: '.',
    worktreeStatus: 'M',
    conflicted: false,
    indexAdded: 0,
    indexRemoved: 0,
    indexBinary: false,
    workAdded: 1,
    workRemoved: 1,
    workBinary: false,
    ...overrides,
  }
}

function workingTreeFor(files: FileStatus[]) {
  return {
    files,
    conflicted: files.filter((f) => f.conflicted),
    staged: files.filter((f) => !f.conflicted && isStagedStatus(f.indexStatus)),
    unstaged: files.filter((f) => !f.conflicted && isUnstagedStatus(f.worktreeStatus)),
    error: null,
    loadStatus: vi.fn(),
  }
}

function changesView(workingTree: ReturnType<typeof workingTreeFor>) {
  return (
    <DialogProvider>
      <ChangesView repoPath={REPO} workingTree={workingTree} />
    </DialogProvider>
  )
}

function renderChanges(files: FileStatus[]) {
  const workingTree = workingTreeFor(files)
  render(changesView(workingTree))
  return workingTree
}

beforeEach(() => giveElementsLayout())

describe('ChangesView staging', () => {
  it('stages an unstaged file from its checkbox and refreshes status', async () => {
    vi.mocked(StatusService.StageFile).mockResolvedValue()
    const workingTree = renderChanges([file('src/app.ts')])

    await userEvent.click(screen.getByRole('checkbox', { name: 'Stage src/app.ts' }))

    expect(StatusService.StageFile).toHaveBeenCalledWith(REPO, 'src/app.ts')
    expect(workingTree.loadStatus).toHaveBeenCalled()
  })

  it('unstages a staged file from its checkbox', async () => {
    vi.mocked(StatusService.UnstageFile).mockResolvedValue()
    renderChanges([file('src/app.ts', { indexStatus: 'M', worktreeStatus: '.' })])

    await userEvent.click(screen.getByRole('checkbox', { name: 'Unstage src/app.ts' }))

    expect(StatusService.UnstageFile).toHaveBeenCalledWith(REPO, 'src/app.ts')
  })

  it('stages a single hunk from the open diff', async () => {
    vi.mocked(DiffService.GetWorkingTreeDiff).mockResolvedValue({
      conflicted: false,
      oldPath: 'src/app.ts',
      newPath: 'src/app.ts',
      binary: false,
      tooLarge: false,
      sizeBytes: 0,
      hunks: [{ header: '@@ -1 +1 @@', raw: 'raw-hunk', lines: [] }],
    })
    vi.mocked(DiffService.StageHunk).mockResolvedValue()
    renderChanges([file('src/app.ts')])

    await userEvent.click(screen.getByRole('button', { name: 'src/app.ts' }))
    await userEvent.click(await screen.findByRole('button', { name: 'Stage hunk' }))

    expect(DiffService.GetWorkingTreeDiff).toHaveBeenCalledWith(REPO, 'src/app.ts', false, false)
    expect(DiffService.StageHunk).toHaveBeenCalledWith(REPO, 'src/app.ts', 'raw-hunk')
  })

  it('reloads the open diff when the status refreshes', async () => {
    vi.mocked(DiffService.GetWorkingTreeDiff).mockResolvedValue({
      conflicted: false,
      oldPath: 'src/app.ts',
      newPath: 'src/app.ts',
      binary: false,
      tooLarge: false,
      sizeBytes: 0,
      hunks: [{ header: '@@ -1 +1 @@', raw: 'raw-hunk', lines: [] }],
    })
    const { rerender } = render(changesView(workingTreeFor([file('src/app.ts')])))

    await userEvent.click(screen.getByRole('button', { name: 'src/app.ts' }))
    await screen.findByRole('button', { name: 'Stage hunk' })
    expect(DiffService.GetWorkingTreeDiff).toHaveBeenCalledTimes(1)

    rerender(changesView(workingTreeFor([file('src/app.ts')])))
    expect(DiffService.GetWorkingTreeDiff).toHaveBeenCalledTimes(2)
  })

  it('shows why an action failed', async () => {
    vi.mocked(StatusService.StageFile).mockRejectedValue(new Error('Index is locked.'))
    renderChanges([file('src/app.ts')])

    await userEvent.click(screen.getByRole('checkbox', { name: 'Stage src/app.ts' }))

    expect(await screen.findByText('Could not update changes: Index is locked.')).toBeInTheDocument()
  })
})

describe('ChangesView discard', () => {
  it('discards only after confirming', async () => {
    vi.mocked(StatusService.DiscardFile).mockResolvedValue()
    renderChanges([file('src/app.ts')])

    await userEvent.click(screen.getByRole('button', { name: 'Discard src/app.ts' }))
    const dialog = await screen.findByRole('alertdialog', { name: 'Discard changes' })
    expect(StatusService.DiscardFile).not.toHaveBeenCalled()

    await userEvent.click(within(dialog).getByRole('button', { name: 'Discard' }))
    expect(StatusService.DiscardFile).toHaveBeenCalledWith(REPO, 'src/app.ts')
  })

  it('does nothing when the confirmation is cancelled', async () => {
    renderChanges([file('src/app.ts')])

    await userEvent.click(screen.getByRole('button', { name: 'Discard src/app.ts' }))
    await userEvent.click(await screen.findByRole('button', { name: 'Cancel' }))

    expect(StatusService.DiscardFile).not.toHaveBeenCalled()
  })
})

describe('ChangesView multi-select', () => {
  const files = [file('src/a.ts'), file('src/b.ts'), file('src/c.ts')]

  it('shift-click selects a range for a bulk stage', async () => {
    vi.mocked(StatusService.StageFiles).mockResolvedValue()
    renderChanges(files)
    const user = userEvent.setup()

    await user.click(screen.getByRole('button', { name: 'src/a.ts' }))
    await user.keyboard('{Shift>}')
    await user.click(screen.getByRole('button', { name: 'src/c.ts' }))
    await user.keyboard('{/Shift}')
    await user.pointer({ keys: '[MouseRight]', target: screen.getByRole('button', { name: 'src/b.ts' }) })
    await user.click(screen.getByRole('button', { name: 'Stage 3 files' }))

    expect(StatusService.StageFiles).toHaveBeenCalledWith(REPO, ['src/a.ts', 'src/b.ts', 'src/c.ts'])
  })

  it('ctrl-click adds a single file to the selection', async () => {
    vi.mocked(StatusService.StageFiles).mockResolvedValue()
    renderChanges(files)
    const user = userEvent.setup()

    await user.click(screen.getByRole('button', { name: 'src/a.ts' }))
    await user.keyboard('{Control>}')
    await user.click(screen.getByRole('button', { name: 'src/c.ts' }))
    await user.keyboard('{/Control}')
    await user.pointer({ keys: '[MouseRight]', target: screen.getByRole('button', { name: 'src/a.ts' }) })
    await user.click(screen.getByRole('button', { name: 'Stage 2 files' }))

    expect(StatusService.StageFiles).toHaveBeenCalledWith(REPO, ['src/a.ts', 'src/c.ts'])
  })
})

describe('ChangesView conflicts', () => {
  const conflictedFile = () => file('src/app.ts', { indexStatus: 'U', worktreeStatus: 'U', conflicted: true })

  it('lists a conflicted file once, in its own section', () => {
    renderChanges([conflictedFile(), file('src/other.ts')])

    expect(screen.getByText('Conflicted')).toBeInTheDocument()
    expect(screen.getAllByRole('button', { name: 'src/app.ts' })).toHaveLength(1)
    expect(screen.getByRole('checkbox', { name: 'Mark src/app.ts resolved' })).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Discard src/app.ts' })).not.toBeInTheDocument()
  })

  it('marks a conflicted file resolved by staging it', async () => {
    vi.mocked(StatusService.StageFile).mockResolvedValue()
    renderChanges([conflictedFile()])

    await userEvent.click(screen.getByRole('checkbox', { name: 'Mark src/app.ts resolved' }))

    expect(StatusService.StageFile).toHaveBeenCalledWith(REPO, 'src/app.ts')
  })

  it('has no conflicted section when nothing is conflicted', () => {
    renderChanges([file('src/app.ts')])

    expect(screen.queryByText('Conflicted')).not.toBeInTheDocument()
  })
})
