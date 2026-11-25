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
  ChangeStructureInfo,
  ReadThroughService,
  RiskReportInfo,
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

const report = (over: Partial<RiskReportInfo> = {}) =>
  new RiskReportInfo({ findings: [], leftOut: {}, commitsNotRead: false, ...over })

const notesChanged = vi.fn()
const closed = vi.fn()

const REQUEST = { ref: 'refs/heads/feat' }

const open = (refreshKey = 0) => (
  <ReadThroughSheet
    repoPath="/repo"
    opening={REQUEST}
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
  vi.mocked(ReadThroughService.Findings).mockResolvedValue(report())
  vi.mocked(ReadThroughService.Structure).mockResolvedValue(new ChangeStructureInfo({ read: true, languages: ['Go'] }))
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

  it('can leave uncommitted changes out when they cannot be read', async () => {
    vi.mocked(ReadThroughService.Open).mockRejectedValueOnce(new Error('too many large untracked files'))
    render(open())
    expect(await screen.findByText('too many large untracked files')).toBeInTheDocument()
    await userEvent.click(screen.getByRole('checkbox', { name: 'Uncommitted changes' }))
    expect(await screen.findByText('1 of 3 read')).toBeInTheDocument()
    expect(ReadThroughService.Open).toHaveBeenLastCalledWith('/repo', expect.objectContaining({ uncommitted: false }))
  })

  it('says when there is nothing to compare with', async () => {
    vi.mocked(ReadThroughService.Open).mockResolvedValue(info({ target: '', files: [] }))
    render(open())
    expect(await screen.findByText('This repository has no default branch to compare with.')).toBeInTheDocument()
  })
})

describe('since last read', () => {
  const again = (over: Partial<ReadThroughFile> = {}) =>
    file('src/a.ts', { again: true, sinceAdded: 1, sinceRemoved: 0, newBlob: 'n2', ...over })
  const mixed = () =>
    info({
      marked: 2,
      files: [
        again(),
        file('src/b.ts', { read: true }),
        file('src/c.ts'),
        file('old.txt', { gone: true, again: true, sinceAdded: 0, sinceRemoved: 4 }),
      ],
    })
  const listedFiles = () => screen.getAllByRole('checkbox', { name: /^Read / }).map((c) => c.getAttribute('aria-label'))
  const inSince = async () => {
    vi.mocked(ReadThroughService.Open).mockResolvedValue(mixed())
    vi.mocked(ReadThroughService.SinceDiff).mockResolvedValue(
      new FileDiff({ hunks: [{ ...diff.hunks[0], lines: [{ ...diff.hunks[0].lines[1], content: 'added since' }] }] }),
    )
    const view = render(open())
    await screen.findByText(/1 of 3 read/)
    await userEvent.click(screen.getByRole('button', { name: 'Since last read' }))
    return view
  }

  it('is offered once something has been read', async () => {
    await opened()
    expect(screen.queryByRole('button', { name: 'Since last read' })).toBeNull()
  })

  it('counts the files changed again, and says so on each', async () => {
    vi.mocked(ReadThroughService.Open).mockResolvedValue(mixed())
    render(open())
    expect(await screen.findByText('1 of 3 read · 2 changed again')).toBeInTheDocument()
    expect(listedFiles()).toEqual(['Read src/a.ts', 'Read src/b.ts', 'Read src/c.ts'])
    expect(screen.getAllByText('again')).toHaveLength(1)
    expect(screen.getByTitle('Changed since it was read: +1 -0.')).toBeInTheDocument()
  })

  it('lists what is unread or changed since, with the lines that changed', async () => {
    await inSince()
    expect(listedFiles()).toEqual(['Read old.txt', 'Read src/a.ts', 'Read src/c.ts'])
    expect(screen.getByText('+3 -5')).toBeInTheDocument()
    await userEvent.click(screen.getByRole('button', { name: /^a\.ts/ }))
    expect(await screen.findByText('added since')).toBeInTheDocument()
    expect(ReadThroughService.SinceDiff).toHaveBeenCalledWith(
      expect.objectContaining({ base: 'base' }),
      'src/a.ts',
      false,
      false,
    )
    expect(screen.getByText('Changed since it was read: +1 -0.')).toBeInTheDocument()
    await userEvent.click(screen.getByRole('button', { name: 'Show the whole change' }))
    await waitFor(() => expect(ReadThroughService.FileDiff).toHaveBeenCalled())
    await userEvent.click(await screen.findByRole('button', { name: 'Show what changed since' }))
    expect(await screen.findByText('added since')).toBeInTheDocument()
  })

  it('shows a file the change no longer holds, and a base that did not merge', async () => {
    await inSince()
    await userEvent.click(screen.getByRole('button', { name: /^old\.txt/ }))
    expect(
      await screen.findByText('Changed since it was read: +0 -4. The change no longer holds it.'),
    ).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Show the whole change' })).toBeNull()
    vi.mocked(ReadThroughService.Open).mockResolvedValue(info({ marked: 1, files: [again({ conflict: true })] }))
    fireEvent.focus(window)
    expect(await screen.findByTitle(/What was read doesn't merge with the new base/)).toBeInTheDocument()
  })

  it('leaves a file in the list and its diff as it is once ticked', async () => {
    await inSince()
    await userEvent.click(screen.getByRole('button', { name: /^a\.ts/ }))
    await screen.findByText('added since')
    vi.mocked(ReadThroughService.Open).mockResolvedValue(
      info({ marked: 2, files: [file('src/a.ts', { newBlob: 'n2', read: true }), file('src/c.ts')] }),
    )
    await userEvent.click(screen.getByRole('checkbox', { name: 'Read' }))
    expect(await screen.findByText('1 of 2 read')).toBeInTheDocument()
    expect(listedFiles()).toEqual(['Read src/a.ts', 'Read src/c.ts'])
    expect(screen.getByText('added since')).toBeInTheDocument()
    expect(ReadThroughService.FileDiff).not.toHaveBeenCalled()
    await userEvent.click(screen.getByRole('checkbox', { name: 'Ignore whitespace', hidden: true }))
    expect(await screen.findByText('new')).toBeInTheDocument()
    expect(ReadThroughService.SinceDiff).toHaveBeenCalledTimes(1)
  })

  it('takes notes on new lines only, as the old side is not the base', async () => {
    vi.mocked(ReadThroughService.Notes).mockResolvedValue([
      new ReadThroughNote({ id: 'n1', path: 'src/a.ts', oldLine: 1, text: 'on the base' }),
    ])
    vi.mocked(ReadThroughService.Open).mockResolvedValue(mixed())
    vi.mocked(ReadThroughService.SinceDiff).mockResolvedValue(diff)
    render(open())
    await userEvent.click(await screen.findByRole('button', { name: 'Since last read' }))
    await userEvent.click(screen.getByRole('button', { name: /^a\.ts/ }))
    expect(await screen.findByRole('button', { name: 'Add note on line 1' })).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Add note on old line 1' })).toBeNull()
    expect(screen.queryByText('on the base')).toBeNull()
    await userEvent.click(screen.getByRole('button', { name: 'Show the whole change' }))
    await userEvent.click(await screen.findByRole('button', { name: 'Add note on old line 1' }))
    expect(screen.getByRole('textbox', { name: 'Note on old line 1' })).toBeInTheDocument()
    await userEvent.click(screen.getByRole('button', { name: 'Show what changed since' }))
    await waitFor(() => expect(ReadThroughService.SinceDiff).toHaveBeenCalledTimes(2))
    expect(screen.queryByRole('textbox', { name: 'Note on old line 1' })).toBeNull()
  })

  it('marks every file as read', async () => {
    await inSince()
    await userEvent.click(screen.getByRole('button', { name: 'Mark all as read' }))
    const [, marked, read] = vi.mocked(ReadThroughService.SetRead).mock.calls[0]
    expect(marked.map((f) => [f.path, f.gone])).toEqual([
      ['src/a.ts', false],
      ['src/c.ts', false],
      ['old.txt', true],
    ])
    expect(read).toBe(true)
    expect(screen.getByRole('button', { name: 'Mark all as read' })).toBeDisabled()
  })

  it('leaves out what was ticked before choosing it', async () => {
    vi.mocked(ReadThroughService.Open).mockResolvedValue(mixed())
    render(open())
    await userEvent.click(await screen.findByRole('checkbox', { name: 'Read src/c.ts' }))
    await userEvent.click(screen.getByRole('button', { name: 'Since last read' }))
    expect(listedFiles()).toEqual(['Read old.txt', 'Read src/a.ts'])
  })

  it('says when there is nothing new', async () => {
    vi.mocked(ReadThroughService.Open).mockResolvedValue(info({ marked: 1, files: [file('src/b.ts', { read: true })] }))
    render(open())
    await userEvent.click(await screen.findByRole('button', { name: 'Since last read' }))
    expect(screen.getByText('Everything is read, and nothing has changed since.')).toBeInTheDocument()
  })
})

describe('worth a second look', () => {
  const finding = (over: object) => ({
    rule: 'ci',
    path: 'src/a.ts',
    line: 0,
    oldLine: 0,
    message: 'CI file changed.',
    ...over,
  })

  it('lists facts by rule, with why, and opens one at its line', async () => {
    vi.mocked(ReadThroughService.Findings).mockResolvedValue(
      report({
        findings: [
          finding({ rule: 'exports', message: 'src now exports a.', line: 1 }),
          finding({ rule: 'fewer-tests', message: '1 test fewer.', oldLine: 1 }),
          finding({
            rule: 'secret',
            path: 'gone.txt',
            message: 'Looks like a key, added in abc1234 and removed since.',
          }),
        ],
        leftOut: { exports: 3 },
        commitsNotRead: true,
      }),
    )
    await opened()
    const section = within(screen.getByRole('region', { name: 'Worth a second look' }))
    await section.findByText('src now exports a.')
    expect(section.getAllByRole('heading', { level: 4 }).map((h) => h.textContent)).toEqual([
      'Secret-looking strings 1Anyone who can read the repository can read it.',
      'Fewer tests 1What they covered may no longer be checked.',
      'What a package exports 4Code that uses the package may need to change.',
    ])
    expect(section.getByText('3 more not listed')).toBeInTheDocument()
    expect(section.getByText(/Only the total change was read for secrets/)).toBeInTheDocument()
    expect(section.getByRole('button', { name: /added in abc1234/ })).toBeDisabled()
    expect(section.getByText('src/a.ts:1 (old version)')).toBeInTheDocument()
    await userEvent.click(section.getByRole('button', { name: /1 test fewer/ }))
    expect(await screen.findByText('old')).toBeInTheDocument()
  })

  it('says when there are none, and what went wrong', async () => {
    const { unmount } = await opened()
    expect(await screen.findByText('None found.')).toBeInTheDocument()
    unmount()
    vi.mocked(ReadThroughService.Findings).mockRejectedValue(new Error('No.'))
    await opened()
    expect(await screen.findByText('No.')).toBeInTheDocument()
  })

  it('looks again only when the change does', async () => {
    const { rerender } = await opened()
    await screen.findByText('None found.')
    rerender(open(1))
    await waitFor(() => expect(ReadThroughService.Open).toHaveBeenCalledTimes(2))
    expect(ReadThroughService.Findings).toHaveBeenCalledTimes(1)
    vi.mocked(ReadThroughService.Open).mockResolvedValue(info({ files: [file('src/a.ts', { newBlob: 'n2' })] }))
    rerender(open(2))
    await waitFor(() => expect(ReadThroughService.Findings).toHaveBeenCalledTimes(2))
    await userEvent.click(screen.getByRole('button', { name: /^a\.ts/ }))
    vi.mocked(ReadThroughService.Open).mockResolvedValue(info({ files: [file('src/a.ts', { newBlob: 'n3' })] }))
    rerender(open(3))
    await waitFor(() => expect(ReadThroughService.Open).toHaveBeenCalledTimes(4))
    expect(ReadThroughService.Findings).toHaveBeenCalledTimes(2)
    await userEvent.click(screen.getByRole('button', { name: 'Overview' }))
    await waitFor(() => expect(ReadThroughService.Findings).toHaveBeenCalledTimes(3))
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
    rerender(open(1))
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

  it('leaves the focus with a file clicked while its box is open', async () => {
    await readA()
    await userEvent.click(screen.getByRole('button', { name: 'Add note' }))
    const next = screen.getByRole('button', { name: /^a\.test\.ts/ })
    await userEvent.click(next)
    expect(screen.queryByRole('textbox', { name: 'Note on src/a.ts' })).toBeNull()
    expect(next).toHaveFocus()
  })

  it('keeps what is being typed when the file changes', async () => {
    const { rerender } = await opened()
    await userEvent.click(screen.getByRole('button', { name: /^a\.ts/ }))
    await screen.findByText('new')
    await userEvent.click(screen.getByRole('button', { name: 'Add note on line 1' }))
    await userEvent.type(screen.getByRole('textbox', { name: 'Note on line 1' }), 'half a tho')
    vi.mocked(ReadThroughService.Open).mockResolvedValue(info({ files: [file('src/a.ts', { newBlob: 'n2' })] }))
    const line = { ...diff.hunks[0].lines[1], content: 'newer' }
    vi.mocked(ReadThroughService.FileDiff).mockResolvedValue(
      new FileDiff({ hunks: [{ ...diff.hunks[0], lines: [diff.hunks[0].lines[0], line] }] }),
    )
    rerender(open(1))
    expect(await screen.findByText('newer')).toBeInTheDocument()
    expect(screen.getByRole('textbox', { name: 'Note on line 1' })).toHaveValue('half a tho')
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

describe('the structure', () => {
  it('opens an area to its files alone, and back to all', async () => {
    vi.mocked(ReadThroughService.Structure).mockResolvedValue(
      new ChangeStructureInfo({
        read: true,
        areas: [
          { path: '', state: '', files: ['go.sum'], added: 1, removed: 0 },
          { path: 'src', state: '', files: ['src/a.ts', 'src/a.test.ts'], added: 4, removed: 2 },
        ],
        readFirst: [{ path: 'src', folder: true, why: 'used', count: 2 }],
      }),
    )
    render(open())
    const structure = within(await screen.findByRole('region', { name: 'Structure' }))
    expect(structure.getByText('2 folders · no link between folders added or removed')).toBeInTheDocument()
    await userEvent.click(
      within(structure.getByRole('list', { name: 'Read first' })).getByRole('button', { name: 'src/' }),
    )
    expect(screen.getByText('Only src')).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: /go\.sum/ })).toBeNull()
    expect(ReadThroughService.FileDiff).toHaveBeenCalledWith(expect.anything(), '', 'src/a.ts', false, false)
    await userEvent.click(screen.getByRole('button', { name: 'Show all' }))
    expect(screen.getByText('go.sum')).toBeInTheDocument()
  })

  it('lists everything again when the folder has nothing left listed, or what is read changes', async () => {
    vi.mocked(ReadThroughService.Structure).mockResolvedValue(
      new ChangeStructureInfo({
        read: true,
        areas: [{ path: 'src', state: '', files: ['src/a.ts'], added: 1, removed: 0 }],
      }),
    )
    const { rerender } = render(open())
    const structure = within(await screen.findByRole('region', { name: 'Structure' }))
    await userEvent.click(structure.getByRole('button', { name: 'src/' }))
    expect(screen.getByText('Only src')).toBeInTheDocument()
    vi.mocked(ReadThroughService.Open).mockResolvedValue(info({ files: [file('docs/x.md')] }))
    rerender(open(1))
    expect(await screen.findByText('x.md')).toBeInTheDocument()
    expect(screen.queryByText('Only src')).toBeNull()
  })
})
