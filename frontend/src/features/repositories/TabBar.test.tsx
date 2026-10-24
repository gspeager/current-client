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

describe('TabBar divider', () => {
  it('separates the core tabs from the reports, with or without the host tab', () => {
    const { container, rerender } = render(<TabBar active="working-copy" onChange={vi.fn()} showChangelog />)

    const divider = container.querySelector('.tab-bar-divider')!
    expect(divider).toHaveAttribute('aria-hidden', 'true')
    expect(divider.previousElementSibling).toHaveTextContent('History')
    expect(divider.nextElementSibling).toHaveTextContent('Activity')

    rerender(<TabBar active="working-copy" onChange={vi.fn()} showChangelog changesLabel="Pull requests" />)
    expect(container.querySelector('.tab-bar-divider')!.previousElementSibling).toHaveTextContent('Pull requests')
  })
})
