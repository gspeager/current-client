import { useEffect, useState, type RefObject } from 'react'
import type { FileDiff } from '@current-client-bindings/app'
import { matchesShortcut } from '../../lib/shortcuts'
import { useWindowKeydown } from '../../lib/useWindowKeydown'
import type { DiffPaneHandle, HunkActions } from './DiffPane'

interface HunkNavigationOptions {
  diff: FileDiff
  // A conflicted file steps through its conflict blocks instead of hunks.
  blockStarts: number[]
  pane: RefObject<HTMLElement | null>
  handle: RefObject<DiffPaneHandle | null>
  hunkActions: HunkActions
}

// F7 and Alt+arrows move between hunks, or conflict blocks; S, U and D act on
// the current hunk while focus is in the diff.
export function useHunkNavigation({ diff, blockStarts, pane, handle, hunkActions }: HunkNavigationOptions) {
  const [currentHunk, setCurrentHunk] = useState(0)
  const [currentBlock, setCurrentBlock] = useState(0)

  useEffect(() => {
    setCurrentHunk(0)
    setCurrentBlock(0)
  }, [diff])

  const stepBlock = (step: number) => {
    const index = Math.min(blockStarts.length - 1, Math.max(0, currentBlock + step))
    setCurrentBlock(index)
    handle.current?.revealBlock(index)
  }

  useWindowKeydown((e) => {
    if (!e.metaKey && !e.ctrlKey && !e.altKey && pane.current?.contains(document.activeElement)) {
      const hunk = diff.hunks[currentHunk]
      const key = e.key.toLowerCase()
      if (hunk && key === 's') hunkActions.onStageHunk?.(hunk.raw)
      if (hunk && key === 'u') hunkActions.onUnstageHunk?.(hunk.raw)
      if (hunk && key === 'd') hunkActions.onDiscardHunk?.(hunk.raw)
    }
    const forward = matchesShortcut(e, 'F7') || matchesShortcut(e, 'alt+ArrowDown')
    const back = matchesShortcut(e, 'shift+F7') || matchesShortcut(e, 'alt+ArrowUp')
    if (!forward && !back) return
    if (diff.conflicted) {
      if (blockStarts.length === 0) return
      e.preventDefault()
      stepBlock(forward ? 1 : -1)
      return
    }
    const total = diff.hunks.length
    if (total === 0) return
    e.preventDefault()
    setCurrentHunk((i) => {
      const next = forward ? Math.min(total - 1, i + 1) : Math.max(0, i - 1)
      handle.current?.revealHunk(next)
      return next
    })
  })

  return { currentHunk, currentBlock, stepBlock }
}
