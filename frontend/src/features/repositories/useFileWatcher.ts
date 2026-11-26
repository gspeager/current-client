import { useEffect } from 'react'
import { useEventCallback } from 'usehooks-ts'
import { Events } from '@wailsio/runtime'
import { WatchService } from '@current-client-bindings/app'

const FILES_CHANGED_EVENT = 'repo:files-changed'

// refChanged is true only when HEAD or the reflog moved.
export function useFileWatcher(repoPath: string | null, onChanged: (refChanged: boolean) => void) {
  const changed = useEventCallback(onChanged)
  useEffect(() => {
    if (!repoPath) return
    WatchService.Start(repoPath).catch(() => undefined)
    const unsubscribe = Events.On(FILES_CHANGED_EVENT, ({ data }) => {
      if (data.repoPath === repoPath) changed(data.refChanged)
    })
    return () => {
      unsubscribe()
      WatchService.Stop().catch(() => undefined)
    }
  }, [repoPath, changed])
}
