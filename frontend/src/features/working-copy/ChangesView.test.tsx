import { fireEvent, render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import {
  DiffService,
  FileContent,
  FileDiff,
  StashService,
  StatusService,
  type FileStatus,
} from '@current-client-bindings/app'
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
    submodule: null,
    lfs: false,
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

  it('stages just the selected lines of a hunk', async () => {
    vi.mocked(DiffService.GetWorkingTreeDiff).mockResolvedValue({
      conflicted: false,
      oldPath: 'src/app.ts',
      newPath: 'src/app.ts',
      binary: false,
      tooLarge: false,
      sizeBytes: 0,
      hunks: [
        {
          header: '@@ -1,1 +1,2 @@',
          raw: 'raw-hunk',
          lines: [
            { kind: 'added', oldLine: 0, newLine: 1, content: 'keep', moved: false },
            { kind: 'added', oldLine: 0, newLine: 2, content: 'later', moved: false },
          ],
        },
      ],
    })
    vi.mocked(DiffService.StageLines).mockResolvedValue()
    renderChanges([file('src/app.ts')])

    await userEvent.click(screen.getByRole('button', { name: 'src/app.ts' }))
    await userEvent.click(await screen.findByRole('button', { name: 'Select added line 1' }))
    await userEvent.click(screen.getByRole('button', { name: 'Stage 1 line' }))

    expect(DiffService.StageLines).toHaveBeenCalledWith(REPO, 'src/app.ts', 'raw-hunk', { added: [1], removed: [] })
    expect(DiffService.StageHunk).not.toHaveBeenCalled()
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

  it('shows a changed image from the index and the working tree, and reloads it with the status', async () => {
    // A new diff each time, as from the backend; the images reload when it changes.
    vi.mocked(DiffService.GetWorkingTreeDiff).mockImplementation(
      () => Promise.resolve(new FileDiff({ binary: true })) as never,
    )
    vi.mocked(DiffService.GetFileContent).mockResolvedValue(new FileContent({ found: true, data: btoa('png') }))
    const { rerender } = render(changesView(workingTreeFor([file('logo.png', { workBinary: true })])))

    await userEvent.click(screen.getByRole('button', { name: 'logo.png' }))

    expect(await screen.findByRole('img', { name: 'After' })).toBeInTheDocument()
    expect(DiffService.GetFileContent).toHaveBeenCalledWith(REPO, 'logo.png', { kind: 'index', rev: '' })
    expect(DiffService.GetFileContent).toHaveBeenCalledWith(REPO, 'logo.png', { kind: 'worktree', rev: '' })
    expect(DiffService.GetFileContent).toHaveBeenCalledTimes(2)

    rerender(changesView(workingTreeFor([file('logo.png', { workBinary: true })])))
    await vi.waitFor(() => expect(DiffService.GetFileContent).toHaveBeenCalledTimes(4))
  })

  it('shows why an action failed', async () => {
    vi.mocked(StatusService.StageFile).mockRejectedValue(new Error('Index is locked.'))
    renderChanges([file('src/app.ts')])

    await userEvent.click(screen.getByRole('checkbox', { name: 'Stage src/app.ts' }))

    expect(await screen.findByText('Could not update changes: Index is locked.')).toBeInTheDocument()
  })
})

describe('ChangesView untracked files', () => {
  it('shows a new file as added, without hunk actions', async () => {
    vi.mocked(DiffService.GetWorkingTreeDiff).mockResolvedValue(
      new FileDiff({
        newPath: 'notes.md',
        hunks: [
          {
            header: '@@ -0,0 +1 @@',
            raw: 'raw',
            lines: [{ kind: 'added', oldLine: 0, newLine: 1, content: 'hello', moved: false }],
          },
        ],
      }),
    )
    renderChanges([file('notes.md', { worktreeStatus: '?', workAdded: 0, workRemoved: 0 })])

    await userEvent.click(screen.getByRole('button', { name: 'notes.md' }))

    expect(await screen.findByText('Untracked')).toBeInTheDocument()
    expect(DiffService.GetWorkingTreeDiff).toHaveBeenCalledWith(REPO, 'notes.md', false, false)
    expect(screen.queryByRole('button', { name: 'Stage hunk' })).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Discard' })).not.toBeInTheDocument()
  })

  it('names a folder when deleting a nested repository', async () => {
    renderChanges([file('vendor/tool/', { indexStatus: '?', worktreeStatus: '?' })])

    await userEvent.click(screen.getByRole('button', { name: 'Discard vendor/tool/' }))

    expect(await screen.findByRole('alertdialog', { name: 'Delete folder' })).toHaveTextContent(
      'Delete vendor/tool/ and everything in it?',
    )
  })
})

describe('ChangesView file menu', () => {
  it("opens from a file row's More actions button", async () => {
    vi.mocked(StatusService.StageFile).mockResolvedValue()
    renderChanges([file('src/app.ts')])
    const user = userEvent.setup()

    await user.click(screen.getByRole('button', { name: 'More actions for src/app.ts' }))
    await user.click(await screen.findByRole('button', { name: 'Stage' }))

    expect(StatusService.StageFile).toHaveBeenCalledWith(REPO, 'src/app.ts')
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

  it("toggles once on macOS's ctrl-click, whatever events WebKit sends for it", async () => {
    vi.mocked(StatusService.StageFiles).mockResolvedValue()
    renderChanges(files)
    const user = userEvent.setup()

    await user.click(screen.getByRole('button', { name: 'src/a.ts' }))
    const c = screen.getByRole('button', { name: 'src/c.ts' })
    fireEvent.mouseDown(c, { ctrlKey: true, button: 0 })
    fireEvent.contextMenu(c, { ctrlKey: true, button: 0 })
    fireEvent.click(c, { ctrlKey: true, button: 0 })
    expect(screen.queryByRole('button', { name: 'Stage' })).not.toBeInTheDocument()

    await user.pointer({ keys: '[MouseRight]', target: screen.getByRole('button', { name: 'src/a.ts' }) })
    await user.click(screen.getByRole('button', { name: 'Stage 2 files' }))

    expect(StatusService.StageFiles).toHaveBeenCalledWith(REPO, ['src/a.ts', 'src/c.ts'])
  })
})

describe('ChangesView stashing', () => {
  function renderWithBranchChanged(files: FileStatus[]) {
    const onBranchChanged = vi.fn()
    render(
      <DialogProvider>
        <ChangesView repoPath={REPO} workingTree={workingTreeFor(files)} onBranchChanged={onBranchChanged} />
      </DialogProvider>,
    )
    return onBranchChanged
  }

  it('shift-click selects across staged and unstaged files to stash them together', async () => {
    vi.mocked(StashService.StashSave).mockResolvedValue()
    renderWithBranchChanged([
      file('staged.txt', { indexStatus: 'A', worktreeStatus: '.' }),
      file('notes.md'),
      file('untracked.txt', { worktreeStatus: '?' }),
    ])
    const user = userEvent.setup()

    await user.click(screen.getByRole('button', { name: 'staged.txt' }))
    await user.keyboard('{Shift>}')
    await user.click(screen.getByRole('button', { name: 'untracked.txt' }))
    await user.keyboard('{/Shift}')
    await user.pointer({ keys: '[MouseRight]', target: screen.getByRole('button', { name: 'notes.md' }) })

    expect(screen.queryByRole('button', { name: /^(Stage|Unstage) 3 files$/ })).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Discard' })).not.toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: 'Stash 3 files' }))

    expect(StashService.StashSave).toHaveBeenCalledWith(REPO, {
      message: '',
      includeUntracked: true,
      keepIndex: false,
      paths: ['staged.txt', 'notes.md', 'untracked.txt'],
    })
  })

  it('stashes just the selected files, including untracked ones', async () => {
    vi.mocked(StashService.StashSave).mockResolvedValue()
    const onBranchChanged = renderWithBranchChanged([
      file('src/a.ts'),
      file('src/b.ts'),
      file('notes.md', { worktreeStatus: '?' }),
    ])
    const user = userEvent.setup()

    await user.click(screen.getByRole('button', { name: 'src/a.ts' }))
    await user.keyboard('{Control>}')
    await user.click(screen.getByRole('button', { name: 'notes.md' }))
    await user.keyboard('{/Control}')
    await user.pointer({ keys: '[MouseRight]', target: screen.getByRole('button', { name: 'src/a.ts' }) })
    await user.click(screen.getByRole('button', { name: 'Stash 2 files' }))

    expect(StashService.StashSave).toHaveBeenCalledWith(REPO, {
      message: '',
      includeUntracked: true,
      keepIndex: false,
      paths: expect.arrayContaining(['src/a.ts', 'notes.md']),
    })
    await vi.waitFor(() => expect(onBranchChanged).toHaveBeenCalled())
  })

  it('stashes one tracked file without untracked files', async () => {
    vi.mocked(StashService.StashSave).mockResolvedValue()
    renderWithBranchChanged([file('src/a.ts'), file('src/b.ts')])
    const user = userEvent.setup()

    await user.pointer({ keys: '[MouseRight]', target: screen.getByRole('button', { name: 'src/b.ts' }) })
    await user.click(screen.getByRole('button', { name: 'Stash' }))

    expect(StashService.StashSave).toHaveBeenCalledWith(REPO, {
      message: '',
      includeUntracked: false,
      keepIndex: false,
      paths: ['src/b.ts'],
    })
  })
})

describe('ChangesView submodules and Git LFS', () => {
  it('labels a submodule entry', () => {
    renderChanges([
      file('vendor/lib', { submodule: { commitChanged: true, modified: false, untracked: false } }),
      file('src/app.ts'),
    ])

    expect(screen.getAllByText('submodule')).toHaveLength(1)
  })

  it('labels a file stored in Git LFS', () => {
    renderChanges([file('art/cover.psd', { lfs: true }), file('src/app.ts')])

    expect(screen.getAllByText('LFS')).toHaveLength(1)
  })
})

describe('ChangesView expanded diff', () => {
  const textDiff = new FileDiff({ hunks: [{ header: '@@ -1 +1 @@', raw: 'raw', lines: [] }] })

  function renderExpandable(diffExpanded: boolean) {
    vi.mocked(DiffService.GetWorkingTreeDiff).mockResolvedValue(textDiff)
    const onDiffExpandedChange = vi.fn()
    render(
      <DialogProvider>
        <ChangesView
          repoPath={REPO}
          workingTree={workingTreeFor([file('src/app.ts')])}
          diffExpanded={diffExpanded}
          onDiffExpandedChange={onDiffExpandedChange}
        />
      </DialogProvider>,
    )
    return onDiffExpandedChange
  }

  it('expands the diff from its header', async () => {
    const onDiffExpandedChange = renderExpandable(false)

    await userEvent.click(screen.getByRole('button', { name: 'src/app.ts' }))
    await userEvent.click(await screen.findByRole('button', { name: 'Expand diff' }))

    expect(onDiffExpandedChange).toHaveBeenCalledWith(true)
  })

  it('hides the file list while expanded, and brings it back from the header', async () => {
    const onDiffExpandedChange = renderExpandable(true)
    // Nothing open yet, so the file list shows.
    expect(screen.getByRole('button', { name: 'src/app.ts' })).toBeVisible()

    await userEvent.click(screen.getByRole('button', { name: 'src/app.ts' }))

    const show = await screen.findByRole('button', { name: 'Show file list' })
    expect(screen.queryByRole('button', { name: 'src/app.ts' })).not.toBeInTheDocument()
    expect(screen.queryByRole('separator', { name: 'Resize file list' })).not.toBeInTheDocument()
    await userEvent.click(show)
    expect(onDiffExpandedChange).toHaveBeenCalledWith(false)
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
