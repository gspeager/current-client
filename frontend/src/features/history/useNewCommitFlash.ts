import { useEffect, useRef, useState } from 'react'
import type { CommitInfo } from '@current-client-bindings/app'

const NEW_COMMIT_FLASH_MS = 400

// The commits above the last top History showed, flashed once and then
// cleared. Only the first load counts; no overlap with previousTopSha (a
// rebase or first open) flashes nothing.
export function useNewCommitFlash(
  commits: CommitInfo[],
  previousTopSha: string | null | undefined,
  onTopShaChange?: (sha: string) => void,
) {
  const [newShas, setNewShas] = useState<Set<string>>(new Set())
  const computed = useRef(false)

  useEffect(() => {
    if (commits.length === 0 || computed.current) return
    computed.current = true
    if (previousTopSha) {
      const priorIndex = commits.findIndex((c) => c.sha === previousTopSha)
      if (priorIndex > 0) setNewShas(new Set(commits.slice(0, priorIndex).map((c) => c.sha)))
    }
    onTopShaChange?.(commits[0].sha)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [commits])

  useEffect(() => {
    if (newShas.size === 0) return
    const timeout = setTimeout(() => setNewShas(new Set()), NEW_COMMIT_FLASH_MS)
    return () => clearTimeout(timeout)
  }, [newShas])

  return newShas
}
