import { useCallback, useLayoutEffect, useRef, useState, type SetStateAction } from 'react'

// State that belongs to the open repository: another repository starts from
// initial, so does coming back, and a late setter call for the one before is
// dropped.
export function usePerRepository<T>(repoPath: string | null, initial: T) {
  const [state, setState] = useState({ repoPath, value: initial })
  if (state.repoPath !== repoPath) setState({ repoPath, value: initial })
  const openPath = useRef(repoPath)
  useLayoutEffect(() => {
    openPath.current = repoPath
  })
  const set = useCallback(
    (next: SetStateAction<T>) => {
      if (repoPath !== openPath.current) return
      setState((prev) => {
        const current = prev.repoPath === repoPath ? prev.value : initial
        return { repoPath, value: next instanceof Function ? next(current) : next }
      })
    },
    [repoPath, initial],
  )
  return [state.repoPath === repoPath ? state.value : initial, set] as const
}
