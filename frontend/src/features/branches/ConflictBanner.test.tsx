import { render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { ConflictService, ConflictStateInfo } from '@current-client-bindings/app'
import { DialogProvider } from '../../components/chrome/DialogProvider'
import ConflictBanner from './ConflictBanner'

const merge = new ConflictStateInfo({
  operation: 'merge',
  conflictedPaths: ['src/app.ts', 'docs/readme.md'],
  ours: 'main',
  theirs: 'feature',
  message: "Merge branch 'feature'",
})

function renderBanner(state = merge, onResolved = vi.fn(), onOpenFile = vi.fn()) {
  render(
    <DialogProvider>
      <ConflictBanner repoPath="/repos/app" state={state} onResolved={onResolved} onOpenFile={onOpenFile} />
    </DialogProvider>,
  )
  return { onResolved, onOpenFile }
}

describe('ConflictBanner', () => {
  it('says what is merging into what, lists each conflicted file and opens the one clicked', async () => {
    const { onOpenFile } = renderBanner()

    expect(screen.getByText('Merging feature into main · 2 files still conflicted')).toBeInTheDocument()
    expect(screen.getByText(/continue the merge from the commit message/)).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Continue' })).not.toBeInTheDocument()
    await userEvent.click(screen.getByRole('button', { name: 'readme.md' }))

    expect(onOpenFile).toHaveBeenCalledWith('docs/readme.md')
  })

  it('points at the composer once every file is resolved', () => {
    renderBanner(new ConflictStateInfo({ ...merge, operation: 'rebase', conflictedPaths: [] }))

    expect(screen.getByText('Rebasing feature onto main · all conflicts resolved')).toBeInTheDocument()
    expect(screen.getByText('Continue the rebase from the commit message.')).toBeInTheDocument()
  })

  it('aborts after confirming', async () => {
    vi.mocked(ConflictService.Abort).mockResolvedValue()
    const { onResolved } = renderBanner()

    await userEvent.click(screen.getByRole('button', { name: 'Abort' }))
    const dialog = await screen.findByRole('alertdialog', { name: 'Abort merge' })
    await userEvent.click(within(dialog).getByRole('button', { name: 'Abort' }))

    expect(ConflictService.Abort).toHaveBeenCalledWith('/repos/app')
    await vi.waitFor(() => expect(onResolved).toHaveBeenCalled())
  })
})
