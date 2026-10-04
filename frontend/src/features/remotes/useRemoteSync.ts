import { BranchService, RemoteService } from '@current-client-bindings/app'
import { useCancellableOperation } from '../../lib/cancellableOperation'
import { errorMessage } from '../../lib/errors'
import { useDialogs } from '../../lib/useDialogs'

const NO_UPSTREAM = 'This branch has no upstream configured.'

export function useRemoteSync(repoPath: string | null, onSynced: () => void) {
  const { confirm } = useDialogs()
  const pullOp = useCancellableOperation()
  const pushOp = useCancellableOperation()

  const pull = () => {
    if (!repoPath) return
    pullOp.run((auth) => RemoteService.Pull(repoPath, auth), onSynced)
  }

  const pullRebase = () => {
    if (!repoPath) return
    pullOp.run((auth) => RemoteService.PullRebase(repoPath, auth), onSynced)
  }

  const pushSettingUpstream = async (path: string) => {
    try {
      const [status, remotes] = await Promise.all([BranchService.CurrentBranchStatus(path), RemoteService.List(path)])
      const remoteName = remotes.find((r) => r.name === 'origin')?.name ?? remotes[0]?.name
      if (!remoteName) {
        pushOp.setError('No remote configured to push to.')
        return
      }
      const confirmed = await confirm({
        title: 'Set upstream',
        message: `Push and set upstream to ${remoteName}/${status.current}?`,
        confirmLabel: 'Push',
      })
      if (!confirmed) return
      void pushOp.run((auth) => RemoteService.PushSetUpstream(path, remoteName, status.current, auth), onSynced)
    } catch (err: unknown) {
      pushOp.setError(errorMessage(err))
    }
  }

  const push = () => {
    if (!repoPath) return
    void pushOp.run(
      (auth) => RemoteService.Push(repoPath, auth),
      onSynced,
      (message) => {
        if (message !== NO_UPSTREAM) return false
        void pushSettingUpstream(repoPath)
        return true
      },
    )
  }

  const forcePush = async () => {
    if (!repoPath) return
    pushOp.setError(null)
    try {
      const status = await BranchService.CurrentBranchStatus(repoPath)
      if (!status.upstream) {
        pushOp.setError('No upstream configured to force-push to.')
        return
      }
      const confirmed = await confirm({
        title: 'Force-push',
        message: `Force-push ${status.current} to ${status.upstream}? This rewrites remote history and can discard commits other people have pushed.`,
        confirmLabel: 'Force-push',
        destructive: true,
      })
      if (!confirmed) return
      void pushOp.run((auth) => RemoteService.ForcePush(repoPath, auth), onSynced)
    } catch (err: unknown) {
      pushOp.setError(errorMessage(err))
    }
  }

  return { pull, pullRebase, push, forcePush, pullOp, pushOp }
}
