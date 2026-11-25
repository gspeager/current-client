import { useEffect, useState, useSyncExternalStore } from 'react'
import { WorktreeService, type BoardInfo } from '@current-client-bindings/app'
import { errorMessage } from '../../lib/errors'

const SOONEST = 2_000
const LATEST = 30_000

// Longer where reading takes longer, so a large repository isn't read all the time.
export const nextRead = (took: number) => Math.min(LATEST, Math.max(SOONEST, took * 10))

const onVisibility = (changed: () => void) => {
  document.addEventListener('visibilitychange', changed)
  return () => document.removeEventListener('visibilitychange', changed)
}

// Reads again and again while it's open and in view. Watching every worktree's
// files takes a file descriptor for each on macOS, and two large ones use them up.
export function useBoard(repoPath: string) {
  const visible = useSyncExternalStore(onVisibility, () => document.visibilityState === 'visible')
  const [board, setBoard] = useState<BoardInfo | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [asked, setAsked] = useState(0)
  useEffect(() => {
    if (!visible) return
    let current = true
    let timer: ReturnType<typeof setTimeout>
    const read = async () => {
      const started = performance.now()
      try {
        const next = await WorktreeService.Board(repoPath)
        if (!current) return
        setBoard(next)
        setError(null)
      } catch (err: unknown) {
        if (!current) return
        setError(errorMessage(err))
      }
      timer = setTimeout(() => void read(), nextRead(performance.now() - started))
    }
    void read()
    return () => {
      current = false
      clearTimeout(timer)
    }
  }, [repoPath, visible, asked])
  return { board, error, refresh: () => setAsked((n) => n + 1) }
}
