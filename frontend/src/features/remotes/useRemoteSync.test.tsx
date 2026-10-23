import type { ReactNode } from 'react'
import { act, renderHook, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import {
  BranchService,
  type BranchStatusInfo,
  PushResult,
  PushWarning,
  RemoteService,
} from '@current-client-bindings/app'
import { DialogProvider } from '../../components/chrome/DialogProvider'
import { DIVERGED, useRemoteSync } from './useRemoteSync'

const REPO = '/repos/app'
const OTHER_REPO = '/repos/site'
const NEEDS_CREDENTIALS =
  'This remote needs credentials. Set up a credential helper, such as Git Credential Manager, for it.'
const AUTH_FAILED = 'Authentication failed. Check the credentials Git uses for this remote.'

function wrapper({ children }: { children: ReactNode }) {
  return <DialogProvider>{children}</DialogProvider>
}

// A push that stays in flight until cancelled, like a Wails CancellablePromise.
function pendingPush() {
  let reject: (err: Error) => void = () => {}
  const promise = new Promise<PushResult>((_, r) => {
    reject = r
  }) as Promise<PushResult> & { cancel: () => Promise<void> }
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
  it('pushes every tag to the primary remote', async () => {
    vi.mocked(RemoteService.List).mockResolvedValue([{ name: 'origin' }] as never)
    vi.mocked(RemoteService.PushTags).mockResolvedValue()
    const onSynced = vi.fn()
    const { result } = renderHook(() => useRemoteSync(REPO, onSynced), { wrapper })

    act(() => result.current.pushTags())

    await vi.waitFor(() => expect(onSynced).toHaveBeenCalled())
    expect(RemoteService.PushTags).toHaveBeenCalledWith(REPO, 'origin', null)
  })

  it('pushes and reports the sync', async () => {
    vi.mocked(RemoteService.Push).mockResolvedValue(new PushResult({}))
    const onSynced = vi.fn()
    const { result } = renderHook(() => useRemoteSync(REPO, onSynced), { wrapper })

    await act(async () => result.current.push())

    expect(RemoteService.Push).toHaveBeenCalledWith(REPO, null)
    expect(onSynced).toHaveBeenCalled()
    expect(result.current.pushOp.error).toBeNull()
  })

  it('asks before pushing to a branch the host protects, and leaves the push unrun on cancel', async () => {
    vi.mocked(RemoteService.PushWarning).mockResolvedValue(
      new PushWarning({
        host: 'gitlab.com',
        branch: 'main',
        pattern: 'main',
        noPush: true,
        noForcePush: true,
        noDelete: true,
      }),
    )
    vi.mocked(RemoteService.Push).mockResolvedValue(new PushResult({}))
    const onSynced = vi.fn()
    const { result } = renderHook(() => useRemoteSync(REPO, onSynced), { wrapper })

    act(() => {
      void result.current.push()
    })
    let dialog = await screen.findByRole('alertdialog', { name: 'Push to a protected branch' })
    expect(dialog).toHaveTextContent('gitlab.com protects main and forbids direct pushes. The push may be rejected.')
    await userEvent.click(within(dialog).getByRole('button', { name: 'Cancel' }))
    expect(RemoteService.Push).not.toHaveBeenCalled()

    act(() => {
      void result.current.push()
    })
    dialog = await screen.findByRole('alertdialog', { name: 'Push to a protected branch' })
    await userEvent.click(within(dialog).getByRole('button', { name: 'Push anyway' }))

    await vi.waitFor(() => expect(onSynced).toHaveBeenCalled())
    expect(RemoteService.PushWarning).toHaveBeenCalledWith(REPO, '')
  })

  it('names what the host forbids in the force-push confirmation', async () => {
    vi.mocked(BranchService.CurrentBranchStatus).mockResolvedValue({
      current: 'release/1.0',
      upstream: 'origin/release/1.0',
    } as BranchStatusInfo)
    vi.mocked(RemoteService.PushWarning).mockResolvedValue(
      new PushWarning({
        host: 'gitlab.com',
        branch: 'release/1.0',
        pattern: 'release/*',
        restrictedPush: true,
        noForcePush: true,
        noDelete: true,
      }),
    )
    const { result } = renderHook(() => useRemoteSync(REPO, vi.fn()), { wrapper })

    act(() => void result.current.forcePush())
    const dialog = await screen.findByRole('alertdialog', { name: 'Force-push' })

    expect(dialog).toHaveTextContent(
      'gitlab.com protects release/1.0 (rule release/*) and forbids pushes except by some roles and force-pushes.',
    )
    expect(RemoteService.PushWarning).toHaveBeenCalledWith(REPO, 'origin/release/1.0')
  })

  it('force-pushes to the upstream it names, after confirming', async () => {
    vi.mocked(BranchService.CurrentBranchStatus).mockResolvedValue({
      current: 'feature',
      upstream: 'team/origin/feat-x',
    } as BranchStatusInfo)
    vi.mocked(RemoteService.ForcePush).mockResolvedValue()
    const onSynced = vi.fn()
    const { result } = renderHook(() => useRemoteSync(REPO, onSynced), { wrapper })

    act(() => void result.current.forcePush())
    const dialog = await screen.findByRole('alertdialog', { name: 'Force-push' })
    expect(dialog).toHaveTextContent('Force-push feature to team/origin/feat-x?')
    await userEvent.click(within(dialog).getByRole('button', { name: 'Force-push' }))

    await vi.waitFor(() => expect(onSynced).toHaveBeenCalled())
    // The backend resolves the remote and branch from the upstream itself.
    expect(RemoteService.ForcePush).toHaveBeenCalledWith(REPO, null)
  })

  it('pulls with rebase and reports the sync', async () => {
    vi.mocked(RemoteService.PullRebase).mockResolvedValue()
    const onSynced = vi.fn()
    const { result } = renderHook(() => useRemoteSync(REPO, onSynced), { wrapper })

    await act(async () => result.current.pullRebase())

    await vi.waitFor(() => expect(onSynced).toHaveBeenCalled())
    expect(RemoteService.PullRebase).toHaveBeenCalledWith(REPO, null)
    expect(RemoteService.Pull).not.toHaveBeenCalled()
  })

  it('pulls with a merge and reports the sync', async () => {
    vi.mocked(RemoteService.PullMerge).mockResolvedValue()
    const onSynced = vi.fn()
    const { result } = renderHook(() => useRemoteSync(REPO, onSynced), { wrapper })

    await act(async () => result.current.pullMerge())

    await vi.waitFor(() => expect(onSynced).toHaveBeenCalled())
    expect(RemoteService.PullMerge).toHaveBeenCalledWith(REPO, null)
    expect(RemoteService.Pull).not.toHaveBeenCalled()
  })

  it('sets the upstream after confirming when the branch has none', async () => {
    vi.mocked(RemoteService.Push).mockRejectedValue(new Error('This branch has no upstream configured.'))
    vi.mocked(BranchService.CurrentBranchStatus).mockResolvedValue({ current: 'feature' } as BranchStatusInfo)
    vi.mocked(RemoteService.List).mockResolvedValue([{ name: 'origin', fetchUrl: '', pushUrl: '' }])
    vi.mocked(RemoteService.PushSetUpstream).mockResolvedValue(new PushResult({}))
    const onSynced = vi.fn()
    const { result } = renderHook(() => useRemoteSync(REPO, onSynced), { wrapper })

    act(() => {
      void result.current.push()
    })
    const dialog = await screen.findByRole('alertdialog', { name: 'Set upstream' })
    await userEvent.click(within(dialog).getByRole('button', { name: 'Push' }))

    await vi.waitFor(() => expect(onSynced).toHaveBeenCalled())
    expect(RemoteService.PushSetUpstream).toHaveBeenCalledWith(REPO, 'origin', 'feature', null)
    expect(result.current.pushOp.error).toBeNull()
  })

  it('asks the user to sign in when the remote needs credentials, then retries with them', async () => {
    vi.mocked(RemoteService.Push)
      .mockRejectedValueOnce(new Error(NEEDS_CREDENTIALS))
      .mockRejectedValueOnce(new Error(AUTH_FAILED))
      .mockResolvedValueOnce(new PushResult({}))
    vi.mocked(RemoteService.CredentialHelper).mockResolvedValue('osxkeychain')
    const onSynced = vi.fn()
    const { result } = renderHook(() => useRemoteSync(REPO, onSynced), { wrapper })

    act(() => {
      void result.current.push()
    })
    let dialog = await screen.findByRole('dialog', { name: 'Sign in to remote' })
    expect(await within(dialog).findByText(/osxkeychain credential helper/)).toBeInTheDocument()
    await userEvent.type(within(dialog).getByLabelText('Username'), 'alice')
    await userEvent.type(within(dialog).getByLabelText('Password or token'), 'wrong{Enter}')

    dialog = await screen.findByRole('dialog', { name: 'Sign in to remote' })
    expect(within(dialog).getByRole('alert')).toHaveTextContent('didn’t accept')
    await userEvent.type(within(dialog).getByLabelText('Username'), 'alice')
    await userEvent.type(within(dialog).getByLabelText('Password or token'), 'ghp_token{Enter}')

    await vi.waitFor(() => expect(onSynced).toHaveBeenCalled())
    expect(vi.mocked(RemoteService.Push).mock.calls).toEqual([
      [REPO, null],
      [REPO, { username: 'alice', password: 'wrong' }],
      [REPO, { username: 'alice', password: 'ghp_token' }],
    ])
    expect(result.current.pushOp.error).toBeNull()
  })

  it('reports the original error when sign-in is cancelled', async () => {
    vi.mocked(RemoteService.Push).mockRejectedValue(new Error(NEEDS_CREDENTIALS))
    vi.mocked(RemoteService.CredentialHelper).mockResolvedValue('')
    const { result } = renderHook(() => useRemoteSync(REPO, vi.fn()), { wrapper })

    act(() => {
      void result.current.push()
    })
    const dialog = await screen.findByRole('dialog', { name: 'Sign in to remote' })
    expect(await within(dialog).findByText(/used once and not saved/)).toBeInTheDocument()
    await userEvent.click(within(dialog).getByRole('button', { name: 'Cancel' }))

    await vi.waitFor(() => expect(result.current.pushOp.error).toBe(NEEDS_CREDENTIALS))
    expect(RemoteService.Push).toHaveBeenCalledTimes(1)
    expect(result.current.pushOp.running).toBe(false)
  })

  it('stops a push when cancelled, without reporting an error', async () => {
    const { promise, cancel } = pendingPush()
    vi.mocked(RemoteService.Push).mockReturnValue(promise as unknown as ReturnType<typeof RemoteService.Push>)
    const onSynced = vi.fn()
    const { result } = renderHook(() => useRemoteSync(REPO, onSynced), { wrapper })

    await act(async () => result.current.push())
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

  it('keeps a failed pull with its repository, and offers the choice for that repository only', async () => {
    vi.mocked(RemoteService.Pull).mockRejectedValue(new Error(DIVERGED))
    vi.mocked(RemoteService.PullMerge).mockResolvedValue()
    const { result, rerender } = renderHook(({ repoPath }) => useRemoteSync(repoPath, vi.fn()), {
      wrapper,
      initialProps: { repoPath: REPO },
    })

    await act(async () => result.current.pull())
    expect(result.current.pullOp.error).toBe(DIVERGED)

    rerender({ repoPath: OTHER_REPO })
    expect(result.current.pullOp.error).toBeNull()

    rerender({ repoPath: REPO })
    expect(result.current.pullOp.error).toBe(DIVERGED)
    await act(async () => result.current.pullMerge())

    expect(RemoteService.PullMerge).toHaveBeenCalledWith(REPO, null)
    expect(result.current.pullOp.error).toBeNull()
  })

  it('keeps a push running in its repository while another is shown, and cancels it there', async () => {
    const { promise, cancel } = pendingPush()
    vi.mocked(RemoteService.Push).mockReturnValue(promise as unknown as ReturnType<typeof RemoteService.Push>)
    const { result, rerender } = renderHook(({ repoPath }) => useRemoteSync(repoPath, vi.fn()), {
      wrapper,
      initialProps: { repoPath: REPO },
    })

    await act(async () => result.current.push())
    rerender({ repoPath: OTHER_REPO })
    expect(result.current.pushOp.running).toBe(false)
    await act(async () => result.current.pushOp.cancel())
    expect(cancel).not.toHaveBeenCalled()

    rerender({ repoPath: REPO })
    expect(result.current.pushOp.running).toBe(true)
    await act(async () => result.current.pushOp.cancel())

    expect(cancel).toHaveBeenCalled()
    expect(result.current.pushOp.running).toBe(false)
  })
})
