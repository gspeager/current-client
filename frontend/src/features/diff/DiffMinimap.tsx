import { useEffect, useMemo, useState, type MouseEvent, type RefObject } from 'react'
import type { DisplayRow } from './diffRows'
import './DiffMinimap.scss'

interface DiffMinimapProps {
  rows: DisplayRow[]
  paneRef: RefObject<HTMLDivElement>
}

type Mark = { top: number; height: number; kind: 'added' | 'removed' }

function marksFor(rows: DisplayRow[]): Mark[] {
  const marks: Mark[] = []
  const unit = 100 / Math.max(rows.length, 1)
  rows.forEach((row, i) => {
    if (!('pair' in row)) return
    const kinds = [row.pair.left?.kind, row.pair.right?.kind]
    const kind = kinds.includes('added') ? 'added' : kinds.includes('removed') ? 'removed' : null
    if (!kind) return
    const last = marks[marks.length - 1]
    if (last?.kind === kind && Math.abs(last.top + last.height - i * unit) < 0.001) last.height += unit
    else marks.push({ top: i * unit, height: unit, kind })
  })
  return marks
}

// Where a file's changes fall, and which part is on screen. A click scrolls there.
function DiffMinimap({ rows, paneRef }: DiffMinimapProps) {
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

  if (!view) return null

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
      <span className="diff-minimap-view" style={{ top: `${view.top}%`, height: `${view.height}%` }} />
    </div>
  )
}

export default DiffMinimap
