import { RemoteService } from '@current-client-bindings/app'
import { useCancellableOperation } from '../../lib/cancellableOperation'

export function useFetchAll(repoPath: string, onFetched?: () => void) {
  const { running, error, run, cancel } = useCancellableOperation()

  const fetchAll = () => {
    run(RemoteService.FetchAll(repoPath), onFetched)
  }

  return { running, error, cancel, fetchAll }
}
