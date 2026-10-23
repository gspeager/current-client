import { useState } from 'react'
import { render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import {
  BisectService,
  PatchService,
  ReflogService,
  UndoService,
  type UndoPlanInfo,
} from '@current-client-bindings/app'
import type { useRepositoryLifecycle } from '../../features/repositories/useRepositoryLifecycle'
import type { useWorkingTree } from '../../features/working-copy/useWorkingTree'
import { DialogProvider } from './DialogProvider'
import NavPane from './NavPane'
import QuickSwitchPalette from './QuickSwitchPalette'
import { useWorkspaceActions } from './useWorkspaceActions'

const REPO = '/repos/app'

const repo = { recentRepos: [], openRecent: vi.fn() } as unknown as ReturnType<typeof useRepositoryLifecycle>

const workingTree = {
  files: [],
  conflicted: [],
  staged: [],
  unstaged: [],
  error: null,
  loadStatus: vi.fn(),
} as unknown as ReturnType<typeof useWorkingTree>

// The palette and the sidebar sharing one set of Workspace actions, as in App.
function Shell({ onBranchChanged = vi.fn() }: { onBranchChanged?: () => void }) {
  const [paletteOpen, setPaletteOpen] = useState(true)
  const workspace = useWorkspaceActions({
    repoPath: REPO,
    onBranchChanged,
    onOpenCommit: vi.fn(),
    onFailure: vi.fn(),
  })
  return (
    <>
      <NavPane
        repoPath={REPO}
        repoVersion={0}
        workingTree={workingTree}
        workspaceActions={workspace.actions}
        workspaceFeedback={workspace.feedback}
        onBranchChanged={onBranchChanged}
        onFetched={vi.fn()}
        onOpenRepository={vi.fn()}
        onOpenHeadCommit={vi.fn()}
        width={240}
        collapsed={false}
        onCollapsedChange={vi.fn()}
      />
      {paletteOpen && (
        <QuickSwitchPalette
          repoPath={REPO}
          repo={repo}
          onClose={() => setPaletteOpen(false)}
          workspaceActions={workspace.actions}
          onOpenSettings={vi.fn()}
          onPull={vi.fn()}
          onPullMerge={vi.fn()}
          onPullRebase={vi.fn()}
          onPush={vi.fn()}
          onFetchAll={vi.fn()}
          onOpenFile={vi.fn()}
          onSearchFiles={vi.fn()}
          onOpenCommit={vi.fn()}
        />
      )}
      {workspace.modals}
    </>
  )
}

function renderPalette(onBranchChanged?: () => void) {
  render(
    <DialogProvider>
      <Shell onBranchChanged={onBranchChanged} />
    </DialogProvider>,
  )
}

const palette = () => screen.getByRole('dialog', { name: 'Go to anything' })
const paletteAction = (name: string) => within(palette()).getByRole('option', { name })

beforeAll(() => {
  // cmdk measures its list and scrolls the selected item into view; jsdom has neither.
  window.ResizeObserver = class {
    observe() {}
    unobserve() {}
    disconnect() {}
  }
  Element.prototype.scrollIntoView = vi.fn()
})

describe('QuickSwitchPalette Workspace actions', () => {
  it('lists every sidebar action by its sidebar name', () => {
    renderPalette()

    const sidebar = [...document.querySelectorAll('.nav-pane-workspace button')].map((b) =>
      b.getAttribute('aria-label'),
    )
    const options = within(palette())
      .getAllByRole('option')
      .map((o) => o.getAttribute('aria-label'))
    expect(sidebar).toEqual([
      'Open Terminal',
      expect.stringMatching(/^(Reveal in|Open in File Manager)/),
      'Open in Editor',
      'Undo last operation',
      'Reflog',
      'Bisect',
      'Import Patch…',
    ])
    for (const name of sidebar) expect(options).toContain(name)
  })

  it('previews the undo and applies it once confirmed', async () => {
    const plan = {
      operation: 'rebase',
      detail: '',
      branch: 'feature',
      from: 'b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0c1',
      to: 'a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0',
      switchTo: '',
      mode: 'keep',
      removed: 1,
      restored: 1,
      pushed: false,
    } as UndoPlanInfo
    vi.mocked(UndoService.Preview).mockResolvedValue(plan)
    vi.mocked(UndoService.Apply).mockResolvedValue()
    const onBranchChanged = vi.fn()
    renderPalette(onBranchChanged)

    await userEvent.click(paletteAction('Undo last operation'))

    const dialog = await screen.findByRole('alertdialog', { name: 'Undo rebase' })
    expect(dialog).toHaveTextContent(
      'feature moves from b2c3d4e to a1b2c3d. 1 commit leaves the branch. 1 commit comes back.',
    )
    expect(screen.queryByRole('dialog', { name: 'Go to anything' })).not.toBeInTheDocument()
    await userEvent.click(within(dialog).getByRole('button', { name: 'Undo rebase' }))

    await vi.waitFor(() => expect(UndoService.Apply).toHaveBeenCalledWith(REPO, plan))
    await vi.waitFor(() => expect(onBranchChanged).toHaveBeenCalled())
  })

  it('shows in the sidebar why there is nothing to undo', async () => {
    vi.mocked(UndoService.Preview).mockRejectedValue(new Error('Nothing to undo.'))
    renderPalette()

    await userEvent.click(paletteAction('Undo last operation'))

    expect(await screen.findByText('Could not undo: Nothing to undo.')).toHaveClass('nav-pane-error')
  })

  it('opens the Reflog window', async () => {
    vi.mocked(ReflogService.GetReflog).mockResolvedValue([])
    renderPalette()

    await userEvent.click(paletteAction('Reflog'))

    const reflog = await screen.findByRole('dialog', { name: 'Reflog' })
    expect(await within(reflog).findByText('No reflog entries.')).toBeInTheDocument()
    expect(ReflogService.GetReflog).toHaveBeenCalledWith(REPO, expect.any(Number))
  })

  it('opens the Bisect wizard', async () => {
    vi.mocked(BisectService.IsActive).mockResolvedValue(false)
    renderPalette()

    await userEvent.click(paletteAction('Bisect'))

    const bisect = await screen.findByRole('dialog', { name: 'Bisect' })
    expect(await within(bisect).findByRole('button', { name: 'Start bisect' })).toBeInTheDocument()
  })

  it('reports an imported patch in the sidebar', async () => {
    vi.mocked(PatchService.ImportPatch).mockResolvedValue('Patch applied to the working tree.')
    renderPalette()

    await userEvent.click(paletteAction('Import Patch…'))

    expect(await screen.findByText('Patch applied to the working tree.')).toHaveClass('nav-pane-success')
    expect(PatchService.ImportPatch).toHaveBeenCalledWith(REPO)
  })
})
