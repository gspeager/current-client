import { fireEvent, render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import {
  BranchService,
  DiffService,
  FileDiff,
  NotesService,
  ReadThroughFile,
  ReadThroughInfo,
  ReadThroughNote,
  ReadThroughService,
  TagService,
} from '@current-client-bindings/app'
import ReadThroughSheet from './ReadThroughSheet'

const file = (path: string, over: Partial<ReadThroughFile> = {}) =>
  new ReadThroughFile({ path, status: 'M', added: 2, removed: 1, oldBlob: 'o', newBlob: 'n', ...over })

const info = (over: Partial<ReadThroughInfo> = {}) =>
  new ReadThroughInfo({
    key: 'refs/heads/feat',
    name: 'feat',
    path: '/repo',
    target: 'main',
    base: 'base',
    head: 'head',
    files: [
      file('src/a.ts'),
      file('src/a.test.ts', { kind: 'test', status: 'A' }),
      file('go.sum', { kind: 'lockfile', read: true }),
    ],
    ...over,
  })

const diff = new FileDiff({
  hunks: [
    {
      header: '@@ -1,2 +1,2 @@',
      raw: 'raw',
      lines: [
        { kind: 'removed', oldLine: 1, newLine: 0, content: 'old', moved: false, inOurs: false, inTheirs: false },
        { kind: 'added', oldLine: 0, newLine: 1, content: 'new', moved: false, inOurs: false, inTheirs: false },
      ],
    },
  ],
})

const notesChanged = vi.fn()
const closed = vi.fn()

const open = (refreshKey = 0, request = { ref: 'refs/heads/feat' }) => (
  <ReadThroughSheet
    repoPath="/repo"
    opening={request}
    refreshKey={refreshKey}
    onNotesChanged={notesChanged}
    onClose={closed}
  />
)

beforeEach(() => {
  Element.prototype.scrollIntoView = vi.fn()
  vi.mocked(ReadThroughService.Open).mockResolvedValue(info())
  vi.mocked(ReadThroughService.FileDiff).mockResolvedValue(diff)
  vi.mocked(ReadThroughService.Notes).mockResolvedValue([])
  vi.mocked(ReadThroughService.AddNote).mockResolvedValue(undefined)
  vi.mocked(NotesService.EditReview).mockResolvedValue(undefined)
  vi.mocked(ReadThroughService.SetRead).mockResolvedValue(undefined)
  vi.mocked(BranchService.ListLocal).mockResolvedValue([])
  vi.mocked(BranchService.ListRemote).mockResolvedValue(['origin/main'])
  vi.mocked(TagService.ListTags).mockResolvedValue([])
  vi.mocked(DiffService.GetDependencySummary).mockResolvedValue({ available: false, changes: [] })
})

const opened = async () => {
  const view = render(open())
  await screen.findByText('1 of 3 read')
  return view
}

describe('the overview', () => {
  it('totals the change and groups its files, tests beside their code', async () => {
    await opened()
    expect(screen.getByText('3 files')).toBeInTheDocument()
    expect(screen.getByText('+6 -3')).toBeInTheDocument()
    expect(screen.getByText('Added 1 · Modified 2')).toBeInTheDocument()
    expect(screen.getByText('.ts 2 · .sum 1')).toBeInTheDocument()
    expect(screen.getByText('Tests 1 · Lockfiles 1')).toBeInTheDocument()
    const src = within(screen.getByRole('region', { name: 'src' }))
    expect(src.getAllByRole('checkbox').map((c) => c.getAttribute('aria-label'))).toEqual([
      'Read src/a.ts',
      'Read src/a.test.ts',
    ])
  })

  it('says what it leaves out', async () => {
    vi.mocked(ReadThroughService.Open).mockResolvedValue(info({ skipped: 2 }))
    await opened()
    expect(screen.getByText('2 large untracked files, too large to compare')).toBeInTheDocument()
  })

  it('compares with another target when one is chosen', async () => {
    await opened()
    await userEvent.selectOptions(screen.getByRole('combobox', { name: 'Compare with' }), 'origin/main')
    expect(ReadThroughService.Open).toHaveBeenLastCalledWith(
      '/repo',
      expect.objectContaining({ target: 'origin/main' }),
    )
  })

  it('offers uncommitted changes only where the branch is checked out', async () => {
    vi.mocked(ReadThroughService.Open).mockResolvedValue(info({ checkedOut: true, uncommitted: true }))
    await opened()
    await userEvent.click(screen.getByRole('checkbox', { name: 'Uncommitted changes' }))
    expect(ReadThroughService.Open).toHaveBeenLastCalledWith('/repo', expect.objectContaining({ uncommitted: false }))
  })

  it('has no such choice elsewhere', async () => {
    await opened()
    expect(screen.queryByRole('checkbox', { name: 'Uncommitted changes' })).toBeNull()
  })

  it('says when there is nothing to compare with', async () => {
    vi.mocked(ReadThroughService.Open).mockResolvedValue(info({ target: '', files: [] }))
    render(open())
    expect(await screen.findByText('This repository has no default branch to compare with.')).toBeInTheDocument()
  })
})

describe('ticking a file', () => {
  it('marks it at once, with the version that was listed', async () => {
    await opened()
    vi.mocked(ReadThroughService.Open).mockReturnValue(new Promise(() => undefined) as never)
    await userEvent.click(screen.getByRole('checkbox', { name: 'Read src/a.ts' }))
    expect(screen.getByText('2 of 3 read')).toBeInTheDocument()
    const [range, files, read] = vi.mocked(ReadThroughService.SetRead).mock.calls[0]
    expect(range).toMatchObject({ key: 'refs/heads/feat', base: 'base', head: 'head', files: [] })
    expect(files).toMatchObject([{ path: 'src/a.ts', newBlob: 'n' }])
    expect(read).toBe(true)
  })

  it('clears when the file changes', async () => {
    const { rerender } = await opened()
    vi.mocked(ReadThroughService.Open).mockResolvedValue(
      info({ files: [file('src/a.ts'), file('src/a.test.ts'), file('go.sum', { newBlob: 'changed' })] }),
    )
    const request = { ref: 'refs/heads/feat' }
    rerender(open(1, request))
    expect(await screen.findByText('0 of 3 read')).toBeInTheDocument()
    expect(screen.getByRole('checkbox', { name: 'Read go.sum' })).not.toBeChecked()
  })

  it('goes back when it could not be saved', async () => {
    vi.mocked(ReadThroughService.SetRead).mockRejectedValue(new Error('Disk full.'))
    await opened()
    await userEvent.click(screen.getByRole('checkbox', { name: 'Read src/a.ts' }))
    expect(await screen.findByText('Disk full.')).toBeInTheDocument()
    await waitFor(() => expect(screen.getByText('1 of 3 read')).toBeInTheDocument())
  })
})

describe('reading', () => {
  it('opens a file, and the next one not read', async () => {
    await opened()
    await userEvent.click(screen.getByRole('button', { name: /^a\.ts/ }))
    expect(await screen.findByText('new')).toBeInTheDocument()
    expect(ReadThroughService.FileDiff).toHaveBeenCalledWith(
      expect.objectContaining({ base: 'base' }),
      '',
      'src/a.ts',
      false,
      false,
    )
    await userEvent.click(screen.getByRole('button', { name: 'Next unread' }))
    await waitFor(() => expect(vi.mocked(ReadThroughService.FileDiff).mock.lastCall?.[2]).toBe('src/a.test.ts'))
    await userEvent.click(screen.getByRole('button', { name: 'Next unread' }))
    await waitFor(() => expect(vi.mocked(ReadThroughService.FileDiff).mock.lastCall?.[2]).toBe('src/a.ts'))
  })

  it('reads what changed again when the window is focused', async () => {
    await opened()
    vi.mocked(ReadThroughService.Open).mockResolvedValue(info({ files: [file('src/new.ts')] }))
    fireEvent.focus(window)
    expect(await screen.findByRole('checkbox', { name: 'Read src/new.ts' })).toBeInTheDocument()
  })
})

describe('notes', () => {
  const note = (over: Partial<ReadThroughNote>) =>
    new ReadThroughNote({ id: 'n1', path: 'src/a.ts', line: 1, text: 'why?', lineText: 'new', ...over })
  const readA = async () => {
    await opened()
    await userEvent.click(screen.getByRole('button', { name: /^a\.ts/ }))
    await screen.findByText('new')
  }

  it('adds one on a line, on an old line and on the file', async () => {
    await readA()
    await userEvent.click(screen.getByRole('button', { name: 'Add note on line 1' }))
    await userEvent.type(screen.getByRole('textbox', { name: 'Note on line 1' }), 'why?{Meta>}{Enter}{/Meta}')
    expect(vi.mocked(ReadThroughService.AddNote).mock.lastCall?.[1]).toMatchObject({
      path: 'src/a.ts',
      line: 1,
      oldLine: 0,
      text: 'why?',
    })
    await waitFor(() => expect(notesChanged).toHaveBeenCalled())
    await userEvent.click(screen.getByRole('button', { name: 'Add note on old line 1' }))
    await userEvent.type(screen.getByRole('textbox', { name: 'Note on old line 1' }), 'gone{Meta>}{Enter}{/Meta}')
    expect(vi.mocked(ReadThroughService.AddNote).mock.lastCall?.[1]).toMatchObject({ line: 0, oldLine: 1 })
    await userEvent.click(screen.getByRole('button', { name: 'Add note' }))
    await userEvent.type(screen.getByRole('textbox', { name: 'Note on src/a.ts' }), 'whole{Meta>}{Enter}{/Meta}')
    expect(vi.mocked(ReadThroughService.AddNote).mock.lastCall?.[1]).toMatchObject({ line: 0, oldLine: 0 })
  })

  it('shows one under its line, to edit and remove', async () => {
    vi.mocked(ReadThroughService.Notes).mockResolvedValue([note({})])
    await readA()
    const block = screen.getByRole('group', { name: 'Note on line 1' })
    expect(block.closest('.diff-pane')).not.toBeNull()
    await userEvent.click(within(block).getByRole('button', { name: 'Edit note' }))
    const text = within(block).getByRole('textbox')
    expect(text).toHaveValue('why?')
    await userEvent.type(text, ' really{Meta>}{Enter}{/Meta}')
    expect(NotesService.EditReview).toHaveBeenLastCalledWith('/repo', 'n1', 'why? really')
    await userEvent.click(within(block).getByRole('button', { name: 'Remove note' }))
    expect(NotesService.EditReview).toHaveBeenLastCalledWith('/repo', 'n1', '')
  })

  it('marks one outdated when its line changes, with the line it was on', async () => {
    vi.mocked(ReadThroughService.Notes).mockResolvedValue([note({ outdated: true, lineText: 'was here' })])
    await readA()
    const block = screen.getByRole('group', { name: 'Note on line 1' })
    expect(block.closest('.diff-pane')).toBeNull()
    expect(within(block).getByText('Outdated')).toBeInTheDocument()
    expect(within(block).getByText('was here')).toBeInTheDocument()
  })

  it('closes its box on Escape, not the sheet, and gives the focus back', async () => {
    await readA()
    const add = screen.getByRole('button', { name: 'Add note' })
    await userEvent.click(add)
    await userEvent.type(screen.getByRole('textbox', { name: 'Note on src/a.ts' }), 'x{Escape}')
    expect(screen.queryByRole('textbox', { name: 'Note on src/a.ts' })).toBeNull()
    expect(closed).not.toHaveBeenCalled()
    expect(add).toHaveFocus()
  })

  it('lists and copies them with their file and line', async () => {
    const writeText = vi.fn(() => Promise.resolve())
    Object.defineProperty(navigator, 'clipboard', { value: { writeText }, configurable: true })
    vi.mocked(ReadThroughService.Notes).mockResolvedValue([
      note({}),
      note({ id: 'n2', line: 0, oldLine: 4, text: 'two\nlines' }),
      note({ id: 'n3', path: 'go.sum', line: 0, text: 'fine' }),
      note({ id: 'n4', line: 9, outdated: true, lineText: 'was', text: 'stale' }),
    ])
    await opened()
    fireEvent.click(await screen.findByRole('button', { name: 'Copy review notes' }))
    expect(writeText).toHaveBeenCalledWith(
      [
        '- `src/a.ts:1` why?',
        '- `src/a.ts:4` (old version) two\n  lines',
        '- `go.sum` fine',
        '- `src/a.ts:9` (line changed since) stale',
      ].join('\n'),
    )
    fireEvent.click(screen.getByRole('button', { name: /src\/a\.ts · line 1/ }))
    expect(await screen.findByText('new')).toBeInTheDocument()
  })
})
