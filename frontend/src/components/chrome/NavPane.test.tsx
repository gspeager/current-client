import { render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import {
  DashboardService,
  HistoryService,
  PatchService,
  PlatformService,
  SettingsService,
  UndoService,
  type UndoPlanInfo,
} from '@current-client-bindings/app'
import type { useWorkingTree } from '../../features/working-copy/useWorkingTree'
import { DialogProvider } from './DialogProvider'
import NavPane from './NavPane'
import { useWorkspaceActions } from './useWorkspaceActions'

const REPO = '/repos/app'

const workingTree = {
  files: [],
  conflicted: [],
  staged: [],
  unstaged: [],
  error: null,
  loadStatus: vi.fn(),
} as unknown as ReturnType<typeof useWorkingTree>

interface ShellProps {
  collapsed: boolean
  onCollapsedChange: (next: boolean) => void
  onOpenHeadCommit?: (sha: string) => void
}

// The sidebar with its Workspace actions wired the way App wires them.
function Shell({ collapsed, onCollapsedChange, onOpenHeadCommit = vi.fn() }: ShellProps) {
  const workspace = useWorkspaceActions({
    repoPath: REPO,
    onBranchChanged: vi.fn(),
    onOpenCommit: vi.fn(),
    onFailure: () => onCollapsedChange(false),
  })
  return (
    <>
      <NavPane
        repoPath={REPO}
        repoVersion={0}
        workingTree={workingTree}
        workspaceActions={workspace.actions}
        workspaceFeedback={workspace.feedback}
        onDismissFeedback={workspace.dismissFeedback}
        onBranchChanged={vi.fn()}
        onFetched={vi.fn()}
        onOpenRepository={vi.fn()}
        onOpenHeadCommit={onOpenHeadCommit}
        width={240}
        collapsed={collapsed}
        onCollapsedChange={onCollapsedChange}
      />
      {workspace.modals}
    </>
  )
}

function renderNav(collapsed: boolean, onCollapsedChange = vi.fn(), onOpenHeadCommit = vi.fn()) {
  const ui = (isCollapsed: boolean) => (
    <DialogProvider>
      <Shell collapsed={isCollapsed} onCollapsedChange={onCollapsedChange} onOpenHeadCommit={onOpenHeadCommit} />
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

  it('separates the open, tools, maintenance and section groups with rules', () => {
    renderNav(true)

    const rail = screen.getByRole('navigation', { name: 'Repository' })
    expect(rail.querySelectorAll('.nav-pane-rail-divider')).toHaveLength(4)
  })

  it('expands to a section picked from the rail and opens it', async () => {
    vi.mocked(SettingsService.SetSidebarSection).mockResolvedValue()
    const { onCollapsedChange } = renderNav(true)

    await userEvent.click(screen.getByRole('button', { name: 'Show Stash' }))

    expect(onCollapsedChange).toHaveBeenCalledWith(false)
    expect(SettingsService.SetSidebarSection).toHaveBeenCalledWith(REPO, 'stash', true)
  })

  it('expands so a failed action can show why', async () => {
    vi.mocked(PlatformService.OpenInEditor).mockRejectedValue(new Error('Editor not found.'))
    const { onCollapsedChange, setCollapsed } = renderNav(true)

    await userEvent.click(screen.getByRole('button', { name: 'Open in VS Code' }))
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

  it('leads with the HEAD card, which opens the commit', async () => {
    vi.mocked(HistoryService.GetHistory).mockResolvedValue([
      { sha: 'abc1234def5678', subject: 'Add the thing' },
    ] as never)
    const onOpenHeadCommit = vi.fn()
    renderNav(false, vi.fn(), onOpenHeadCommit)

    const head = await screen.findByRole('button', { name: 'HEAD abc1234 Add the thing' })
    expect(head.parentElement).toHaveClass('nav-pane-head')
    expect(head.querySelector('.head-commit-panel-label')).toHaveTextContent('HEAD')
    expect(head.querySelector('.head-commit-panel-sha')).toHaveTextContent('abc1234')
    expect(head.querySelector('.head-commit-panel-subject')).toHaveTextContent('Add the thing')
    expect(head.querySelector('svg')).not.toBeNull()
    await userEvent.click(head)

    expect(onOpenHeadCommit).toHaveBeenCalledWith('abc1234def5678')
  })

  it('ends with Open in and Tools groups of outlined buttons, and Maintenance behind a disclosure', async () => {
    vi.mocked(PlatformService.OpenInEditor).mockResolvedValue()
    vi.mocked(DashboardService.RunFsckCheck).mockResolvedValue([])
    renderNav(false)

    const groups = [...document.querySelectorAll('.nav-pane-footer .nav-pane-tool-group')]
    expect(groups.map((row) => row.textContent)).toEqual([
      expect.stringMatching(/^Open inTerminal(Finder|File Explorer|File Manager)VS Code$/),
      'ToolsUndoReflogBisectApply patchMaintenance',
    ])
    for (const button of groups.flatMap((g) => [...g.querySelectorAll('button')])) {
      expect(button).toHaveClass('nav-pane-tool')
      expect(button.querySelector('svg')).not.toBeNull()
    }
    expect(screen.queryByRole('button', { name: 'Check integrity' })).not.toBeInTheDocument()

    await userEvent.click(screen.getByRole('button', { name: 'VS Code' }))
    expect(PlatformService.OpenInEditor).toHaveBeenCalledWith(REPO)

    await userEvent.click(screen.getByRole('button', { name: 'Maintenance' }))
    await userEvent.click(screen.getByRole('button', { name: 'Check integrity' }))
    expect(await screen.findByText('Repository is healthy.')).toBeInTheDocument()
  })

  it('lets a message be dismissed', async () => {
    vi.mocked(PatchService.ImportPatch).mockResolvedValue('Patch applied to the working tree.')
    renderNav(false)

    await userEvent.click(screen.getByRole('button', { name: 'Apply patch' }))
    const message = await screen.findByText('Patch applied to the working tree.')
    await userEvent.click(within(message.parentElement!).getByRole('button', { name: 'Dismiss' }))

    expect(screen.queryByText('Patch applied to the working tree.')).not.toBeInTheDocument()
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

    await userEvent.click(screen.getByRole('button', { name: 'Undo' }))
    const dialog = await screen.findByRole('alertdialog', { name: 'Undo merge' })
    expect(dialog).toHaveTextContent('main moves from b2c3d4e to a1b2c3d. 2 commits leave the branch.')
    await userEvent.click(within(dialog).getByRole('button', { name: 'Undo merge' }))

    expect(UndoService.Apply).toHaveBeenCalledWith(REPO, plan)
  })

  it('does nothing when cancelled', async () => {
    vi.mocked(UndoService.Preview).mockResolvedValue(plan)
    renderNav(false)

    await userEvent.click(screen.getByRole('button', { name: 'Undo' }))
    await userEvent.click(within(await screen.findByRole('alertdialog')).getByRole('button', { name: 'Cancel' }))

    expect(UndoService.Apply).not.toHaveBeenCalled()
  })

  it('says why there is nothing to undo', async () => {
    vi.mocked(UndoService.Preview).mockRejectedValue(new Error('Nothing to undo.'))
    renderNav(false)

    await userEvent.click(screen.getByRole('button', { name: 'Undo' }))

    expect(await screen.findByText('Could not undo: Nothing to undo.')).toBeInTheDocument()
  })
})

describe('NavPane import patch', () => {
  it('says how the patch was applied', async () => {
    vi.mocked(PatchService.ImportPatch).mockResolvedValue('Patch applied to the working tree.')
    renderNav(false)

    await userEvent.click(screen.getByRole('button', { name: 'Apply patch' }))

    expect(await screen.findByText('Patch applied to the working tree.')).toBeInTheDocument()
  })

  it('says nothing when the file dialog is cancelled', async () => {
    vi.mocked(PatchService.ImportPatch).mockResolvedValue('')
    renderNav(false)

    await userEvent.click(screen.getByRole('button', { name: 'Apply patch' }))

    await vi.waitFor(() => expect(PatchService.ImportPatch).toHaveBeenCalledWith(REPO))
    expect(screen.queryByText(/Patch applied/)).not.toBeInTheDocument()
  })
})
