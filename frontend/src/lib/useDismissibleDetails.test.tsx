import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { useDismissibleDetails } from './useDismissibleDetails'

function Menu() {
  const ref = useDismissibleDetails()
  return (
    <>
      <details ref={ref} data-testid="menu">
        <summary>Open</summary>
        <button type="button">Inside</button>
      </details>
      <button type="button">Outside</button>
    </>
  )
}

describe('useDismissibleDetails', () => {
  it('closes on Escape and on a click outside, but not on a click inside', async () => {
    render(<Menu />)
    const menu = screen.getByTestId('menu') as HTMLDetailsElement

    await userEvent.click(screen.getByText('Open'))
    await userEvent.click(screen.getByRole('button', { name: 'Inside' }))
    expect(menu.open).toBe(true)
    await userEvent.keyboard('{Escape}')
    expect(menu.open).toBe(false)

    await userEvent.click(screen.getByText('Open'))
    await userEvent.click(screen.getByRole('button', { name: 'Outside' }))
    expect(menu.open).toBe(false)
  })
})
