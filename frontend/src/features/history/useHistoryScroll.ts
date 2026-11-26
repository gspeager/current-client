import { useEffect, useRef, useState } from 'react'
import type { Virtualizer } from '@tanstack/react-virtual'
import type { CommitInfo } from '@current-client-bindings/app'

interface HistoryScrollOptions {
  virtualizer: Virtualizer<HTMLDivElement, Element>
  // The rows shown, after the search and Hide merges.
  rows: { commit: CommitInfo }[]
  // Rows above the commits, such as the working changes row.
  offset: number
  commits: CommitInfo[]
  loading: boolean
  done: boolean
  loadMore: () => void
  loadThrough: (index: number) => void
  initialSelectedSha: string | null
}

// Loading further as the list nears its end, scrolling to the commit History
// opened on, and jumping to a row of the full history; returns the jump.
export function useHistoryScroll({
  virtualizer,
  rows,
  offset,
  commits,
  loading,
  done,
  loadMore,
  loadThrough,
  initialSelectedSha,
}: HistoryScrollOptions) {
  const virtualItems = virtualizer.getVirtualItems()
  useEffect(() => {
    const last = virtualItems[virtualItems.length - 1]
    if (last && last.index >= commits.length - 20) loadMore()
  }, [virtualItems, commits.length, done, loadMore])

  // initialLoadThrough guarantees the commit is loaded, not visible.
  const scrolledToInitial = useRef(false)
  useEffect(() => {
    if (scrolledToInitial.current || !initialSelectedSha) return
    const index = rows.findIndex((r) => r.commit.sha === initialSelectedSha)
    if (index === -1) return
    scrolledToInitial.current = true
    virtualizer.scrollToIndex(index + offset, { align: 'center' })
  }, [rows, offset, initialSelectedSha, virtualizer])

  // A jump waits for its row to load. A row the search or Hide merges has
  // taken out lands on the next one still shown.
  const [jumpRow, setJumpRow] = useState<number | null>(null)
  useEffect(() => {
    if (jumpRow === null) return
    if (jumpRow >= commits.length && !done) {
      loadThrough(jumpRow)
      return
    }
    setJumpRow(null)
    const positions = new Map(commits.map((commit, index) => [commit.sha, index]))
    const shown = rows.findIndex((row) => (positions.get(row.commit.sha) ?? 0) >= jumpRow)
    const target = shown === -1 ? rows.length - 1 : shown
    if (target >= 0) virtualizer.scrollToIndex(target + offset, { align: 'start' })
  }, [jumpRow, commits, rows, offset, done, loading, loadThrough, virtualizer])

  return setJumpRow
}
