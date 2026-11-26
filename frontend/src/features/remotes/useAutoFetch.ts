import { useEffect } from 'react'
import { useEventCallback } from 'usehooks-ts'
import { RemoteService } from '@current-client-bindings/app'

// intervalMinutes of 0 means off, the default.
export function useAutoFetch(repoPath: string | null, intervalMinutes: number, onFetched: () => void) {
  const fetched = useEventCallback(onFetched)
  useEffect(() => {
    if (!repoPath || intervalMinutes <= 0) return
    const id = window.setInterval(
      () => {
        RemoteService.FetchAll(repoPath, null)
          .then(fetched)
          .catch(() => undefined) // a background fetch failing silently is fine — the next interval tries again
      },
      intervalMinutes * 60 * 1000,
    )
    return () => window.clearInterval(id)
  }, [repoPath, intervalMinutes, fetched])
}
