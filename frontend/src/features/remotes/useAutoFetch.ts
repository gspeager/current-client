import { useEffect } from 'react'
import { RemoteService } from '@current-client-bindings/app'

// intervalMinutes of 0 means off, the default.
export function useAutoFetch(repoPath: string | null, intervalMinutes: number, onFetched: () => void) {
  useEffect(() => {
    if (!repoPath || intervalMinutes <= 0) return
    const id = window.setInterval(
      () => {
        RemoteService.FetchAll(repoPath, null)
          .then(onFetched)
          .catch(() => undefined) // a background fetch failing silently is fine — the next interval tries again
      },
      intervalMinutes * 60 * 1000,
    )
    return () => window.clearInterval(id)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [repoPath, intervalMinutes])
}
