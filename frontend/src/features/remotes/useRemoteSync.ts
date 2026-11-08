import { BranchService, RemoteService, type PushResult } from '@current-client-bindings/app'
import { useCancellableOperation } from '../../lib/cancellableOperation'
import type { ActionResult } from '../../lib/actionResult'
import { errorMessage } from '../../lib/errors'
import { protectionFor } from '../../lib/protection'
import { primaryRemoteName } from '../../lib/remotes'
import { useDialogs } from '../../lib/useDialogs'

const NO_UPSTREAM = 'This branch has no upstream configured.'
// Matches core/gitexec's divergedMessage.
export const DIVERGED = 'This branch and its upstream have diverged. Choose Pull (merge) or Pull (rebase).'

export function useRemoteSync(
  repoPath: string | null,
  onSynced: () => void,
  onPushed?: (result: PushResult) => void,
  // After every kind of push, including a force-push, which has no report.
  afterPush?: () => void,
  onResult?: (result: ActionResult) => void,
) {
  // then, when given, replaces onPushed so the caller can carry on (the
  // sidebar opens the create form).
  const pushed = (result: PushResult, then?: (result: PushResult) => void) => {
    onSynced()
    if (!then) onPushed?.(result)
    afterPush?.()
    onResult?.({ text: `Pushed ${result.branch} to ${result.upstream}` })
    then?.(result)
  }
  const pulled = () => {
    onSynced()
    onResult?.({ text: 'Pulled', undoable: true })
  }
  const { confirm } = useDialogs()
  const pullOp = useCancellableOperation(repoPath ?? '')
  const pushOp = useCancellableOperation(repoPath ?? '')

  const pullWith = (pullRepo: typeof RemoteService.Pull) => () => {
    if (!repoPath) return
    pullOp.run((auth) => pullRepo(repoPath, auth), pulled)
  }

  const pushSettingUpstream = async (path: string, then?: (result: PushResult) => void) => {
    try {
      const [status, remoteName] = await Promise.all([BranchService.CurrentBranchStatus(path), primaryRemoteName(path)])
      const protection = await protectionFor(path, `${remoteName}/${status.current}`, 'push')
      const confirmed = await confirm({
        title: 'Set upstream',
        message: [`Push and set upstream to ${remoteName}/${status.current}?`, protection].filter(Boolean).join(' '),
        confirmLabel: 'Push',
      })
      if (!confirmed) return
      await pushOp.run(
        (auth) => RemoteService.PushSetUpstream(path, remoteName, status.current, auth),
        (result) => pushed(result, then),
      )
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
          'What it overwrites, as of the last fetch, stays in Recover.',
          protection,
        ]
          .filter(Boolean)
          .join(' '),
        confirmLabel: 'Force-push',
        destructive: true,
      })
      if (!confirmed) return
      void pushOp.run(
        (auth) => RemoteService.ForcePush(repoPath, auth),
        () => {
          onSynced()
          afterPush?.()
          onResult?.({ text: `Force-pushed to ${status.upstream}` })
        },
      )
    } catch (err: unknown) {
      pushOp.setError(errorMessage(err))
    }
  }

  const pushTags = () => {
    if (!repoPath) return
    primaryRemoteName(repoPath)
      .then((remote) =>
        pushOp.run(
          (auth) => RemoteService.PushTags(repoPath, remote, auth),
          () => {
            onSynced()
            onResult?.({ text: `Pushed tags to ${remote}` })
          },
        ),
      )
      .catch((err: unknown) => pushOp.setError(errorMessage(err)))
  }

  return {
    pull: pullWith(RemoteService.Pull),
    pullMerge: pullWith(RemoteService.PullMerge),
    pullRebase: pullWith(RemoteService.PullRebase),
    push,
    forcePush,
    pushTags,
    pushSetUpstream: (then?: (result: PushResult) => void) =>
      repoPath ? pushSettingUpstream(repoPath, then) : Promise.resolve(),
    pullOp,
    pushOp,
  }
}
