import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { DialogProvider } from '../../components/chrome/DialogProvider'
import ConflictBanner from './ConflictBanner'

describe('ConflictBanner', () => {
  it('lists each conflicted file and opens the one clicked', async () => {
    const onOpenFile = vi.fn()
    render(
      <DialogProvider>
        <ConflictBanner
          repoPath="/repos/app"
          state={{ operation: 'merge', conflictedPaths: ['src/app.ts', 'docs/readme.md'] }}
          onResolved={vi.fn()}
          onOpenFile={onOpenFile}
        />
      </DialogProvider>,
    )

    expect(screen.getByText(/2 files conflicted/)).toBeInTheDocument()
    await userEvent.click(screen.getByRole('button', { name: 'readme.md' }))

    expect(onOpenFile).toHaveBeenCalledWith('docs/readme.md')
  })
})
