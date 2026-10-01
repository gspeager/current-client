import type { ReactNode } from 'react'
import { act, renderHook, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { BranchService, RemoteService, type BranchStatusInfo } from '@current-client-bindings/app'
import { DialogProvider } from '../../components/chrome/DialogProvider'
import { useRemoteSync } from './useRemoteSync'

const REPO = '/repos/app'

function wrapper({ children }: { children: ReactNode }) {
  return <DialogProvider>{children}</DialogProvider>
}

// A push that stays in flight until cancelled, like a Wails CancellablePromise.
function pendingPush() {
  let reject: (err: Error) => void = () => {}
  const promise = new Promise<void>((_, r) => {
    reject = r
  }) as Promise<void> & { cancel: () => Promise<void> }
  const cancel = vi.fn(() => {
    const err = new Error('cancelled')
    err.name = 'CancelError'
    reject(err)
    return Promise.resolve()
  })
  promise.cancel = cancel
  return { promise, cancel }
}

describe('useRemoteSync', () => {
  it('pushes and reports the sync', async () => {
    vi.mocked(RemoteService.Push).mockResolvedValue()
    const onSynced = vi.fn()
    const { result } = renderHook(() => useRemoteSync(REPO, onSynced), { wrapper })

    await act(async () => result.current.push())

    expect(RemoteService.Push).toHaveBeenCalledWith(REPO)
    expect(onSynced).toHaveBeenCalled()
    expect(result.current.pushOp.error).toBeNull()
  })

  it('sets the upstream after confirming when the branch has none', async () => {
    vi.mocked(RemoteService.Push).mockRejectedValue(new Error('This branch has no upstream configured.'))
    vi.mocked(BranchService.CurrentBranchStatus).mockResolvedValue({ current: 'feature' } as BranchStatusInfo)
    vi.mocked(RemoteService.List).mockResolvedValue([{ name: 'origin', fetchUrl: '', pushUrl: '' }])
    vi.mocked(RemoteService.PushSetUpstream).mockResolvedValue()
    const onSynced = vi.fn()
    const { result } = renderHook(() => useRemoteSync(REPO, onSynced), { wrapper })

    act(() => result.current.push())
    const dialog = await screen.findByRole('alertdialog', { name: 'Set upstream' })
    await userEvent.click(within(dialog).getByRole('button', { name: 'Push' }))

    await vi.waitFor(() => expect(onSynced).toHaveBeenCalled())
    expect(RemoteService.PushSetUpstream).toHaveBeenCalledWith(REPO, 'origin', 'feature')
    expect(result.current.pushOp.error).toBeNull()
  })

  it('stops a push when cancelled, without reporting an error', async () => {
    const { promise, cancel } = pendingPush()
    vi.mocked(RemoteService.Push).mockReturnValue(promise as ReturnType<typeof RemoteService.Push>)
    const onSynced = vi.fn()
    const { result } = renderHook(() => useRemoteSync(REPO, onSynced), { wrapper })

    act(() => result.current.push())
    expect(result.current.pushOp.running).toBe(true)
    await act(async () => result.current.pushOp.cancel())

    expect(cancel).toHaveBeenCalled()
    expect(result.current.pushOp.running).toBe(false)
    expect(result.current.pushOp.error).toBeNull()
    expect(onSynced).not.toHaveBeenCalled()
  })

  it('reports why a push failed', async () => {
    vi.mocked(RemoteService.Push).mockRejectedValue(new Error('The remote stopped responding.'))
    const { result } = renderHook(() => useRemoteSync(REPO, vi.fn()), { wrapper })

    await act(async () => result.current.push())

    expect(result.current.pushOp.error).toBe('The remote stopped responding.')
  })
})
