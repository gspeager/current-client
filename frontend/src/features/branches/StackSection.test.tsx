import { render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { RemoteService, StackService, type StackListInfo } from '@current-client-bindings/app'
import { useState } from 'react'
import { DialogProvider } from '../../components/chrome/DialogProvider'
import StackSection from './StackSection'

const layer = (branch: string, parent: string, depth: number, marks = {}) => ({
  branch,
  parent,
  depth,
  commits: 1,
  missing: false,
  merged: false,
  behind: false,
  ...marks,
})

const list = (overrides: Partial<StackListInfo> = {}): StackListInfo => ({
  stacks: [
    {
      root: 'bottom',
      base: 'main',
      needsRestack: true,
      layers: [layer('bottom', 'main', 0, { merged: true }), layer('top', 'bottom', 1, { behind: true })],
    },
  ],
  canUndo: false,
  ...overrides,
})

function Harness({
  info,
  onChanged,
  onCheckout,
}: {
  info: StackListInfo
  onChanged: () => void
  onCheckout: () => void
}) {
  const [restack, setRestack] = useState<string | null>(null)
  return (
    <DialogProvider>
      <StackSection
        repoPath="/repo"
        list={info}
        current="top"
        restack={restack}
        onRestackChange={setRestack}
        onCheckout={onCheckout}
        onRowMenu={() => undefined}
        onChanged={onChanged}
      />
    </DialogProvider>
  )
}

function renderStacks(info = list()) {
  const onChanged = vi.fn()
  const onCheckout = vi.fn()
  render(<Harness info={info} onChanged={onChanged} onCheckout={onCheckout} />)
  return { onChanged, onCheckout }
}

const preview = (overrides = {}) => ({
  layers: [
    { branch: 'bottom', onto: 'main', commits: [], droppedMerges: 0, missing: false, merged: true, moves: false },
    {
      branch: 'top',
      onto: 'main',
      commits: [{ sha: 'abcdef1234', subject: 'top work', outcome: 'clean', files: [] }],
      droppedMerges: 0,
      missing: false,
      merged: false,
      moves: true,
    },
  ],
  predicted: true,
  dirty: false,
  blocked: '',
  ...overrides,
})

describe('StackSection', () => {
  it('shows each layer, indented, with what has happened to it', () => {
    renderStacks()
    expect(screen.getByRole('group', { name: 'Stack on main' })).toBeInTheDocument()
    expect(screen.getByText('Merged')).toBeInTheDocument()
    expect(screen.getByText('Behind')).toBeInTheDocument()
    expect(screen.getByText('top').closest('li')).toHaveStyle({ paddingLeft: 'calc(var(--space-md) * 1)' })
  })

  it('previews the whole stack, then restacks and reports what stopped and left', async () => {
    const user = userEvent.setup()
    vi.mocked(StackService.Preview).mockResolvedValue(preview())
    vi.mocked(StackService.Restack).mockResolvedValue({ stopped: 'top', left: ['bottom'], moved: 1 })
    const { onChanged } = renderStacks()

    await user.click(screen.getByRole('button', { name: 'Restack' }))
    const dialog = await screen.findByRole('dialog', { name: 'Restack bottom' })
    expect(await within(dialog).findByText(/Merged into main/)).toBeInTheDocument()
    expect(within(dialog).getByText('top work')).toBeInTheDocument()
    expect(StackService.Restack).not.toHaveBeenCalled()

    await user.click(within(dialog).getByRole('button', { name: 'Restack' }))
    expect(await screen.findByText(/Stopped on top.*bottom left the stack/)).toBeInTheDocument()
    expect(StackService.Restack).toHaveBeenCalledWith('/repo', 'bottom')
    expect(onChanged).toHaveBeenCalled()
  })

  it("doesn't let a blocked or dirty restack start", async () => {
    const user = userEvent.setup()
    vi.mocked(StackService.Preview).mockResolvedValue(preview({ blocked: 'top is checked out in the worktree at /x.' }))
    renderStacks()
    await user.click(screen.getByRole('button', { name: 'Restack' }))
    const dialog = await screen.findByRole('dialog')
    expect(await within(dialog).findByText(/checked out in the worktree/)).toBeInTheDocument()
    expect(within(dialog).getByRole('button', { name: 'Restack' })).toBeDisabled()
  })

  it('has nothing to restack on a stack in step', () => {
    renderStacks(list({ stacks: [{ root: 'a', base: 'main', needsRestack: false, layers: [layer('a', 'main', 0)] }] }))
    expect(screen.getByRole('button', { name: 'Restack' })).toBeDisabled()
  })

  it('undoes a restack, showing a refusal', async () => {
    const user = userEvent.setup()
    vi.mocked(StackService.UndoRestack).mockRejectedValue(new Error('top has changed since the restack'))
    const { onChanged } = renderStacks(list({ canUndo: true }))
    await user.click(screen.getByRole('button', { name: 'Undo restack' }))
    expect(await screen.findByRole('alert')).toHaveTextContent('top has changed')
    expect(onChanged).toHaveBeenCalled()
  })

  it('confirms Push all, naming the branches and the lease', async () => {
    const user = userEvent.setup()
    vi.mocked(RemoteService.List).mockResolvedValue([{ name: 'origin' } as never])
    vi.mocked(StackService.PushAll).mockResolvedValue()
    renderStacks()

    await user.click(screen.getByRole('button', { name: 'Push all' }))
    const dialog = await screen.findByRole('alertdialog', { name: 'Push all' })
    expect(within(dialog).getByText(/bottom, top to origin with --force-with-lease/)).toBeInTheDocument()
    expect(StackService.PushAll).not.toHaveBeenCalled()

    await user.click(within(dialog).getByRole('button', { name: 'Push all' }))
    expect(await screen.findByText('Pushed 2 branches.')).toBeInTheDocument()
    expect(StackService.PushAll).toHaveBeenCalledWith('/repo', 'bottom', 'origin', null)
  })

  it('checks out a layer, but not the current one', async () => {
    const user = userEvent.setup()
    const { onCheckout } = renderStacks()
    await user.click(screen.getByText('top'))
    expect(onCheckout).not.toHaveBeenCalled()
    await user.click(screen.getByText('bottom'))
    expect(onCheckout).toHaveBeenCalledWith('bottom')
  })
})
