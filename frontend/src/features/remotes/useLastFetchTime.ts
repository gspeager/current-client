import { RemoteService } from '@current-client-bindings/app'
import { useAsyncData } from '../../lib/useAsyncData'

// FETCH_HEAD's mtime is git's own record of the last fetch, so it survives restarts.
export function useLastFetchTime(repoPath: string | null) {
  const { data, reload } = useAsyncData(() => (repoPath ? RemoteService.LastFetchTime(repoPath) : null), [repoPath])
  return { lastFetchedAt: data ? new Date(data) : null, reloadLastFetchTime: reload }
}
