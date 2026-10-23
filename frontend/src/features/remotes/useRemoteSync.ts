import { BranchService, RemoteService, type PushResult } from '@current-client-bindings/app'
import { useCancellableOperation } from '../../lib/cancellableOperation'
import { errorMessage } from '../../lib/errors'
import { protectionFor } from '../../lib/protection'
import { primaryRemoteName } from '../../lib/remotes'
import { useDialogs } from '../../lib/useDialogs'

const NO_UPSTREAM = 'This branch has no upstream configured.'
// Matches core/gitexec's divergedMessage.
export const DIVERGED = 'This branch and its upstream have diverged. Choose Pull (merge) or Pull (rebase).'

export function useRemoteSync(repoPath: string | null, onSynced: () => void, onPushed?: (result: PushResult) => void) {
  const pushed = (result: PushResult) => {
    onSynced()
    onPushed?.(result)
  }
  const { confirm } = useDialogs()
  const pullOp = useCancellableOperation(repoPath ?? '')
  const pushOp = useCancellableOperation(repoPath ?? '')

  const pull = () => {
    if (!repoPath) return
    pullOp.run((auth) => RemoteService.Pull(repoPath, auth), onSynced)
  }

  const pullMerge = () => {
    if (!repoPath) return
    pullOp.run((auth) => RemoteService.PullMerge(repoPath, auth), onSynced)
  }

  const pullRebase = () => {
    if (!repoPath) return
    pullOp.run((auth) => RemoteService.PullRebase(repoPath, auth), onSynced)
  }

  const pushSettingUpstream = async (path: string) => {
    try {
      const [status, remoteName] = await Promise.all([BranchService.CurrentBranchStatus(path), primaryRemoteName(path)])
      const protection = await protectionFor(path, `${remoteName}/${status.current}`, 'push')
      const confirmed = await confirm({
        title: 'Set upstream',
        message: [`Push and set upstream to ${remoteName}/${status.current}?`, protection].filter(Boolean).join(' '),
        confirmLabel: 'Push',
      })
      if (!confirmed) return
      void pushOp.run((auth) => RemoteService.PushSetUpstream(path, remoteName, status.current, auth), pushed)
    } catch (err: unknown) {
      pushOp.setError(errorMessage(err))
    }
  }

  // A branch the host protects is confirmed first; the host still decides.
  const push = async () => {
    if (!repoPath) return
    const protection = await protectionFor(repoPath, '', 'push')
    if (protection) {
      const confirmed = await confirm({
        title: 'Push to a protected branch',
        message: protection,
        confirmLabel: 'Push anyway',
      })
      if (!confirmed) return
    }
    void pushOp.run(
      (auth) => RemoteService.Push(repoPath, auth),
      pushed,
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
      const protection = await protectionFor(repoPath, status.upstream, 'force-push')
      const confirmed = await confirm({
        title: 'Force-push',
        message: [
          `Force-push ${status.current} to ${status.upstream}? This rewrites remote history and can discard commits other people have pushed.`,
          protection,
        ]
          .filter(Boolean)
          .join(' '),
        confirmLabel: 'Force-push',
        destructive: true,
      })
      if (!confirmed) return
      void pushOp.run((auth) => RemoteService.ForcePush(repoPath, auth), onSynced)
    } catch (err: unknown) {
      pushOp.setError(errorMessage(err))
    }
  }

  return { pull, pullMerge, pullRebase, push, forcePush, pullOp, pushOp }
}
