import { useRef, useState } from 'react'

// The watcher reports an action's own ref changes about 400 ms after the
// action refreshed everything, so one this soon after needs no second refresh.
const SETTLE_MS = 1000

export function useRepoVersion() {
  const [repoVersion, setRepoVersion] = useState(0)
  const lastBump = useRef(-Infinity)

  const bumpRepoVersion = () => {
    lastBump.current = Date.now()
    setRepoVersion((v) => v + 1)
  }
  const justBumped = () => Date.now() - lastBump.current < SETTLE_MS

  return { repoVersion, bumpRepoVersion, justBumped }
}
