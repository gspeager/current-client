import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { useState } from 'react'
import { describe, expect, it, vi } from 'vitest'
import { useRovingRows } from './useRovingRows'

function List({ onPick, selectOnMove = false }: { onPick?: (n: number) => void; selectOnMove?: boolean }) {
  const ref = useRovingRows('.row', { selectOnMove })
  return (
    <div ref={ref}>
      {[0, 1, 2].map((n) => (
        <div key={n} className="row" tabIndex={0} role="button" aria-label={`Row ${n}`} onClick={() => onPick?.(n)}>
          <button type="button">Action {n}</button>
        </div>
      ))}
    </div>
  )
}

describe('useRovingRows', () => {
  it('leaves one Tab stop in the list', async () => {
    render(<List />)
    await userEvent.tab()
    expect(screen.getByRole('button', { name: 'Row 0' })).toHaveFocus()
    await userEvent.tab()
    expect(document.body).toHaveFocus()
  })

  it('moves with the arrow keys, Home and End', async () => {
    render(<List />)
    screen.getByRole('button', { name: 'Row 0' }).focus()
    await userEvent.keyboard('{ArrowDown}')
    expect(screen.getByRole('button', { name: 'Row 1' })).toHaveFocus()
    await userEvent.keyboard('{End}')
    expect(screen.getByRole('button', { name: 'Row 2' })).toHaveFocus()
    await userEvent.keyboard('{Home}')
    expect(screen.getByRole('button', { name: 'Row 0' })).toHaveFocus()
  })

  it('reaches a row control with the right arrow', async () => {
    render(<List />)
    screen.getByRole('button', { name: 'Row 0' }).focus()
    await userEvent.keyboard('{ArrowRight}')
    expect(screen.getByRole('button', { name: 'Action 0' })).toHaveFocus()
  })

  it('selects as it moves when asked', async () => {
    const onPick = vi.fn()
    render(<List onPick={onPick} selectOnMove />)
    screen.getByRole('button', { name: 'Row 0' }).focus()
    await userEvent.keyboard('{ArrowDown}')
    expect(onPick).toHaveBeenCalledWith(1)
  })
})

// Draws a few of its rows, as a virtualized list does.
function Windowed({ reversed = false }: { reversed?: boolean }) {
  const [first, setFirst] = useState(0)
  const at = (el: HTMLElement) => Number(el.dataset.at)
  const ref = useRovingRows('.row', {
    scrollPast: (row, forward) => setFirst(at(row) + (forward ? 0 : -2)),
    compareRows: (a, b) => at(a) - at(b),
  })
  const drawn = [first, first + 1, first + 2]
  return (
    <>
      <div ref={ref}>
        {(reversed ? [...drawn].reverse() : drawn).map((n) => (
          <button key={n} type="button" className="row" data-at={n}>
            Row {n}
          </button>
        ))}
      </div>
      <p onClick={() => setFirst(50)}>Elsewhere</p>
    </>
  )
}

describe('useRovingRows in a list that draws only some rows', () => {
  it('carries on past the last row drawn', async () => {
    render(<Windowed />)
    screen.getByRole('button', { name: 'Row 2' }).focus()
    await userEvent.keyboard('{ArrowDown}')
    await waitFor(() => expect(screen.getByRole('button', { name: 'Row 3' })).toHaveFocus())
  })

  it('follows the order on screen, not the order in the page', async () => {
    render(<Windowed reversed />)
    screen.getByRole('button', { name: 'Row 0' }).focus()
    await userEvent.keyboard('{ArrowDown}')
    expect(screen.getByRole('button', { name: 'Row 1' })).toHaveFocus()
  })

  it('leaves focus alone when a row clicked away from is later dropped', async () => {
    render(<Windowed />)
    screen.getByRole('button', { name: 'Row 1' }).focus()
    await userEvent.click(screen.getByText('Elsewhere'))
    await screen.findByRole('button', { name: 'Row 50' })
    await new Promise((resolve) => setTimeout(resolve))
    expect(document.body).toHaveFocus()
  })
})
