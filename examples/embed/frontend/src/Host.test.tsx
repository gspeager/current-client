import { render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import Host from './Host'

function currentClientRoot(): HTMLElement {
  return document.querySelector('.current-client') as HTMLElement
}

describe('Host', () => {
  it('starts on its own view with Current Client mounted but hidden', () => {
    render(<Host />)

    expect(screen.getByRole('heading', { name: 'Host view' })).toBeInTheDocument()
    expect(currentClientRoot()).not.toBeVisible()
  })

  it('switches to Current Client and back without remounting it', async () => {
    render(<Host />)
    const root = currentClientRoot()

    await userEvent.click(screen.getByRole('button', { name: 'Current Client' }))
    expect(currentClientRoot()).toBe(root)
    expect(root).toBeVisible()
    expect(screen.queryByRole('heading', { name: 'Host view' })).not.toBeInTheDocument()

    // The toggle is also in Current Client's own header (here its welcome screen).
    await userEvent.click(within(root).getByRole('button', { name: 'Host view' }))
    expect(currentClientRoot()).toBe(root)
    expect(root).not.toBeVisible()
  })
})
