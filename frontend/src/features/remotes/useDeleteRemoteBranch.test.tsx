import type { ReactNode } from 'react'
import { act, renderHook, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { RemoteService } from '@current-client-bindings/app'
import { DialogProvider } from '../../components/chrome/DialogProvider'
import { useDeleteRemoteBranch } from './useDeleteRemoteBranch'

const REPO = '/repos/app'

function wrapper({ children }: { children: ReactNode }) {
  return <DialogProvider>{children}</DialogProvider>
}

describe('useDeleteRemoteBranch', () => {
  it('does nothing when the confirmation is cancelled', async () => {
    const onDeleted = vi.fn()
    const { result } = renderHook(() => useDeleteRemoteBranch(REPO, onDeleted), { wrapper })

    act(() => void result.current.deleteRemoteBranch('origin/feature'))
    const dialog = await screen.findByRole('alertdialog', { name: 'Delete on remote' })
    expect(dialog).toHaveTextContent('Delete origin/feature from its remote?')
    await userEvent.click(within(dialog).getByRole('button', { name: 'Cancel' }))

    expect(RemoteService.DeleteRemoteBranch).not.toHaveBeenCalled()
    expect(onDeleted).not.toHaveBeenCalled()
  })

  it('reports why the remote refused', async () => {
    vi.mocked(RemoteService.DeleteRemoteBranch).mockRejectedValue(new Error('The remote refused the push.'))
    const onDeleted = vi.fn()
    const { result } = renderHook(() => useDeleteRemoteBranch(REPO, onDeleted), { wrapper })

    act(() => void result.current.deleteRemoteBranch('origin/main'))
    const dialog = await screen.findByRole('alertdialog', { name: 'Delete on remote' })
    await userEvent.click(within(dialog).getByRole('button', { name: 'Delete' }))

    await vi.waitFor(() => expect(result.current.error).toBe('The remote refused the push.'))
    expect(onDeleted).not.toHaveBeenCalled()
  })
})
