import { useState } from 'react'
import { useEventListener } from 'usehooks-ts'

export function useWindowNarrowerThan(width: number): boolean {
  const [narrow, setNarrow] = useState(() => window.innerWidth < width)
  useEventListener('resize', () => setNarrow(window.innerWidth < width))
  return narrow
}

// Below these, the sidebar becomes its rail and split diffs read as unified.
export const NARROW_SIDEBAR_WIDTH = 1120
export const NARROW_SPLIT_WIDTH = 1040
