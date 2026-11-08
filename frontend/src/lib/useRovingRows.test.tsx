import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
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
