import { useEffect, useLayoutEffect, useMemo, useRef, useState, type KeyboardEvent, type MouseEvent } from 'react'
import { useCurrentClientRoot } from '../../lib/currentClientRoot'
import { canvasPainter, drawMap, readMapColors } from './drawMap'
import { folderOf } from './folderAreas'
import { headerAt, layoutMap, neighbor, tileAt, type Direction, type MapItem, type MapLayout } from './mapLayout'
import './CodeMap.scss'

const DIRECTIONS: Record<string, Direction> = {
  ArrowLeft: 'left',
  ArrowRight: 'right',
  ArrowUp: 'up',
  ArrowDown: 'down',
}

interface CodeMapProps {
  items: MapItem[]
  // The folder that fills the map.
  root: string
  label: string
  // A tile in words, with nothing left to its color.
  describe: (item: MapItem) => string
  onOpen: (item: MapItem) => void
  onMenu?: (e: MouseEvent | KeyboardEvent, item: MapItem) => void
  onRoot: (folder: string) => void
  // The tile pointed at or focused.
  onActive: (item: MapItem | null) => void
}

// The folder one step into the map that holds the file, or null when it's directly
// in the folder that fills it.
export function stepInto(layout: MapLayout, root: string, path: string): string | null {
  const holders = layout.folders.filter((folder) => path.startsWith(`${folder.path}/`) && folder.path !== root)
  return holders.sort((a, b) => a.path.length - b.path.length)[0]?.path ?? null
}

// Tens of thousands of tiles, so they're drawn on a canvas, with one element over
// it for the tile that's pointed at or focused.
function CodeMap({ items, root, label, describe, onOpen, onMenu, onRoot, onActive }: CodeMapProps) {
  const box = useRef<HTMLDivElement>(null)
  const canvas = useRef<HTMLCanvasElement>(null)
  const { element: themed } = useCurrentClientRoot()
  const [size, setSize] = useState({ width: 0, height: 0 })
  const [active, setActive] = useState<string | null>(null)
  const [overHeader, setOverHeader] = useState<string | null>(null)
  const [focused, setFocused] = useState(false)
  const [pointed, setPointed] = useState(false)

  useLayoutEffect(() => {
    const el = box.current!
    // A hidden view has no size, and keeps its picture.
    const measure = () => {
      if (el.clientWidth > 0) setSize({ width: el.clientWidth, height: el.clientHeight })
    }
    measure()
    if (typeof ResizeObserver === 'undefined') return
    const observer = new ResizeObserver(measure)
    observer.observe(el)
    return () => observer.disconnect()
  }, [])

  const layout = useMemo(() => layoutMap(items, root, size.width, size.height), [items, root, size])
  const index = useMemo(() => layout.tiles.findIndex((tile) => tile.item.path === active), [layout, active])
  // The place is kept while focus is elsewhere, such as in what was opened from it.
  const tile = focused || pointed ? layout.tiles[index] : undefined

  useEffect(() => onActive(tile?.item ?? null), [tile?.item]) // eslint-disable-line react-hooks/exhaustive-deps

  useLayoutEffect(() => {
    const draw = () => {
      const el = canvas.current
      const context = el?.getContext('2d')
      if (!el || !context || layout.width === 0) return
      const ratio = window.devicePixelRatio || 1
      el.width = Math.round(layout.width * ratio)
      el.height = Math.round(layout.height * ratio)
      context.scale(ratio, ratio)
      drawMap(canvasPainter(context, getComputedStyle(el).getPropertyValue('--font-mono')), layout, readMapColors(el))
    }
    draw()
    const observer = new MutationObserver(draw)
    if (themed) observer.observe(themed, { attributes: true, attributeFilter: ['data-theme'] })
    // Text drawn before its font has loaded is in another.
    let current = true
    void document.fonts?.ready.then(() => current && draw())
    return () => {
      current = false
      observer.disconnect()
    }
  }, [layout, themed])

  const at = (e: MouseEvent) => {
    const bounds = box.current!.getBoundingClientRect()
    const x = e.clientX - bounds.left
    const y = e.clientY - bounds.top
    const header = headerAt(layout, x, y)
    return { header, tile: header ? undefined : layout.tiles[tileAt(layout, x, y)] }
  }

  const onMouseMove = (e: MouseEvent) => {
    const found = at(e)
    setPointed(true)
    setActive(found.tile?.item.path ?? null)
    setOverHeader(found.header?.path ?? null)
  }

  const onClick = (e: MouseEvent) => {
    const found = at(e)
    if (found.header) onRoot(found.header.path)
    else if (found.tile) onOpen(found.tile.item)
  }

  const onKeyDown = (e: KeyboardEvent) => {
    if (e.altKey || e.ctrlKey || e.metaKey) return
    const go = (next: number) => {
      const item = layout.tiles[next]?.item
      if (item) setActive(item.path)
    }
    const direction = DIRECTIONS[e.key]
    if (direction) go(index < 0 ? 0 : neighbor(layout, index, direction))
    // Every tile in turn, since an arrow may never land on one.
    else if (e.key === 'PageDown') go(index + 1)
    else if (e.key === 'PageUp') go(index - 1)
    else if (e.key === 'Home') go(0)
    else if (e.key === 'End') go(layout.tiles.length - 1)
    else if (e.key === 'Enter' && e.shiftKey && tile) {
      const into = stepInto(layout, root, tile.item.path)
      if (into !== null) onRoot(into)
    } else if (e.key === 'Enter' && tile) onOpen(tile.item)
    else if ((e.key === 'Backspace' || e.key === 'Escape') && root !== '') onRoot(folderOf(root))
    else if (onMenu && tile && (e.key === 'ContextMenu' || (e.key === 'F10' && e.shiftKey))) onMenu(e, tile.item)
    else return
    e.preventDefault()
  }

  const facts = tile && describe(tile.item)
  const header = layout.folders.find((folder) => folder.header && folder.path === overHeader)
  return (
    <div
      ref={box}
      className="code-map"
      role="listbox"
      aria-label={label}
      aria-activedescendant={tile ? 'code-map-tile' : undefined}
      tabIndex={0}
      title={header ? `Open ${header.path}` : facts}
      onMouseMove={onMouseMove}
      onMouseLeave={() => {
        setOverHeader(null)
        setPointed(false)
      }}
      onClick={onClick}
      onContextMenu={(e) => {
        const found = at(e)
        if (found.tile && onMenu) onMenu(e, found.tile.item)
      }}
      onKeyDown={onKeyDown}
      onFocus={() => {
        setFocused(true)
        if (index < 0 && layout.tiles.length > 0) setActive(layout.tiles[0].item.path)
      }}
      onBlur={() => setFocused(false)}
    >
      <canvas ref={canvas} className="code-map-canvas" aria-hidden="true" />
      {tile && (
        <div
          id="code-map-tile"
          className="code-map-tile"
          role="option"
          aria-selected="true"
          aria-label={facts}
          aria-posinset={index + 1}
          aria-setsize={layout.tiles.length}
          style={{ left: tile.x0, top: tile.y0, width: tile.x1 - tile.x0, height: tile.y1 - tile.y0 }}
        />
      )}
    </div>
  )
}

export default CodeMap
