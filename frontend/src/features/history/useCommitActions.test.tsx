import { useState } from 'react'
import { render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { BranchService, CommitInfo, CommitService } from '@current-client-bindings/app'
import { Links } from '@current-client-bindings/core/hostlink'
import { DialogProvider } from '../../components/chrome/DialogProvider'
import { useCommitActions } from './useCommitActions'

const REPO = '/repos/app'
const commit = new CommitInfo({ sha: 'abc1234def5678', subject: 'Add the thing' })

// Every action as a button, with the hook's dialogs and message beside them.
function Harness({ onChanged = vi.fn(), hostLinks = null }: { onChanged?: () => void; hostLinks?: Links | null }) {
  const { actionsFor, error, overlays } = useCommitActions(REPO, onChanged, hostLinks)
  const [actions] = useState(() => actionsFor(commit))
  return (
    <>
      {actions.map((a) => (
        <button key={a.id} type="button" onClick={a.run}>
          {a.label}
        </button>
      ))}
      {error && <p>{error}</p>}
      {overlays}
    </>
  )
}

function renderActions(props: { onChanged?: () => void; hostLinks?: Links | null } = {}) {
  render(
    <DialogProvider>
      <Harness {...props} />
    </DialogProvider>,
  )
}

describe('useCommitActions', () => {
  it('lists the actions in one order, with Open on host only for a host', () => {
    renderActions()
    expect(screen.getAllByRole('button').map((b) => b.textContent)).toEqual([
      'Check out',
      'Branch here',
      'Cherry-pick',
      'Revert',
      'Reset to here…',
      'Export patch…',
      'Copy SHA',
    ])
  })

  it('adds Open on host when a remote points at a host', () => {
    renderActions({ hostLinks: new Links({ commit: 'https://github.com/o/r/commit/' }) })
    expect(screen.getByRole('button', { name: 'Open on host' })).toBeInTheDocument()
  })

  it('detaches HEAD at the commit after explaining what that means', async () => {
    vi.mocked(BranchService.CheckoutCommit).mockResolvedValue()
    const onChanged = vi.fn()
    renderActions({ onChanged })

    await userEvent.click(screen.getByRole('button', { name: 'Check out' }))
    const dialog = await screen.findByRole('alertdialog', { name: 'Check out commit' })
    expect(dialog).toHaveTextContent('Check out abc1234? HEAD will be detached')
    await userEvent.click(within(dialog).getByRole('button', { name: 'Check out' }))

    await vi.waitFor(() => expect(onChanged).toHaveBeenCalled())
    expect(BranchService.CheckoutCommit).toHaveBeenCalledWith(REPO, commit.sha)
  })

  it('does nothing when the check out is cancelled', async () => {
    renderActions()

    await userEvent.click(screen.getByRole('button', { name: 'Check out' }))
    const dialog = await screen.findByRole('alertdialog', { name: 'Check out commit' })
    await userEvent.click(within(dialog).getByRole('button', { name: 'Cancel' }))

    expect(BranchService.CheckoutCommit).not.toHaveBeenCalled()
  })

  it('resets through one dialog that names each mode, with hard marked destructive', async () => {
    vi.mocked(BranchService.CurrentBranchStatus).mockResolvedValue({ current: 'main' } as never)
    vi.mocked(CommitService.Reset).mockResolvedValue()
    const onChanged = vi.fn()
    renderActions({ onChanged })

    await userEvent.click(screen.getByRole('button', { name: 'Reset to here…' }))
    const dialog = await screen.findByRole('dialog', { name: 'Reset to abc1234' })
    expect(await within(dialog).findByText(/main moves to abc1234/)).toBeInTheDocument()
    expect(within(dialog).getByRole('radio', { name: /Mixed/ })).toBeChecked()
    expect(within(dialog).getByText('git reset --soft abc1234')).toBeInTheDocument()
    expect(within(dialog).getByRole('button', { name: 'Reset (mixed)' })).toHaveClass('dialog-button-primary')

    await userEvent.click(within(dialog).getByRole('radio', { name: /Hard/ }))
    const confirm = within(dialog).getByRole('button', { name: 'Reset (hard)' })
    expect(confirm).toHaveClass('dialog-button-destructive')
    await userEvent.click(confirm)

    expect(CommitService.Reset).toHaveBeenCalledWith(REPO, commit.sha, 'hard')
    await vi.waitFor(() => expect(onChanged).toHaveBeenCalled())
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
  })

  it('says why a cherry-pick failed and refreshes, since a conflict is left behind', async () => {
    vi.mocked(CommitService.CherryPick).mockRejectedValue(new Error('Conflicts in app.ts.'))
    const onChanged = vi.fn()
    renderActions({ onChanged })

    await userEvent.click(screen.getByRole('button', { name: 'Cherry-pick' }))

    expect(await screen.findByText('Could not cherry-pick: Conflicts in app.ts.')).toBeInTheDocument()
    expect(onChanged).toHaveBeenCalled()
  })
})
