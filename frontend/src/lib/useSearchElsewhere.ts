import { useEffect, useState } from 'react'
import { useEventCallback } from 'usehooks-ts'
import type { CancellablePromise } from '@wailsio/runtime'
import { useStableValue } from './useStableValue'

export interface Elsewhere<T> {
  path: string
  matches: T
}

// A repository slower than this doesn't hold back the next; its results still come.
export const NEXT_AFTER_MS = 2000

// Searches each repository in turn, adding each one's results as they come. A
// new query, or none, cancels what's running; query names everything the search
// depends on.
export function useSearchElsewhere<T>(paths: string[], query: string, search: (path: string) => CancellablePromise<T>) {
  const [found, setFound] = useState<Elsewhere<T>[]>([])
  const [waiting, setWaiting] = useState(0)
  const searched = useStableValue(paths)
  const searchIn = useEventCallback(search)

  useEffect(() => {
    setFound([])
    if (!query || searched.length === 0) {
      setWaiting(0)
      return
    }
    setWaiting(searched.length)
    let stopped = false
    const running = new Set<{ cancel: () => void }>()
    void (async () => {
      for (const path of searched) {
        if (stopped) return
        const search = searchIn(path)
        running.add(search)
        const done = search
          .then((matches) => {
            if (!stopped) setFound((prev) => [...prev, { path, matches }])
          })
          .catch(() => undefined)
          .finally(() => {
            running.delete(search)
            if (!stopped) setWaiting((n) => n - 1)
          })
        await Promise.race([done, new Promise((resolve) => setTimeout(resolve, NEXT_AFTER_MS))])
      }
    })()
    return () => {
      stopped = true
      running.forEach((search) => search.cancel())
    }
  }, [searched, query, searchIn])

  return { found: found.filter((f) => paths.includes(f.path)), waiting }
}
