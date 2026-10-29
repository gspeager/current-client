import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import TabBar from './TabBar'

describe('TabBar', () => {
  it('counts changed files on Changes, and conflicted files instead during a merge', () => {
    const { rerender } = render(<TabBar active="working-copy" onChange={vi.fn()} changedCount={5} />)
    expect(screen.getByRole('button', { name: 'Changes, 5 changed files' })).toHaveTextContent('Changes5')

    rerender(<TabBar active="working-copy" onChange={vi.fn()} changedCount={5} conflictCount={1} />)
    const tab = screen.getByRole('button', { name: 'Changes, 1 conflicted file' })
    expect(tab.querySelector('.tab-count-conflict')).toHaveTextContent('1')
  })

  it("labels an off host's tabs off, and opens its settings on click", async () => {
    const onOffHost = vi.fn()
    render(
      <TabBar
        active="working-copy"
        onChange={vi.fn()}
        offHost={{ hostname: 'github.com', changesLabel: 'Pull requests' }}
        onOffHost={onOffHost}
      />,
    )
    const tab = screen.getByRole('button', { name: 'Pull requests off' })
    expect(tab).toHaveClass('tab-off')
    expect(tab).toHaveAttribute('title', 'github.com is off · Integration settings')
    expect(screen.getByRole('button', { name: 'Issues off' })).toBeInTheDocument()
    await userEvent.click(tab)
    expect(onOffHost).toHaveBeenCalled()
  })
})

describe('TabBar divider', () => {
  it('separates the core tabs from the reports, with or without the host tab', () => {
    const { container, rerender } = render(<TabBar active="working-copy" onChange={vi.fn()} />)

    const divider = container.querySelector('.tab-bar-divider')!
    expect(divider).toHaveAttribute('aria-hidden', 'true')
    expect(divider.previousElementSibling).toHaveTextContent('History')
    expect(divider.nextElementSibling).toHaveTextContent('Activity')

    rerender(<TabBar active="working-copy" onChange={vi.fn()} changesLabel="Pull requests" />)
    expect(container.querySelector('.tab-bar-divider')!.previousElementSibling).toHaveTextContent('Pull requests')
  })
})
