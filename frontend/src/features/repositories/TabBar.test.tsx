import { render, screen } from '@testing-library/react'
import TabBar from './TabBar'

describe('TabBar', () => {
  it('counts changed files on Changes, and conflicted files instead during a merge', () => {
    const { rerender } = render(<TabBar active="working-copy" onChange={vi.fn()} showChangelog changedCount={5} />)
    expect(screen.getByRole('button', { name: 'Changes, 5 changed files' })).toHaveTextContent('Changes5')

    rerender(<TabBar active="working-copy" onChange={vi.fn()} showChangelog changedCount={5} conflictCount={1} />)
    const tab = screen.getByRole('button', { name: 'Changes, 1 conflicted file' })
    expect(tab.querySelector('.tab-count-conflict')).toHaveTextContent('1')
  })
})
