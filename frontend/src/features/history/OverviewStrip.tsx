import { useLayoutEffect, useMemo, useRef, useState, type MouseEvent } from 'react'
import type { OverviewDayInfo, OverviewMarkerInfo } from '@current-client-bindings/app'
import { useLaneColors } from '../../lib/laneColor'
import './OverviewStrip.scss'

const HEIGHT = 40

interface OverviewStripProps {
  days: OverviewDayInfo[]
  markers: OverviewMarkerInfo[]
  // The dates of the rows on screen.
  visible: { newest: number; oldest: number } | null
  onJump: (row: number) => void
}

const DAY_MS = 24 * 60 * 60 * 1000

// Busy days would flatten the rest on a linear scale.
function activityPath(days: OverviewDayInfo[], x: (ms: number) => number, width: number): string {
  const columns = new Array<number>(Math.max(1, Math.ceil(width))).fill(0)
  for (const day of days) {
    const column = Math.min(columns.length - 1, Math.max(0, Math.floor(x(Date.parse(day.date)))))
    columns[column] += day.commits
  }
  const peak = Math.sqrt(Math.max(...columns, 1))
  const y = (count: number) => HEIGHT - (Math.sqrt(count) / peak) * (HEIGHT - 2)
  return `M0 ${HEIGHT} ${columns.map((count, i) => `L${i} ${y(count).toFixed(1)} L${i + 1} ${y(count).toFixed(1)}`).join(' ')} L${columns.length} ${HEIGHT} Z`
}

function OverviewStrip({ days, markers, visible, onJump }: OverviewStripProps) {
  const { branchColor } = useLaneColors()
  const box = useRef<HTMLDivElement>(null)
  const [width, setWidth] = useState(0)

  useLayoutEffect(() => {
    const element = box.current
    if (!element) return
    const measure = () => setWidth(element.offsetWidth)
    measure()
    if (typeof ResizeObserver === 'undefined') return
    const observer = new ResizeObserver(measure)
    observer.observe(element)
    return () => observer.disconnect()
  }, [])

  const { start, end } = useMemo(
    () => ({ start: Date.parse(days[0].date), end: Date.parse(days[days.length - 1].date) + DAY_MS }),
    [days],
  )
  const x = (ms: number) => ((ms - start) / (end - start)) * width
  const path = useMemo(() => (width > 0 ? activityPath(days, x, width) : ''), [days, width, start, end]) // eslint-disable-line react-hooks/exhaustive-deps

  const color = (marker: OverviewMarkerInfo): string => {
    switch (marker.kind) {
      case 'head':
        return 'var(--primary)'
      case 'branch':
        return branchColor(marker.name, false)
      case 'match':
        return 'var(--warning)'
      default:
        return 'var(--on-surface-variant)'
    }
  }

  const jump = (e: MouseEvent<HTMLDivElement>) => {
    const left = e.currentTarget.getBoundingClientRect().left
    const clicked = start + ((e.clientX - left) / (width || 1)) * (end - start)
    let nearest = days[0]
    for (const day of days) {
      const middle = (ms: number) => Math.abs(ms + DAY_MS / 2 - clicked)
      if (middle(Date.parse(day.date)) < middle(Date.parse(nearest.date))) nearest = day
    }
    onJump(nearest.firstRow)
  }

  // Rows are in graph order, which dates needn't follow.
  const windowLeft = visible ? x(Math.max(Math.min(visible.oldest, visible.newest), start)) : 0
  const windowRight = visible ? x(Math.min(Math.max(visible.oldest, visible.newest), end)) : 0

  // One tick per pixel and color, however many refs share it. Last wins, so HEAD,
  // listed first, is drawn again on top.
  const ticks = useMemo(() => {
    const byPixel = new Map<string, OverviewMarkerInfo>()
    for (const marker of markers) {
      const ms = Date.parse(marker.date)
      if (ms >= start && ms <= end) byPixel.set(`${Math.round(x(ms))}-${color(marker)}`, marker)
    }
    return [...byPixel.values()].sort((a, b) => Number(a.kind === 'head') - Number(b.kind === 'head'))
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [markers, width, start, end])

  return (
    <div ref={box} className="overview-strip" role="img" aria-label="Commit activity over time" onClick={jump}>
      <svg width={width} height={HEIGHT} aria-hidden="true">
        <path className="overview-strip-area" d={path} />
        {ticks.map((marker, index) => (
          <rect
            key={`${marker.kind}-${marker.name}-${index}`}
            data-marker={marker.kind}
            x={x(Date.parse(marker.date))}
            y={0}
            width={1}
            height={HEIGHT}
            fill={color(marker)}
          >
            <title>{marker.name}</title>
          </rect>
        ))}
        {visible && (
          <rect
            className="overview-strip-window"
            data-testid="overview-window"
            x={windowLeft}
            y={0}
            width={Math.max(2, windowRight - windowLeft)}
            height={HEIGHT}
          />
        )}
      </svg>
    </div>
  )
}

export default OverviewStrip
