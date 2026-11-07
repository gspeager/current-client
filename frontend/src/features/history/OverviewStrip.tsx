import { plural } from '../../lib/plural'
import { useLayoutEffect, useMemo, useRef, useState, type MouseEvent } from 'react'
import type { OverviewDayInfo, OverviewMarkerInfo } from '@current-client-bindings/app'
import { useLaneColors } from '../../lib/laneColor'
import './OverviewStrip.scss'

const HEIGHT = 44
const PLOT_HEIGHT = 30
const MIN_POINT_SPACING = 12
// Where a ref's short tick sits. Any other marker is a full-height line.
const MARKER_ROW_TOP: Record<string, number> = { branch: 32, tag: 38, stash: 38 }
const MARKER_ROW_TICK = { width: 3, height: 4 }
const MIN_WINDOW_WIDTH = 4

interface OverviewStripProps {
  days: OverviewDayInfo[]
  markers: OverviewMarkerInfo[]
  // The dates of the rows on screen.
  visible: { newest: number; oldest: number } | null
  onJump: (row: number) => void
}

const DAY_MS = 24 * 60 * 60 * 1000

// Busy days would flatten the rest on a linear scale.
function activityLine(days: OverviewDayInfo[], x: (ms: number) => number, width: number, spacing: number): string {
  const counts = new Array<number>(Math.max(1, Math.ceil(width / spacing))).fill(0)
  for (const day of days) {
    const index = Math.floor(x(Date.parse(day.date) + DAY_MS / 2) / spacing)
    counts[Math.min(counts.length - 1, Math.max(0, index))] += day.commits
  }
  const peak = Math.sqrt(Math.max(...counts, 1))
  const y = (count: number) => (PLOT_HEIGHT - (Math.sqrt(count) / peak) * (PLOT_HEIGHT - 4)).toFixed(1)
  const points = counts.map((count, i) => ({ x: (i + 0.5) * spacing, y: y(count) }))
  const curves = points.slice(1).map((point, i) => {
    const middle = (points[i].x + point.x) / 2
    return `C${middle} ${points[i].y} ${middle} ${point.y} ${point.x} ${point.y}`
  })
  return `M0 ${points[0].y} L${points[0].x} ${points[0].y} ${curves.join(' ')} L${width} ${points[points.length - 1].y}`
}

function OverviewStrip({ days, markers, visible, onJump }: OverviewStripProps) {
  const { branchColor } = useLaneColors()
  const box = useRef<HTMLDivElement>(null)
  const [width, setWidth] = useState(0)
  const [hover, setHover] = useState<{ x: number; text: string } | null>(null)

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
  const line = useMemo(() => {
    if (width <= 0) return ''
    const dayWidth = (DAY_MS / (end - start)) * width
    return activityLine(days, x, width, Math.max(MIN_POINT_SPACING, dayWidth))
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [days, width, start, end])

  const color = (marker: OverviewMarkerInfo): string => {
    switch (marker.kind) {
      case 'head':
        return 'var(--primary)'
      case 'branch':
        return branchColor(marker.name, false)
      case 'tag':
        return 'var(--secondary)'
      default:
        return 'var(--warning)'
    }
  }

  const dayAt = (clientX: number, left: number) => {
    const at = start + ((clientX - left) / (width || 1)) * (end - start)
    let nearest = days[0]
    for (const day of days) {
      const middle = (ms: number) => Math.abs(ms + DAY_MS / 2 - at)
      if (middle(Date.parse(day.date)) < middle(Date.parse(nearest.date))) nearest = day
    }
    return nearest
  }

  const jump = (e: MouseEvent<HTMLDivElement>) => {
    onJump(dayAt(e.clientX, e.currentTarget.getBoundingClientRect().left).firstRow)
  }

  // Rows are in graph order, which dates needn't follow.
  const windowLeft = visible ? x(Math.max(Math.min(visible.oldest, visible.newest), start)) : 0
  const windowRight = visible ? x(Math.min(Math.max(visible.oldest, visible.newest), end)) : 0
  const windowWidth = Math.max(MIN_WINDOW_WIDTH, windowRight - windowLeft)

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
    <div
      ref={box}
      className="overview-strip"
      role="img"
      aria-label="Commit activity over time"
      onClick={jump}
      onMouseLeave={() => setHover(null)}
      onMouseMove={(e) => {
        const left = e.currentTarget.getBoundingClientRect().left
        const day = dayAt(e.clientX, left)
        const date = new Date(Date.parse(day.date)).toLocaleDateString(undefined, {
          month: 'short',
          day: 'numeric',
          year: 'numeric',
        })
        setHover({ x: e.clientX - left, text: `${date}: ${plural(day.commits, 'commit')}. Click to go there` })
      }}
    >
      <svg width={width} height={HEIGHT} aria-hidden="true">
        {visible && (
          <rect
            className="overview-strip-window"
            data-testid="overview-window"
            x={Math.min(windowLeft, width - windowWidth)}
            y={0}
            width={windowWidth}
            height={HEIGHT}
          />
        )}
        <path className="overview-strip-line" d={line} />
        {ticks.map((marker, index) => {
          const rowTop = MARKER_ROW_TOP[marker.kind]
          return (
            <rect
              key={`${marker.kind}-${marker.name}-${index}`}
              data-marker={marker.kind}
              x={x(Date.parse(marker.date))}
              y={rowTop ?? 0}
              width={rowTop === undefined ? 1 : MARKER_ROW_TICK.width}
              height={rowTop === undefined ? HEIGHT : MARKER_ROW_TICK.height}
              fill={color(marker)}
            >
              <title>{marker.name}</title>
            </rect>
          )
        })}
      </svg>
      {hover && (
        <span className="overview-strip-tip" style={{ left: Math.min(hover.x, width - 220) }}>
          {hover.text}
        </span>
      )}
    </div>
  )
}

export default OverviewStrip
