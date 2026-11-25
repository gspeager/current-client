import { act, render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { useState } from 'react'
import {
  ExploreService,
  HistoryService,
  StructureInfo,
  type ExploreComparison,
  type StructureLink,
  type StructureNode,
} from '@current-client-bindings/app'
import StructureView from './StructureView'

const node = (path: string, over: Partial<StructureNode> = {}): StructureNode => ({
  path,
  label: `${path}/`,
  side: 'card',
  folder: true,
  count: { files: 2, lines: 1500, changed: 0 },
  state: '',
  doc: '',
  slot: 1,
  ...over,
})
const link = (from: number, to: number, over: Partial<StructureLink> = {}): StructureLink => ({
  from,
  to,
  count: 1,
  main: true,
  state: '',
  ...over,
})

const top = (over: Partial<StructureInfo> = {}) =>
  new StructureInfo({
    path: '',
    crumbs: [{ path: '', label: '' }],
    folder: true,
    count: { files: 12, lines: 4200, changed: 0 },
    nodes: [node('app'), node('core'), node('docs')],
    links: [link(0, 1, { count: 3 })],
    cards: 3,
    languages: ['Go'],
    ...over,
  })
const app = () =>
  new StructureInfo({
    path: 'app',
    crumbs: [
      { path: '', label: '' },
      { path: 'app', label: 'app' },
    ],
    folder: true,
    count: { files: 2, lines: 100, changed: 0 },
    nodes: [
      node('app/run.go', { label: 'run.go', folder: false }),
      node('main.go', { label: 'main.go', side: 'used-by', folder: false }),
      node('core', { side: 'uses' }),
    ],
    links: [link(1, 0), link(0, 2)],
    cards: 1,
    languages: ['Go'],
  })

const onHistory = vi.fn()
const onBlame = vi.fn()
const onDiff = vi.fn()

function Harness({ comparison = null }: { comparison?: ExploreComparison | null }) {
  const [folder, setFolder] = useState('')
  return (
    <StructureView
      repoPath="/repo"
      folder={folder}
      onFolder={setFolder}
      comparison={comparison}
      version={0}
      commitsOf={(path) => (path === 'app/run.go' ? 7 : undefined)}
      onHistory={onHistory}
      onBlame={onBlame}
      onDiff={onDiff}
    />
  )
}

const levels = (byPath: Record<string, StructureInfo>) =>
  vi.mocked(ExploreService.Structure).mockImplementation((_repo, path) => Promise.resolve(byPath[path]) as never)
const card = (name: RegExp) => screen.findByRole('button', { name })

it('names each card with what it holds and what its lines say', async () => {
  levels({ '': top() })
  render(<Harness />)
  expect(await card(/^app, 2 files · 1,500 lines, uses core\/$/)).toBeInTheDocument()
  expect(await card(/^core, 2 files · 1,500 lines, used by app\/$/)).toBeInTheDocument()
  expect(screen.getByText('12 files · 4,200 lines')).toBeInTheDocument()
  expect(screen.getByText(/Links are read from Go\. Tests, generated/)).toBeInTheDocument()
  expect(document.querySelectorAll('.structure-line')).toHaveLength(1)
})

it('goes into a card and back up by keyboard, keeping the focus on the cards', async () => {
  levels({ '': top(), app: app() })
  render(<Harness />)
  const first = await card(/^app,/)
  act(() => first.focus())
  await userEvent.keyboard('{Enter}')
  await card(/^app\/run.go, 1,500 lines · 7 commits, uses core\/, used by main.go$/)
  await waitFor(() => expect(document.activeElement).toHaveClass('structure-card'))
  expect(screen.getByRole('button', { name: /^main.go, outside, uses this/ })).toBeInTheDocument()
  expect(screen.getByText('Used by')).toBeInTheDocument()
  expect(screen.getByText('Uses')).toBeInTheDocument()
  await userEvent.keyboard('{Escape}')
  expect(await card(/^docs,/)).toBeInTheDocument()
})

it('is one tab stop, moved by the arrow keys', async () => {
  levels({ '': top() })
  render(<Harness />)
  act(() => document.querySelector<HTMLElement>('.structure-card')?.focus())
  const first = await card(/^app,/)
  act(() => first.focus())
  await userEvent.keyboard('{ArrowRight}')
  expect(document.activeElement).toHaveAccessibleName(/^core,/)
  expect(document.querySelectorAll('.structure-card[tabindex="0"]')).toHaveLength(1)
})

it('fades what a card does not touch', async () => {
  levels({ '': top() })
  render(<Harness />)
  await userEvent.hover(await card(/^app,/))
  expect(await card(/^docs,/)).toHaveClass('structure-dim')
  expect(await card(/^core,/)).not.toHaveClass('structure-dim')
})

it('draws the main links until all are asked for', async () => {
  levels({ '': top({ links: [link(0, 1), link(0, 2, { main: false }), link(1, 2, { main: false })] }) })
  render(<Harness />)
  const all = await screen.findByRole('button', { name: 'All links (3)' })
  expect(document.querySelectorAll('.structure-line')).toHaveLength(1)
  await userEvent.click(all)
  expect(document.querySelectorAll('.structure-line')).toHaveLength(3)
  expect(screen.getByRole('button', { name: 'Main links' })).toHaveAttribute('aria-pressed', 'true')
})

it('says what most cards share, and lists the rest', async () => {
  levels({
    '': top({
      cards: 1603,
      widelyUses: [{ path: 'helpers', label: 'helpers/', folder: true, cards: 1431 }],
      rest: [node('x', { label: 'x/' }), node('y', { label: 'y/', state: 'added' })],
      moreUses: 2,
      unread: 1,
    }),
    x: app(),
  })
  render(<Harness />)
  expect(await screen.findByText(/1,431 of 1,603/)).toHaveTextContent('helpers/')
  expect(screen.getByText('2 more, less connected')).toBeInTheDocument()
  expect(
    screen.getByText(/1 file of code in other languages isn't read\. 2 other places outside aren't shown\./),
  ).toBeInTheDocument()
  await userEvent.click(screen.getByText('x/'))
  expect(ExploreService.Structure).toHaveBeenLastCalledWith('/repo', 'x', null)
})

it('marks what is new, gone and changed without relying on color', async () => {
  levels({
    '': top({
      compared: true,
      target: 'main',
      count: { files: 12, lines: 10, changed: 1234 },
      nodes: [
        node('app', { count: { files: 2, lines: 5, changed: 3 } }),
        node('fresh', { state: 'added' }),
        node('gone', { state: 'removed', count: { files: 0, lines: 0, changed: 0 } }),
      ],
      links: [link(0, 1, { state: 'added' }), link(0, 2, { state: 'removed' })],
    }),
  })
  render(<Harness comparison={{ kind: 'branch', ref: 'feat' }} />)
  expect(await card(/^app, 2 files · 3 changed, uses fresh\/, gone\/$/)).toHaveTextContent('3 changed')
  expect(await card(/^fresh, new · 2 files .*, used by app\/$/)).toHaveTextContent('new')
  expect(await card(/^gone, removed, used by app\/$/)).toHaveTextContent('removed')
  expect([...document.querySelectorAll('.structure-mark')].map((el) => el.textContent)).toEqual(['+', '−'])
  expect(screen.getByText('against main')).toBeInTheDocument()
  expect(screen.getByText(/1,234 changed/)).toBeInTheDocument()
  expect(ExploreService.Structure).toHaveBeenCalledWith('/repo', '', { kind: 'branch', ref: 'feat' })
})

it('ends at a file, with its commits, history, blame and diff', async () => {
  vi.mocked(HistoryService.GetFileHistory).mockResolvedValue([
    { sha: 'abcdef123456', subject: 'Fix it', authorName: 'Ada', date: new Date().toISOString() } as never,
  ])
  levels({
    '': new StructureInfo({
      path: 'app/run.go',
      crumbs: [{ path: '', label: '' }],
      count: { files: 1, lines: 9, changed: 1 },
      nodes: [node('app/run.go', { label: 'run.go', folder: false })],
      compared: true,
      cards: 1,
    }),
  })
  render(<Harness comparison={{ kind: 'working', ref: '' }} />)
  const end = within(await screen.findByRole('region', { name: 'Latest commits' }))
  expect(await end.findByText('Fix it')).toBeInTheDocument()
  await userEvent.click(end.getByRole('button', { name: 'History' }))
  await userEvent.click(end.getByRole('button', { name: 'Blame' }))
  await userEvent.click(end.getByRole('button', { name: 'Diff' }))
  for (const called of [onHistory, onBlame, onDiff]) expect(called).toHaveBeenCalledWith('app/run.go')
})

it('says when nothing is read, when a clone is partial, and what went wrong', async () => {
  levels({ '': top({ languages: [], links: [], partial: true }) })
  const { unmount } = render(<Harness />)
  expect(await screen.findByText(/No links: none of its languages is read/)).toBeInTheDocument()
  expect(screen.getByText(/a partial clone would fetch/)).toBeInTheDocument()
  unmount()
  vi.mocked(ExploreService.Structure).mockRejectedValue(new Error('nowhere is not a commit'))
  render(<Harness />)
  expect(await screen.findByText('nowhere is not a commit')).toBeInTheDocument()
})
