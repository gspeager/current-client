import { RemoteService } from '@current-client-bindings/app'
import { useCancellableOperation } from '../../lib/cancellableOperation'
import { errorMessage } from '../../lib/errors'
import { primaryRemoteName } from '../../lib/remotes'

// The header's Fetch and its menu. `prune` is the setting; fetchPrune prunes
// whatever the setting says.
export function useFetchAll(repoPath: string, onFetched?: () => void, prune = false) {
  const { running, error, run, cancel, setError } = useCancellableOperation()

  const fetchAll = () => {
    run((auth) => (prune ? RemoteService.FetchAllPrune : RemoteService.FetchAll)(repoPath, auth), onFetched)
  }

  // The remote's name is looked up first, so the fetch itself stays cancellable.
  const fetchPrimary = (pruneNow = prune) => {
    primaryRemoteName(repoPath)
      .then((remote) =>
        run((auth) => (pruneNow ? RemoteService.FetchPrune : RemoteService.Fetch)(repoPath, remote, auth), onFetched),
      )
      .catch((err: unknown) => setError(errorMessage(err)))
  }

  return { running, error, cancel, fetchAll, fetchPrimary, fetchPrune: () => fetchPrimary(true) }
}
