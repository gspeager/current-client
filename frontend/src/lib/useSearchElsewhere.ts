import { useEffect, useRef, useState } from 'react'
import type { CancellablePromise } from '@wailsio/runtime'

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
  const key = paths.join('\0')
  const searchRef = useRef(search)
  searchRef.current = search

  useEffect(() => {
    setFound([])
    if (!query || paths.length === 0) {
      setWaiting(0)
      return
    }
    setWaiting(paths.length)
    let stopped = false
    const running = new Set<{ cancel: () => void }>()
    void (async () => {
      for (const path of paths) {
        if (stopped) return
        const search = searchRef.current(path)
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
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [key, query])

  return { found: found.filter((f) => paths.includes(f.path)), waiting }
}
