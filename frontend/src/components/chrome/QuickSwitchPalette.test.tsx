import { useState } from 'react'
import { render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import {
  BisectService,
  BranchService,
  PatchService,
  RemoteService,
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
        onDismissFeedback={workspace.dismissFeedback}
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

describe('QuickSwitchPalette at rest', () => {
  it('lists the checked-out branch first, at most eight local branches, and no settings until typed for', async () => {
    const names = ['a', 'b', 'c', 'd', 'e', 'f', 'g', 'h', 'i']
    vi.mocked(BranchService.ListLocal).mockResolvedValue([
      ...names.map((name) => ({ name, current: false })),
      { name: 'main', current: true },
    ] as never)
    vi.mocked(BranchService.ListRemote).mockResolvedValue(['origin/zed'])
    vi.mocked(RemoteService.List).mockResolvedValue([{ name: 'origin', fetchUrl: 'git@example.com:g/r.git' }] as never)
    renderPalette()

    await within(palette()).findByRole('option', { name: 'main, current branch' })
    const branches = within(palette())
      .getAllByRole('option')
      .filter((o) => o.getAttribute('data-value')?.startsWith('branch:'))
    expect(branches).toHaveLength(8)
    expect(branches[0]).toHaveAccessibleName('main, current branch')
    expect(within(palette()).queryByRole('option', { name: /origin\/zed/ })).not.toBeInTheDocument()
    expect(within(palette()).queryByRole('option', { name: 'Settings: Theme' })).not.toBeInTheDocument()

    await userEvent.type(screen.getByRole('combobox'), 'the')
    expect(await within(palette()).findByRole('option', { name: 'Settings: Theme' })).toBeInTheDocument()
  })
})

describe('QuickSwitchPalette Workspace actions', () => {
  it('lists every sidebar action by its sidebar name', () => {
    renderPalette()

    const sidebar = [...document.querySelectorAll('.nav-pane-action:not([aria-expanded])')].map((b) =>
      b.getAttribute('title'),
    )
    const options = within(palette())
      .getAllByRole('option')
      .map((o) => o.getAttribute('aria-label'))
    expect(sidebar).toEqual([
      'Open Terminal',
      expect.stringMatching(/^(Reveal in|Open in File Manager)/),
      'Open in VS Code',
      'Undo last operation',
      'Reflog',
      'Bisect',
      'Import patch…',
    ])
    for (const name of sidebar) expect(options).toContain(name)
    expect(options).toContain('Check integrity')
    expect(options).toContain('Run cleanup')
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

    expect((await screen.findByText('Could not undo: Nothing to undo.')).parentElement).toHaveClass(
      'nav-pane-feedback-error',
    )
  })

  it('opens the Reflog window', async () => {
    vi.mocked(ReflogService.GetReflog).mockResolvedValue([])
    renderPalette()

    await userEvent.click(paletteAction('Reflog'))

    const reflog = await screen.findByRole('dialog', { name: 'Reflog' })
    expect(await within(reflog).findByText('No reflog entries.')).toBeInTheDocument()
    expect(ReflogService.GetReflog).toHaveBeenCalledWith(REPO, expect.any(Number))
  })

  it('acts on a reflog entry from its row', async () => {
    vi.mocked(ReflogService.GetReflog).mockResolvedValue([
      {
        selector: 'HEAD@{0}',
        sha: 'abc1234def',
        action: 'commit: Add thing',
        date: '2026-10-01T00:00:00Z',
        subject: 'Add thing',
      },
    ])
    vi.mocked(BranchService.CheckoutCommit).mockResolvedValue()
    renderPalette()

    await userEvent.click(paletteAction('Reflog'))
    const reflog = await screen.findByRole('dialog', { name: 'Reflog' })
    expect(reflog).toHaveClass('diff-sheet')
    await userEvent.click(await within(reflog).findByRole('button', { name: 'More actions for abc1234' }))
    expect([...document.querySelectorAll('.context-menu button')].map((b) => b.textContent)).toEqual([
      'Check out',
      'Branch here',
      'Cherry-pick',
      'Revert',
      'Reset to here…',
      'Export patch…',
      'Copy SHA',
    ])
    await userEvent.click(screen.getByRole('button', { name: 'Check out' }))
    await userEvent.click(within(await screen.findByRole('alertdialog')).getByRole('button', { name: 'Check out' }))
    expect(BranchService.CheckoutCommit).toHaveBeenCalledWith(REPO, 'abc1234def')
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

    await userEvent.click(paletteAction('Import patch…'))

    expect((await screen.findByText('Patch applied to the working tree.')).parentElement).toHaveClass(
      'nav-pane-feedback-notice',
    )
    expect(PatchService.ImportPatch).toHaveBeenCalledWith(REPO)
  })
})
