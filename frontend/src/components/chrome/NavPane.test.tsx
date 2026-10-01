import { render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { PlatformService, UndoService, type UndoPlanInfo } from '@current-client-bindings/app'
import type { useWorkingTree } from '../../features/working-copy/useWorkingTree'
import { DialogProvider } from './DialogProvider'
import NavPane from './NavPane'

const REPO = '/repos/app'

const workingTree = {
  files: [],
  conflicted: [],
  staged: [],
  unstaged: [],
  error: null,
  loadStatus: vi.fn(),
} as unknown as ReturnType<typeof useWorkingTree>

function renderNav(collapsed: boolean, onCollapsedChange = vi.fn()) {
  const ui = (isCollapsed: boolean) => (
    <DialogProvider>
      <NavPane
        repoPath={REPO}
        repoVersion={0}
        workingTree={workingTree}
        onBranchChanged={vi.fn()}
        onFetched={vi.fn()}
        onOpenHeadCommit={vi.fn()}
        width={240}
        collapsed={isCollapsed}
        onCollapsedChange={onCollapsedChange}
      />
    </DialogProvider>
  )
  const { rerender } = render(ui(collapsed))
  return { onCollapsedChange, setCollapsed: (next: boolean) => rerender(ui(next)) }
}

describe('NavPane collapsed rail', () => {
  it('runs workspace actions straight from their icons', async () => {
    vi.mocked(PlatformService.OpenTerminal).mockResolvedValue()
    const { onCollapsedChange } = renderNav(true)

    await userEvent.click(screen.getByRole('button', { name: 'Open Terminal' }))

    expect(PlatformService.OpenTerminal).toHaveBeenCalledWith(REPO)
    expect(onCollapsedChange).not.toHaveBeenCalled()
    expect(screen.queryByText('Branches')).not.toBeInTheDocument()
  })

  it('expands to show a section picked from the rail', async () => {
    const { onCollapsedChange } = renderNav(true)

    await userEvent.click(screen.getByRole('button', { name: 'Show Branches' }))

    expect(onCollapsedChange).toHaveBeenCalledWith(false)
  })

  it('expands so a failed action can show why', async () => {
    vi.mocked(PlatformService.OpenInEditor).mockRejectedValue(new Error('Editor not found.'))
    const { onCollapsedChange, setCollapsed } = renderNav(true)

    await userEvent.click(screen.getByRole('button', { name: 'Open in Editor' }))
    await vi.waitFor(() => expect(onCollapsedChange).toHaveBeenCalledWith(false))
    setCollapsed(false)

    expect(await screen.findByText('Could not open editor: Editor not found.')).toBeInTheDocument()
  })
})

describe('NavPane expanded', () => {
  it('collapses to the rail', async () => {
    const { onCollapsedChange } = renderNav(false)

    await userEvent.click(screen.getByRole('button', { name: 'Collapse sidebar' }))

    expect(onCollapsedChange).toHaveBeenCalledWith(true)
  })
})

describe('NavPane undo', () => {
  const plan = {
    operation: 'merge',
    detail: 'Merge made by the ort strategy.',
    branch: 'main',
    from: 'b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0c1',
    to: 'a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0',
    switchTo: '',
    mode: 'keep',
    removed: 2,
    restored: 0,
    pushed: false,
  } as UndoPlanInfo

  it('previews the undo and applies it once confirmed', async () => {
    vi.mocked(UndoService.Preview).mockResolvedValue(plan)
    vi.mocked(UndoService.Apply).mockResolvedValue()
    renderNav(false)

    await userEvent.click(screen.getByRole('button', { name: 'Undo last operation' }))
    const dialog = await screen.findByRole('alertdialog', { name: 'Undo merge' })
    expect(dialog).toHaveTextContent('main moves from b2c3d4e to a1b2c3d. 2 commits leave the branch.')
    await userEvent.click(within(dialog).getByRole('button', { name: 'Undo merge' }))

    expect(UndoService.Apply).toHaveBeenCalledWith(REPO, plan)
  })

  it('does nothing when cancelled', async () => {
    vi.mocked(UndoService.Preview).mockResolvedValue(plan)
    renderNav(false)

    await userEvent.click(screen.getByRole('button', { name: 'Undo last operation' }))
    await userEvent.click(within(await screen.findByRole('alertdialog')).getByRole('button', { name: 'Cancel' }))

    expect(UndoService.Apply).not.toHaveBeenCalled()
  })

  it('says why there is nothing to undo', async () => {
    vi.mocked(UndoService.Preview).mockRejectedValue(new Error('Nothing to undo.'))
    renderNav(false)

    await userEvent.click(screen.getByRole('button', { name: 'Undo last operation' }))

    expect(await screen.findByText('Could not undo: Nothing to undo.')).toBeInTheDocument()
  })
})
