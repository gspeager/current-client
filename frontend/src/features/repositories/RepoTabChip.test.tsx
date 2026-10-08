import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import RepoTabChip from './RepoTabChip'

describe('RepoTabChip', () => {
  it('is a button named after the repository and its state, separate from Close', async () => {
    const onFocus = vi.fn()
    const onClose = vi.fn()
    render(
      <RepoTabChip
        path="/repos/app"
        summary={{
          path: '/repos/app',
          name: 'app',
          available: true,
          dirty: 2,
          currentBranch: 'main',
          ahead: 1,
          behind: 3,
          activity: [],
        }}
        onFocus={onFocus}
        onClose={onClose}
        onDragStart={vi.fn()}
        onDragOver={vi.fn()}
        onDrop={vi.fn()}
      />,
    )

    const tab = screen.getByRole('button', { name: 'app, main, 3 behind, 1 ahead, uncommitted changes' })
    tab.focus()
    await userEvent.keyboard('{Enter}')
    expect(onFocus).toHaveBeenCalled()
    expect(onClose).not.toHaveBeenCalled()

    await userEvent.click(screen.getByRole('button', { name: 'Close app' }))
    expect(onClose).toHaveBeenCalled()
  })
})
