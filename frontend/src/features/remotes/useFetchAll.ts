import { RemoteService } from '@current-client-bindings/app'
import { useCancellableOperation } from '../../lib/cancellableOperation'

export function useFetchAll(repoPath: string, onFetched?: () => void, prune = false) {
  const { running, error, run, cancel } = useCancellableOperation()

  const fetchAll = () => {
    run((auth) => (prune ? RemoteService.FetchAllPrune : RemoteService.FetchAll)(repoPath, auth), onFetched)
  }

  return { running, error, cancel, fetchAll }
}
