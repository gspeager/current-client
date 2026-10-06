import { RemoteService } from '@current-client-bindings/app'
import { useCancellableOperation } from '../../lib/cancellableOperation'
import { useDialogs } from '../../lib/useDialogs'

// remoteBranch is a remote-tracking branch as listed, such as "origin/feature".
export function useDeleteRemoteBranch(repoPath: string, onDeleted?: () => void) {
  const { confirm } = useDialogs()
  const { running, error, run } = useCancellableOperation()

  const deleteRemoteBranch = async (remoteBranch: string) => {
    const confirmed = await confirm({
      title: 'Delete on remote',
      message: `Delete ${remoteBranch} from its remote? It's removed for everyone who uses that remote. Local branches are kept.`,
      confirmLabel: 'Delete',
      destructive: true,
    })
    if (!confirmed) return
    await run((auth) => RemoteService.DeleteRemoteBranch(repoPath, remoteBranch, auth), onDeleted)
  }

  return { running, error, deleteRemoteBranch }
}
