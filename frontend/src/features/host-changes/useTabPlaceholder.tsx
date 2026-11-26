import { useEffect } from 'react'

export interface TabProps {
  repoPath: string
  number: number
  onLoaded: (count: number) => void
}

// What a tab shows in place of its list while it loads, fails or is empty; null once there is a list.
function useTabPlaceholder(
  count: number | undefined,
  error: string | null,
  onLoaded: (count: number) => void,
  empty: string,
) {
  useEffect(() => {
    if (count !== undefined) onLoaded(count)
  }, [count, onLoaded])
  if (error) return <p className="host-changes-error">{error}</p>
  if (count === undefined) return <p className="host-changes-empty">Loading…</p>
  if (count === 0) return <p className="host-changes-empty">{empty}</p>
  return null
}

export default useTabPlaceholder
