import { render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { BranchService, CompareService, GridInfo, OverlapService, TagService } from '@current-client-bindings/app'
import ConflictGridDialog from './ConflictGridDialog'

const grid = (over: Partial<GridInfo> = {}) =>
  new GridInfo({
    supported: true,
    branches: ['main', 'one', 'two'].map((name) => ({ name, current: false, lastCommitDate: '' })),
    conflicts: [{ a: 0, b: 1, files: ['src/a.ts', 'src/b.ts'] }],
    unchecked: [],
    older: 0,
    unrelated: [],
    ...over,
  })

const open = (info: GridInfo) => {
  vi.mocked(OverlapService.Grid).mockResolvedValue(info)
  return render(<ConflictGridDialog repoPath="/repo" onClose={() => {}} />)
}
const cell = (name: RegExp) => screen.findByRole('gridcell', { name })
const rows = () => screen.getAllByRole('rowheader').map((el) => el.getAttribute('aria-label'))

beforeEach(() => {
  vi.mocked(BranchService.ListLocal).mockResolvedValue([])
  vi.mocked(BranchService.ListRemote).mockResolvedValue([])
  vi.mocked(TagService.ListTags).mockResolvedValue([])
  vi.mocked(CompareService.GetChangedFiles).mockResolvedValue([])
})

it('opens a pair to its files, and compares them', async () => {
  open(grid())
  await userEvent.click(await cell(/one and main/))
  const opened = screen.getByRole('region', { name: 'Opened pair' })
  expect(opened).toHaveTextContent('one and main: 2 files would conflict.')
  await userEvent.click(within(opened).getByRole('button', { name: 'src/b.ts' }))
  expect(CompareService.GetChangedFiles).toHaveBeenCalledWith('/repo', 'one', 'main')
  expect(screen.getByRole('dialog', { name: 'Compare' })).toBeInTheDocument()
})

it('says a clean pair is, and still compares it', async () => {
  open(grid())
  await userEvent.click(await cell(/main and two/))
  const opened = screen.getByRole('region', { name: 'Opened pair' })
  expect(opened).toHaveTextContent('main and two merge without conflicts.')
  await userEvent.click(within(opened).getByRole('button', { name: 'Compare' }))
  expect(CompareService.GetChangedFiles).toHaveBeenCalledWith('/repo', 'main', 'two')
})

it('shows only branches with a conflict', async () => {
  open(grid())
  await cell(/main and two/)
  await userEvent.click(screen.getByRole('checkbox', { name: 'Only branches with a conflict' }))
  expect(rows()).toEqual(['main', 'one'])
})

it('says when no branches would conflict', async () => {
  open(grid({ conflicts: [] }))
  await cell(/main and two/)
  await userEvent.click(screen.getByRole('checkbox', { name: 'Only branches with a conflict' }))
  expect(screen.getByText('No branches would conflict.')).toBeInTheDocument()
})

it('goes on with the pairs there was no time for', async () => {
  open(grid({ unchecked: [[1, 2]] }))
  expect(await cell(/one and two, not checked/)).toHaveTextContent('?')
  expect(screen.getByRole('status')).toHaveTextContent('1 pair not checked in time.')
  vi.mocked(OverlapService.Grid).mockResolvedValue(grid())
  await userEvent.click(screen.getByRole('button', { name: 'Check more' }))
  expect(await cell(/one and two, no conflicts/)).toBeInTheDocument()
  expect(screen.queryByRole('button', { name: 'Check more' })).toBeNull()
})

it('says what is left out', async () => {
  open(grid({ older: 4, unrelated: ['gh-pages'] }))
  await cell(/main and two/)
  expect(screen.getByText(/Left out: 4 older branches; gh-pages \(no shared history\)/)).toBeInTheDocument()
})

it('says when Git is too old, and when there is nothing to compare', async () => {
  const { unmount } = open(new GridInfo())
  expect(await screen.findByText('Predicting conflicts needs Git 2.38 or newer.')).toBeInTheDocument()
  unmount()
  open(grid({ branches: [], conflicts: [] }))
  expect(await screen.findByText('No two unmerged local branches.')).toBeInTheDocument()
})

it('keeps the opened pair when branches change places', async () => {
  const { rerender } = open(grid())
  await userEvent.click(await cell(/main and one/))
  vi.mocked(OverlapService.Grid).mockResolvedValue(
    grid({
      branches: ['two', 'main', 'one'].map((name) => ({ name, current: false, lastCommitDate: '' })),
      conflicts: [{ a: 1, b: 2, files: ['src/a.ts'] }],
    }),
  )
  rerender(<ConflictGridDialog repoPath="/repo" refreshKey={1} onClose={() => {}} />)
  await screen.findByText('1 file would conflict.', { exact: false })
  expect(rows()).toEqual(['two', 'main', 'one'])
  expect(screen.getByRole('gridcell', { name: /main and one/ })).toHaveAttribute('aria-selected', 'true')
})
