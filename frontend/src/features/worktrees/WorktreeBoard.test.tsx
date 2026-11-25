import { act, fireEvent, render, screen, within } from '@testing-library/react'
import { BoardColumn, BoardInfo, WorktreeService } from '@current-client-bindings/app'
import { ReadThroughContext } from '../read-through/readThrough'
import { nextRead } from './useBoard'
import WorktreeBoard from './WorktreeBoard'

const column = (name: string, over: Partial<BoardColumn> = {}, worktree = {}) =>
  new BoardColumn({
    worktree: {
      path: `/work/${name}`,
      branch: `branch-${name}`,
      head: 'abcdef1234567',
      main: false,
      current: false,
      locked: false,
      missing: false,
      bare: false,
      ...worktree,
    },
    changed: over.files?.length ?? 0,
    files: [],
    ...over,
  })

const board = (over: Partial<BoardInfo> = {}) =>
  new BoardInfo({
    base: 'origin/main',
    columns: [column('main', {}, { main: true, current: true }), column('one'), column('two')],
    collisions: [],
    unchecked: [],
    ...over,
  })

const edits = (...paths: string[]) => paths.map((path) => ({ path, status: 'M' }))

const onOpenRepository = vi.fn()
const onOpenTerminal = vi.fn()
const onNewWorktree = vi.fn()
const readThrough = vi.fn()

const open = async (info: BoardInfo, terminal = true) => {
  vi.mocked(WorktreeService.Board).mockResolvedValue(info)
  render(
    <ReadThroughContext.Provider value={readThrough}>
      <WorktreeBoard
        repoPath="/work/main"
        onOpenRepository={onOpenRepository}
        onOpenTerminal={terminal ? onOpenTerminal : undefined}
        onNewWorktree={onNewWorktree}
        onClose={() => {}}
      />
    </ReadThroughContext.Provider>,
  )
  await act(() => vi.advanceTimersByTimeAsync(0))
}
const after = (ms: number) => act(() => vi.advanceTimersByTimeAsync(ms))
const col = (name: string) => within(screen.getByRole('region', { name }))

beforeEach(() => {
  vi.useFakeTimers()
})
afterEach(() => {
  vi.useRealTimers()
})

it('shows a column for each worktree, with what it has', async () => {
  await open(
    board({
      columns: [
        column('main', {}, { current: true }),
        column('one', {
          files: edits('a.ts', 'b.ts'),
          changed: 5,
          added: 4,
          removed: 2,
          ahead: 3,
          behind: 1,
          skipped: 2,
        }),
        column('two', {}, { branch: '', locked: true }),
        column('gone', {}, { missing: true }),
      ],
    }),
  )
  const one = screen.getByRole('region', { name: 'one' })
  expect(one).toHaveTextContent('5 changed files+4-2')
  expect(one).toHaveTextContent('3 more files')
  expect(one).toHaveTextContent('↑3 ↓1origin/main')
  expect(one).toHaveTextContent('2 large untracked files not counted.')
  expect(screen.getByRole('region', { name: 'two' })).toHaveTextContent('detached abcdef1')
  expect(screen.getByRole('region', { name: 'two' })).toHaveTextContent('No changes')
  const gone = screen.getByRole('region', { name: 'gone' })
  expect(gone).toHaveTextContent('missing')
  expect(within(gone).queryByRole('button')).toBeNull()
})

it('updates a column as its files change', async () => {
  await open(board())
  expect(screen.getByRole('region', { name: 'one' })).toHaveTextContent('No changes')
  vi.mocked(WorktreeService.Board).mockResolvedValue(
    board({ columns: [column('main'), column('one', { files: edits('late.ts') })] }),
  )
  await after(1900)
  expect(WorktreeService.Board).toHaveBeenCalledTimes(1)
  await after(200)
  expect(screen.getByRole('region', { name: 'one' })).toHaveTextContent('late.ts')
  // A worktree that was removed is gone.
  expect(screen.queryByRole('region', { name: 'two' })).toBeNull()
})

it('reads nothing while it is out of view', async () => {
  await open(board())
  const visible = vi.spyOn(document, 'visibilityState', 'get').mockReturnValue('hidden')
  act(() => void document.dispatchEvent(new Event('visibilitychange')))
  await after(60_000)
  expect(WorktreeService.Board).toHaveBeenCalledTimes(1)
  visible.mockReturnValue('visible')
  act(() => void document.dispatchEvent(new Event('visibilitychange')))
  await after(0)
  expect(WorktreeService.Board).toHaveBeenCalledTimes(2)
})

it('keeps reading after a failure', async () => {
  await open(board())
  vi.mocked(WorktreeService.Board).mockRejectedValue(new Error('index locked'))
  await after(2100)
  expect(screen.getByText('index locked')).toBeInTheDocument()
  expect(screen.getByRole('region', { name: 'one' })).toBeInTheDocument()
  vi.mocked(WorktreeService.Board).mockResolvedValue(board())
  await after(2100)
  expect(screen.queryByText('index locked')).toBeNull()
})

it('waits longer where reading takes longer', () => {
  expect(nextRead(5)).toBe(2000)
  expect(nextRead(900)).toBe(9000)
  expect(nextRead(60_000)).toBe(30_000)
})

it('marks what is running, unless there is no terminal', async () => {
  const running = board({ columns: [column('main'), column('one', { running: true })] })
  await open(running)
  expect(col('one').getByText('running')).toBeInTheDocument()
  expect(col('main').queryByText('running')).toBeNull()
})

it('leaves the terminal and the running mark out where there is none', async () => {
  await open(board({ columns: [column('main'), column('one', { running: true })] }), false)
  expect(screen.queryByText('running')).toBeNull()
  expect(screen.queryByRole('button', { name: /Terminal/ })).toBeNull()
  expect(col('one').getByRole('button', { name: 'Read through one' })).toBeInTheDocument()
})

it('marks a file that collides in each column, committed or not', async () => {
  await open(
    board({
      columns: [
        column('main', { files: edits('z.ts', 'a.ts') }),
        column('one', { files: edits('z.ts') }),
        column('two'),
      ],
      collisions: [
        { a: 0, b: 1, files: ['z.ts'] },
        { a: 0, b: 2, files: ['z.ts'] },
      ],
    }),
  )
  expect(col('main').getByRole('img', { name: 'Collides with one, two' })).toBeInTheDocument()
  expect(
    col('main')
      .getAllByRole('listitem')
      .map((li) => li.textContent),
  ).toEqual([expect.stringContaining('z.ts'), expect.stringContaining('a.ts')])
  expect(col('one').getByRole('img', { name: 'Collides with main' })).toBeInTheDocument()
  // Committed there, so it isn't one of its changed files.
  expect(screen.getByRole('region', { name: 'two' })).toHaveTextContent('No changes')
  expect(col('two').getByRole('img', { name: 'Collides with main' }).closest('li')).toHaveTextContent('z.ts')
})

it('opens a pair of the grid to its files', async () => {
  await open(board({ collisions: [{ a: 1, b: 2, files: ['x.ts'] }], unchecked: [[0, 1]] }))
  fireEvent.click(screen.getByRole('gridcell', { name: 'two and one, 1 file would conflict' }))
  expect(screen.getByRole('region', { name: 'Opened pair' })).toHaveTextContent(
    'two and one: 1 file would conflict.x.ts',
  )
  expect(screen.getByRole('gridcell', { name: 'main and one, not checked' })).toBeInTheDocument()
})

it('says why collisions are not predicted', async () => {
  await open(board({ unpredicted: 'partial' }))
  expect(screen.getByText(/Collisions aren’t predicted in a partial clone/)).toBeInTheDocument()
  expect(screen.queryByRole('grid')).toBeNull()
})

it('opens a worktree, its terminal and its review, and makes a new one', async () => {
  await open(board({ more: 3 }))
  expect(screen.getByText('3 more worktrees not shown.')).toBeInTheDocument()
  expect(col('main').queryByRole('button', { name: 'Open main' })).toBeNull()
  fireEvent.click(col('one').getByRole('button', { name: 'Open one' }))
  expect(onOpenRepository).toHaveBeenCalledWith('/work/one')
  fireEvent.click(col('one').getByRole('button', { name: 'Terminal in one' }))
  expect(onOpenTerminal).toHaveBeenCalledWith('/work/one')
  fireEvent.click(col('two').getByRole('button', { name: 'Read through two' }))
  expect(readThrough).toHaveBeenCalledWith({ path: '/work/two' })
  fireEvent.click(screen.getByRole('button', { name: 'New worktree' }))
  expect(onNewWorktree).toHaveBeenCalled()
})

it('names two folders of one name by their paths', async () => {
  await open(board({ columns: [column('one'), column('two', {}, { path: '/elsewhere/one' })] }))
  expect(screen.getByRole('region', { name: '/work/one' })).toBeInTheDocument()
  expect(screen.getByRole('region', { name: '/elsewhere/one' })).toBeInTheDocument()
})

it('shows why a worktree could not be read, beside the others', async () => {
  await open(board({ columns: [column('main'), column('one', { problem: 'index file corrupt' })] }))
  expect(col('one').getByText('index file corrupt')).toBeInTheDocument()
  expect(screen.getByRole('region', { name: 'main' })).toHaveTextContent('No changes')
})
