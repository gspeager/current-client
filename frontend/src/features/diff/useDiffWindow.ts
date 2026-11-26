import { useCallback, useLayoutEffect, useRef, useState, type RefObject } from 'react'
import { defaultRangeExtractor, measureElement, observeElementRect, useVirtualizer } from '@tanstack/react-virtual'
import { useEventCallback } from 'usehooks-ts'

const OVERSCAN = 30

export interface DiffWindow {
  // The rows to draw, in order. Where two aren't neighbors, the gap's height is between them.
  indexes: number[]
  startOf: (index: number) => number
  endOf: (index: number) => number
  total: number
  // For the rows whose height isn't known until drawn.
  measureRow: (index: number) => ((el: HTMLElement | null) => void) | undefined
  reveal: (index: number, block: 'center' | 'nearest') => void
}

interface DiffWindowOptions {
  // False draws every row, which a short diff does.
  windowed: boolean
  count: number
  // Heights are remembered by key, so it names the layout too. A new function
  // works every position out again.
  keyOf: (index: number) => string
  layout: string
  // An estimate, until the row is drawn and measured. Width is the pane's, or 0 before it's known.
  heightOf: (index: number, width: number) => number
  // Whether a row's height has to be measured once drawn.
  measured: (index: number) => boolean
  // Rows that hold something a reader could lose, such as a reply being typed.
  keep: number[]
  scroller: () => HTMLElement | null
  paneRef: RefObject<HTMLElement>
}

const rowIndex = (node: Node | null): number | null => {
  const row = (node instanceof Element ? node : node?.parentElement)?.closest<HTMLElement>('[data-index]')
  return row ? Number(row.dataset.index) : null
}

// The rows of a selection that reaches into the pane, or null.
export function selectedRows(pane: HTMLElement | null): [number, number] | null {
  const selection = document.getSelection()
  if (!pane || !selection || selection.isCollapsed) return null
  if (!pane.contains(selection.anchorNode) || !pane.contains(selection.focusNode)) return null
  const ends = [rowIndex(selection.anchorNode), rowIndex(selection.focusNode)]
  if (ends[0] === null || ends[1] === null) return null
  return [Math.min(ends[0], ends[1]), Math.max(ends[0], ends[1])]
}

// Draws the rows on screen and leaves the rest as blank height. The row with
// focus and both ends of a text selection stay drawn, so scrolling loses neither.
export function useDiffWindow({
  windowed,
  count,
  keyOf,
  layout,
  heightOf,
  measured,
  keep,
  scroller,
  paneRef,
}: DiffWindowOptions): DiffWindow {
  const [width, setWidth] = useState(0)
  const getItemKey = useCallback((index: number) => `${layout}:${keyOf(index)}`, [layout, keyOf])
  const virtualizer = useVirtualizer<HTMLElement, HTMLElement>({
    enabled: windowed,
    count,
    getScrollElement: scroller,
    getItemKey,
    estimateSize: (index) => heightOf(index, width),
    overscan: OVERSCAN,
    initialRect: { width: 0, height: 1000 },
    // A hidden view has no size: it keeps its rows and their heights.
    observeElementRect: (instance, onRect) =>
      observeElementRect(instance, (rect) => {
        if (rect.height > 0) onRect(rect)
      }),
    measureElement: (row, entry, instance) =>
      measureElement(row, entry, instance) || instance.measurementsCache[instance.indexFromElement(row)].size,
    rangeExtractor: (range) => {
      const pane = paneRef.current
      const focused = pane?.contains(document.activeElement) ? rowIndex(document.activeElement) : null
      const drawn = new Set([...defaultRangeExtractor(range), ...keep, ...(selectedRows(pane) ?? [])])
      if (focused !== null) drawn.add(focused)
      return [...drawn].filter((index) => index < range.count).sort((a, b) => a - b)
    },
  })

  // Wrapped rows take other heights at another width, so every height is worked
  // out again, with the same row kept at the top.
  const topRow = useRef<number | null>(null)
  const scrollElement = useEventCallback(scroller)
  useLayoutEffect(() => {
    const el = scrollElement()
    if (!windowed || !el) return
    setWidth(el.clientWidth)
    if (typeof ResizeObserver === 'undefined') return
    const observer = new ResizeObserver(() => {
      if (el.clientWidth === 0) return
      const paneTop = el.getBoundingClientRect().top
      const rows = [...el.querySelectorAll<HTMLElement>('[data-index]')]
      topRow.current = rowIndex(rows.find((row) => row.getBoundingClientRect().bottom > paneTop) ?? null)
      setWidth(el.clientWidth)
    })
    observer.observe(el)
    return () => observer.disconnect()
  }, [windowed, layout, scrollElement])

  // A request from before the width is known would scroll by the wrong heights.
  const waiting = useRef<{ index: number; block: 'center' | 'nearest' } | null>(null)
  const measuredWidth = useRef(0)
  const reveal = (index: number, block: 'center' | 'nearest') => {
    if (measuredWidth.current > 0) virtualizer.scrollToIndex(index, { align: block === 'center' ? 'center' : 'auto' })
    else waiting.current = { index, block }
  }

  useLayoutEffect(() => {
    const el = scroller()
    if (!windowed || !el || width === 0 || width === measuredWidth.current) return
    const resized = measuredWidth.current > 0
    measuredWidth.current = width
    virtualizer.measure()
    virtualizer.getTotalSize()
    for (const row of el.querySelectorAll<HTMLElement>('[data-index]')) {
      const index = Number(row.dataset.index)
      if (measured(index)) virtualizer.resizeItem(index, row.offsetHeight)
    }
    // Works the positions out now, for what scrolls next.
    virtualizer.getTotalSize()
    if (waiting.current) reveal(waiting.current.index, waiting.current.block)
    else if (resized && topRow.current) virtualizer.scrollToIndex(topRow.current, { align: 'start' })
    waiting.current = null
  })

  if (!windowed) {
    return {
      indexes: Array.from({ length: count }, (_, index) => index),
      startOf: () => 0,
      endOf: () => 0,
      total: 0,
      measureRow: () => undefined,
      reveal: (index, block) => paneRef.current?.querySelector(`[data-index="${index}"]`)?.scrollIntoView({ block }),
    }
  }
  const items = virtualizer.getVirtualItems()
  const byIndex = new Map(items.map((item) => [item.index, item]))
  return {
    indexes: items.map((item) => item.index),
    startOf: (index) => byIndex.get(index)?.start ?? 0,
    endOf: (index) => byIndex.get(index)?.end ?? 0,
    total: virtualizer.getTotalSize(),
    measureRow: (index) => (measured(index) ? virtualizer.measureElement : undefined),
    reveal,
  }
}
