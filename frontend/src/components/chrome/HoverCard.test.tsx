import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import HoverCard from './HoverCard'

describe('HoverCard', () => {
  it('appears after a pause of the pointer and goes on mouse-out', async () => {
    render(
      <HoverCard card={<p className="hover-card-title">Add the thing</p>}>
        <button type="button">!138</button>
      </HoverCard>,
    )
    await userEvent.hover(screen.getByRole('button', { name: '!138' }))
    expect(screen.queryByRole('tooltip')).not.toBeInTheDocument()
    expect(await screen.findByRole('tooltip')).toHaveTextContent('Add the thing')
    await userEvent.unhover(screen.getByRole('button', { name: '!138' }))
    await waitFor(() => expect(screen.queryByRole('tooltip')).not.toBeInTheDocument())
  })

  it('appears on keyboard focus and goes on Escape', async () => {
    render(
      <HoverCard card={<p>Login fails</p>}>
        <a href="https://example.com/issues/4">#4</a>
      </HoverCard>,
    )
    await userEvent.tab()
    expect(await screen.findByRole('tooltip')).toHaveTextContent('Login fails')
    await userEvent.keyboard('{Escape}')
    await waitFor(() => expect(screen.queryByRole('tooltip')).not.toBeInTheDocument())
  })

  it('is just the anchor without a card', () => {
    render(
      <HoverCard card={null}>
        <span>plain</span>
      </HoverCard>,
    )
    expect(screen.getByText('plain')).toBeInTheDocument()
    expect(screen.queryByRole('tooltip')).not.toBeInTheDocument()
  })
})
