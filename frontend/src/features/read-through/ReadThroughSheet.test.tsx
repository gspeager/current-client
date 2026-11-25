import { fireEvent, render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import {
  BranchService,
  DiffService,
  FileDiff,
  ReadThroughFile,
  ReadThroughInfo,
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

const open = (refreshKey = 0, request = { ref: 'refs/heads/feat' }) => (
  <ReadThroughSheet repoPath="/repo" request={request} refreshKey={refreshKey} onClose={() => undefined} />
)

beforeEach(() => {
  vi.mocked(ReadThroughService.Open).mockResolvedValue(info())
  vi.mocked(ReadThroughService.FileDiff).mockResolvedValue(diff)
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
