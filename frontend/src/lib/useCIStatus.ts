import { useEffect, useRef, useState } from 'react'
import { HostService, type CIMark } from '@current-client-bindings/app'

export type CIMarks = Record<string, CIMark | undefined>

const DEBOUNCE_MS = 250

// The pipeline result for each SHA on screen. Scrolling adds SHAs; each is
// asked for once (the host service remembers "none" too) in one batch per
// pause, so a fast scroll makes few requests. Empty while the host is off
// or in a build without host integration.
export function useCIStatus(repoPath: string, shas: string[]): CIMarks {
  const [marks, setMarks] = useState<CIMarks>({})
  const known = useRef(new Set<string>())
  const pending = useRef(new Set<string>())
  const key = shas.join('\n')

  useEffect(() => {
    known.current.clear()
    pending.current.clear()
    setMarks({})
  }, [repoPath])

  useEffect(() => {
    const waiting = pending.current
    const fresh = shas.filter((sha) => !known.current.has(sha) && !waiting.has(sha))
    if (fresh.length === 0) return
    fresh.forEach((sha) => waiting.add(sha))
    const timer = window.setTimeout(() => {
      const batch = [...waiting]
      waiting.clear()
      HostService.FetchCIStatus(repoPath, batch)
        .then((answer) => {
          batch.forEach((sha) => known.current.add(sha))
          setMarks((prev) => ({ ...prev, ...answer }))
        })
        .catch(() => undefined)
    }, DEBOUNCE_MS)
    return () => {
      window.clearTimeout(timer)
      fresh.forEach((sha) => waiting.delete(sha))
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [repoPath, key])

  return marks
}
