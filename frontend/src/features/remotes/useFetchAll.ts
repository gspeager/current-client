import { RemoteService } from '@current-client-bindings/app'
import { useCancellableOperation } from '../../lib/cancellableOperation'

export function useFetchAll(repoPath: string, onFetched?: () => void) {
  const { running, error, run, cancel } = useCancellableOperation()

  const fetchAll = () => {
    run((auth) => RemoteService.FetchAll(repoPath, auth), onFetched)
  }

  return { running, error, cancel, fetchAll }
}
