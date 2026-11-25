import { act, render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import ConflictGrid, { type GridCell } from './ConflictGrid'

const onSelect = vi.fn()
const cell = (a: number, b: number): GridCell => {
  const pair = [a, b].sort().join()
  if (pair === '0,1') return { files: ['a.ts', 'b.ts'] }
  return pair === '1,2' ? 'unchecked' : 'clean'
}
const view = (names = ['main', 'one', 'two']) =>
  render(<ConflictGrid label="Conflicts" names={names} cell={cell} selected={null} onSelect={onSelect} />)
const focused = () => document.activeElement?.getAttribute('aria-label')
const start = () => act(() => document.querySelector<HTMLElement>('[tabindex="0"]')!.focus())

it('says each pair in words and as text', () => {
  view()
  const conflict = screen.getByRole('gridcell', { name: 'main and one, 2 files would conflict' })
  expect(conflict).toHaveTextContent('2')
  expect(screen.getByRole('gridcell', { name: 'one and main, 2 files would conflict' })).toHaveTextContent('2')
  expect(screen.getByRole('gridcell', { name: 'one and two, not checked' })).toHaveTextContent('?')
  expect(screen.getByRole('gridcell', { name: 'main and two, no conflicts' })).toHaveTextContent('·')
  expect(screen.getAllByRole('gridcell')).toHaveLength(6)
})

it('is one tab stop, and the arrow keys pass over a name against itself', async () => {
  view()
  expect(document.querySelectorAll('[tabindex="0"]')).toHaveLength(1)
  start()
  expect(focused()).toBe('main and one, 2 files would conflict')
  await userEvent.keyboard('{ArrowDown}')
  expect(focused()).toBe('two and one, not checked')
  await userEvent.keyboard('{ArrowLeft}')
  expect(focused()).toBe('two and main, no conflicts')
  await userEvent.keyboard('{ArrowUp}{ArrowRight}')
  expect(focused()).toBe('one and two, not checked')
  await userEvent.keyboard('{ArrowRight}')
  expect(focused()).toBe('one and two, not checked')
  await userEvent.keyboard('{Home}')
  expect(focused()).toBe('one and main, 2 files would conflict')
  await userEvent.keyboard('{Enter}')
  expect(onSelect).toHaveBeenCalledWith(1, 0)
})

it('keeps a tab stop when there are fewer names', async () => {
  const { rerender } = view()
  start()
  await userEvent.keyboard('{ArrowDown}')
  rerender(<ConflictGrid label="Conflicts" names={['main', 'one']} cell={cell} selected={null} onSelect={onSelect} />)
  expect(document.querySelectorAll('[tabindex="0"]')).toHaveLength(1)
})
