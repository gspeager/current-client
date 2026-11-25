import { render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { ChangeStructureInfo, type ChangeArea, type ChangeLink } from '@current-client-bindings/app'
import ChangeStructure from './ChangeStructure'

const area = (path: string, over: Partial<ChangeArea> = {}): ChangeArea => ({
  path,
  state: '',
  files: [`${path}/x.go`],
  added: 10,
  removed: 2,
  ...over,
})
const link = (from: string, to: string, state = 'added', count = 1): ChangeLink => ({ from, to, count, state })

const onArea = vi.fn()
const onFile = vi.fn()
const show = (shape: Partial<ChangeStructureInfo>) =>
  render(
    <ChangeStructure
      shape={new ChangeStructureInfo({ read: true, ...shape })}
      error={null}
      onArea={onArea}
      onFile={onFile}
    />,
  )

it('draws the folders and the links the change adds or removes, and opens a folder', async () => {
  show({
    areas: [area('app'), area('text', { state: 'added' }), area('old', { state: 'removed', added: 0 })],
    links: [link('app', 'text'), link('app', 'core', 'removed', 3)],
  })
  expect(screen.getByText('3 folders · 1 new · 1 gone · 2 links between folders added or removed')).toBeInTheDocument()
  expect(screen.getByRole('button', { name: /^app, 1 file · \+10 -2, uses text\/, core\/$/ })).toBeInTheDocument()
  expect(screen.getByRole('button', { name: /^core, not changed, used by app\/$/ })).toBeInTheDocument()
  expect(screen.getByRole('button', { name: /^text, 1 file · \+10 -2, used by app\/$/ })).toBeInTheDocument()
  expect([...document.querySelectorAll('.structure-mark')].map((el) => el.textContent)).toEqual(['+', '−'])
  await userEvent.click(screen.getByRole('button', { name: /^text,/ }))
  expect(onArea).toHaveBeenCalledWith('text')
})

it('puts what to read first before what to skim, each with why', async () => {
  show({
    areas: [area('core'), area('app')],
    readFirst: [
      { path: 'core', folder: true, why: 'used', count: 2 },
      { path: 'core/types.go', folder: false, why: 'used', count: 14 },
    ],
    skim: [
      { path: 'go.sum', folder: false, why: 'lockfile', count: 0 },
      { path: 'a/b.go', folder: false, why: 'renamed', count: 0 },
      { path: 'app', folder: true, why: 'few', count: 3 },
    ],
  })
  const lists = screen.getAllByRole('list').map((list) => list.getAttribute('aria-label'))
  expect(lists.indexOf('Read first')).toBeLessThan(lists.indexOf('Skim'))
  const first = within(screen.getByRole('list', { name: 'Read first' }))
  expect(first.getAllByRole('listitem').map((li) => li.textContent)).toEqual([
    'core/used by 2 of the change’s folders',
    'core/types.goused by 14 files',
  ])
  expect(
    within(screen.getByRole('list', { name: 'Skim' }))
      .getAllByRole('listitem')
      .map((li) => li.textContent),
  ).toEqual(['go.sumLockfile', 'a/b.goRenamed, unchanged', 'app/3 lines in all'])
  await userEvent.click(first.getByRole('button', { name: 'core/types.go' }))
  expect(onFile).toHaveBeenCalledWith('core/types.go')
  await userEvent.click(first.getByRole('button', { name: 'core/' }))
  expect(onArea).toHaveBeenCalledWith('core')
})

it('lists a large restructure instead of drawing it', async () => {
  const links = Array.from({ length: 35 }, (_, i) => link(`from${i}`, `to${i}`, i % 2 ? 'removed' : 'added'))
  show({ areas: [area('a'), area('b')], links })
  expect(document.querySelector('.structure-graph')).toBeNull()
  expect(screen.getByText('A large restructure, so it’s listed rather than drawn.')).toBeInTheDocument()
  const listed = within(screen.getByRole('list', { name: 'Links added or removed' })).getAllByRole('listitem')
  expect(listed).toHaveLength(30)
  expect(listed[1]).toHaveTextContent('−from1/usesto1/1 link gone')
  expect(screen.getByText('5 more.')).toBeInTheDocument()
  await userEvent.click(within(listed[0]).getByRole('button', { name: 'to0/' }))
  expect(onArea).toHaveBeenCalledWith('to0')
})

it('lists a change whose links go to too many folders to draw', () => {
  const areas = Array.from({ length: 10 }, (_, i) => area(`a${i}`))
  show({ areas, links: Array.from({ length: 10 }, (_, i) => link(`a${i}`, `elsewhere${i}`)) })
  expect(document.querySelector('.structure-graph')).toBeNull()
  expect(screen.getByRole('list', { name: 'Links added or removed' })).toBeInTheDocument()
})

it('lists the folders when no link changes, and says when links are not read', () => {
  const { unmount } = show({ areas: [area(''), area('docs', { state: 'added' })] })
  expect(screen.getByText('2 folders · 1 new · no link between folders added or removed')).toBeInTheDocument()
  expect(screen.getByRole('button', { name: 'Top folder' })).toBeInTheDocument()
  expect(screen.getByText(/1 file · \+10 -2 · new/)).toBeInTheDocument()
  unmount()
  show({ read: false, areas: [area('src')] })
  expect(screen.getByText('No links: none of its languages is read. Folders only.')).toBeInTheDocument()
  expect(screen.getByText('1 folder')).toBeInTheDocument()
})

it('says why a partial clone has no links, and what went wrong', () => {
  const { unmount } = show({ read: false, partial: true, areas: [area('src')] })
  expect(screen.getByText(/aren’t read in a partial clone/)).toBeInTheDocument()
  unmount()
  render(<ChangeStructure shape={null} error="base is gone" onArea={onArea} onFile={onFile} />)
  expect(screen.getByText('base is gone')).toBeInTheDocument()
})
