import { useEffect, useMemo, useState, type MouseEvent, type RefObject } from 'react'
import type { DisplayRow } from './diffRows'
import './DiffMinimap.scss'

interface DiffMinimapProps {
  rows: DisplayRow[]
  paneRef: RefObject<HTMLDivElement>
  // Takes its place before the pane is measured, so appearing doesn't narrow it.
  always?: boolean
}

type Mark = { top: number; height: number; kind: 'added' | 'removed' }

// About what the strip can show; a long diff's rows share them.
const SLOTS = 1000

function marksFor(rows: DisplayRow[]): Mark[] {
  const marks: Mark[] = []
  const slots = Math.min(Math.max(rows.length, 1), SLOTS)
  const unit = 100 / slots
  let filled = -1
  rows.forEach((row, i) => {
    if (!('pair' in row)) return
    const kinds = [row.pair.left?.kind, row.pair.right?.kind]
    const kind = kinds.includes('added') ? 'added' : kinds.includes('removed') ? 'removed' : null
    const slot = Math.floor((i / rows.length) * slots)
    if (!kind || slot === filled) return
    const last = marks[marks.length - 1]
    if (last?.kind === kind && slot === filled + 1) last.height += unit
    else marks.push({ top: slot * unit, height: unit, kind })
    filled = slot
  })
  return marks
}

// Where a file's changes fall, and which part is on screen. A click scrolls there.
function DiffMinimap({ rows, paneRef, always = false }: DiffMinimapProps) {
  const marks = useMemo(() => marksFor(rows), [rows])
  const [view, setView] = useState<{ top: number; height: number } | null>(null)

  useEffect(() => {
    const pane = paneRef.current
    if (!pane) return
    const measure = () =>
      setView(
        pane.scrollHeight > pane.clientHeight
          ? {
              top: (pane.scrollTop / pane.scrollHeight) * 100,
              height: (pane.clientHeight / pane.scrollHeight) * 100,
            }
          : null,
      )
    measure()
    pane.addEventListener('scroll', measure, { passive: true })
    if (typeof ResizeObserver === 'undefined') return () => pane.removeEventListener('scroll', measure)
    const observer = new ResizeObserver(measure)
    observer.observe(pane)
    return () => {
      pane.removeEventListener('scroll', measure)
      observer.disconnect()
    }
  }, [paneRef, rows])

  if (!view && !always) return null

  const jump = (e: MouseEvent<HTMLDivElement>) => {
    const pane = paneRef.current
    if (!pane) return
    const box = e.currentTarget.getBoundingClientRect()
    pane.scrollTop = ((e.clientY - box.top) / box.height) * pane.scrollHeight - pane.clientHeight / 2
  }

  return (
    <div className="diff-minimap" onClick={jump} aria-hidden="true">
      {marks.map((mark) => (
        <span
          key={mark.top}
          className={`diff-minimap-mark diff-minimap-${mark.kind}`}
          style={{ top: `${mark.top}%`, height: `max(2px, ${mark.height}%)` }}
        />
      ))}
      {view && <span className="diff-minimap-view" style={{ top: `${view.top}%`, height: `${view.height}%` }} />}
    </div>
  )
}

export default DiffMinimap
