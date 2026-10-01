import { useEffect } from 'react'
import { Events } from '@wailsio/runtime'
import { WatchService } from '@current-client-bindings/app'

const FILES_CHANGED_EVENT = 'repo:files-changed'

// refChanged is true only when HEAD or the reflog moved.
export function useFileWatcher(repoPath: string | null, onChanged: (refChanged: boolean) => void) {
  useEffect(() => {
    if (!repoPath) return
    WatchService.Start(repoPath).catch(() => undefined)
    const unsubscribe = Events.On(FILES_CHANGED_EVENT, ({ data }) => {
      if (data.repoPath === repoPath) onChanged(data.refChanged)
    })
    return () => {
      unsubscribe()
      WatchService.Stop().catch(() => undefined)
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [repoPath])
}
