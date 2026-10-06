import type { ReactNode } from 'react'
import { act, renderHook, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { BranchService } from '@current-client-bindings/app'
import { DialogProvider } from '../../components/chrome/DialogProvider'
import { useCommitActions } from './useCommitActions'

const REPO = '/repos/app'
const SHA = 'abc1234def5678'

function wrapper({ children }: { children: ReactNode }) {
  return <DialogProvider>{children}</DialogProvider>
}

describe('useCommitActions checkoutCommit', () => {
  it('detaches HEAD at the commit after explaining what that means', async () => {
    vi.mocked(BranchService.CheckoutCommit).mockResolvedValue()
    const onChanged = vi.fn()
    const { result } = renderHook(() => useCommitActions(REPO, onChanged), { wrapper })

    act(() => void result.current.checkoutCommit(SHA))
    const dialog = await screen.findByRole('alertdialog', { name: 'Check out commit' })
    expect(dialog).toHaveTextContent('Check out abc1234? HEAD will be detached')
    await userEvent.click(within(dialog).getByRole('button', { name: 'Check out' }))

    await vi.waitFor(() => expect(onChanged).toHaveBeenCalled())
    expect(BranchService.CheckoutCommit).toHaveBeenCalledWith(REPO, SHA)
  })

  it('does nothing when cancelled', async () => {
    const { result } = renderHook(() => useCommitActions(REPO, vi.fn()), { wrapper })

    act(() => void result.current.checkoutCommit(SHA))
    const dialog = await screen.findByRole('alertdialog', { name: 'Check out commit' })
    await userEvent.click(within(dialog).getByRole('button', { name: 'Cancel' }))

    expect(BranchService.CheckoutCommit).not.toHaveBeenCalled()
  })
})
