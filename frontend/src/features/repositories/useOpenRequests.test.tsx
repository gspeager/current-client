import { act, render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { RepositoryService } from '@current-client-bindings/app'
import { DialogProvider } from '../../components/chrome/DialogProvider'
import { useOpenRequests } from './useOpenRequests'

function Harness({ repo }: { repo: Parameters<typeof useOpenRequests>[0] }) {
  useOpenRequests(repo)
  return null
}

function renderWith(restored = true) {
  const repo = { restored, openFromOutside: vi.fn(), showOpenProblem: vi.fn() }
  const view = render(
    <DialogProvider>
      <Harness repo={repo} />
    </DialogProvider>,
  )
  return { repo, ...view }
}

const take = () => vi.mocked(RepositoryService.TakeOpenRequests)

describe('useOpenRequests', () => {
  it('waits for tabs to be restored, then opens a path straight away', async () => {
    take().mockResolvedValue([{ path: '/repos/a', confirm: false, problem: '' }])
    const repo = { restored: false, openFromOutside: vi.fn(), showOpenProblem: vi.fn() }
    const { rerender } = render(
      <DialogProvider>
        <Harness repo={repo} />
      </DialogProvider>,
    )
    expect(take()).not.toHaveBeenCalled()

    const ready = { ...repo, restored: true }
    rerender(
      <DialogProvider>
        <Harness repo={ready} />
      </DialogProvider>,
    )
    await vi.waitFor(() => expect(ready.openFromOutside).toHaveBeenCalledWith('/repos/a'))
  })

  it('shows the full path and asks before opening what a link names', async () => {
    const user = userEvent.setup()
    take().mockResolvedValue([{ path: '/downloads/evil', confirm: true, problem: '' }])
    const { repo } = renderWith()

    const dialog = await screen.findByRole('alertdialog', { name: 'Open repository from a link' })
    expect(within(dialog).getByText(/\/downloads\/evil/)).toBeInTheDocument()
    expect(repo.openFromOutside).not.toHaveBeenCalled()

    await user.click(within(dialog).getByRole('button', { name: 'Open' }))
    await vi.waitFor(() => expect(repo.openFromOutside).toHaveBeenCalledWith('/downloads/evil'))
  })

  it('opens nothing when the question is declined', async () => {
    const user = userEvent.setup()
    take().mockResolvedValue([{ path: '/downloads/evil', confirm: true, problem: '' }])
    const { repo } = renderWith()

    const dialog = await screen.findByRole('alertdialog')
    await user.click(within(dialog).getByRole('button', { name: 'Cancel' }))
    await act(async () => {})
    expect(repo.openFromOutside).not.toHaveBeenCalled()
  })

  it('says why a link is no use', async () => {
    take().mockResolvedValue([{ path: '', confirm: false, problem: "This link isn't one Current Client can open." }])
    const { repo } = renderWith()
    await vi.waitFor(() =>
      expect(repo.showOpenProblem).toHaveBeenCalledWith("This link isn't one Current Client can open."),
    )
    expect(repo.openFromOutside).not.toHaveBeenCalled()
  })
})
