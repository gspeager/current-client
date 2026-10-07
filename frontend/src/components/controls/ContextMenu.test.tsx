import { useState } from 'react'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import ContextMenu, { isMenuKey, menuAnchor, type ContextMenuState } from './ContextMenu'

function Opener({ onPick = vi.fn() }: { onPick?: () => void }) {
  const [menu, setMenu] = useState<ContextMenuState | null>(null)
  const items = [
    { label: 'First', onClick: onPick },
    { label: 'Disabled', onClick: vi.fn(), disabled: true },
    { label: 'Last', onClick: vi.fn() },
  ]
  return (
    <>
      <button
        type="button"
        onClick={(e) => setMenu({ ...menuAnchor(e), items })}
        onKeyDown={(e) => isMenuKey(e) && setMenu({ ...menuAnchor(e), items })}
      >
        Open
      </button>
      {menu && <ContextMenu state={menu} onClose={() => setMenu(null)} />}
    </>
  )
}

describe('ContextMenu from the keyboard', () => {
  it('opens on Shift+F10, takes focus, and skips disabled items with the arrow keys', async () => {
    const user = userEvent.setup()
    render(<Opener />)

    screen.getByRole('button', { name: 'Open' }).focus()
    await user.keyboard('{Shift>}{F10}{/Shift}')

    expect(await screen.findByRole('button', { name: 'First' })).toHaveFocus()
    await user.keyboard('{ArrowDown}')
    expect(screen.getByRole('button', { name: 'Last' })).toHaveFocus()
    await user.keyboard('{ArrowDown}')
    expect(screen.getByRole('button', { name: 'First' })).toHaveFocus()
    await user.keyboard('{ArrowUp}')
    expect(screen.getByRole('button', { name: 'Last' })).toHaveFocus()
  })

  it('gives focus back to what opened it when it closes', async () => {
    const user = userEvent.setup()
    render(<Opener />)

    screen.getByRole('button', { name: 'Open' }).focus()
    await user.keyboard('{Enter}')
    await screen.findByRole('button', { name: 'First' })
    await user.keyboard('{Escape}')

    expect(screen.queryByRole('button', { name: 'First' })).not.toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Open' })).toHaveFocus()
  })

  it('runs the chosen item', async () => {
    const onPick = vi.fn()
    const user = userEvent.setup()
    render(<Opener onPick={onPick} />)

    await user.click(screen.getByRole('button', { name: 'Open' }))
    await user.keyboard('{Enter}')

    expect(onPick).toHaveBeenCalled()
  })
})
