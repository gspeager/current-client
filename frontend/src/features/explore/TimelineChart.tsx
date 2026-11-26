import clsx from 'clsx'
import {
  useEffect,
  useLayoutEffect,
  useMemo,
  useRef,
  useState,
  type KeyboardEvent,
  type MouseEvent,
  type ReactNode,
} from 'react'
import { useEventCallback } from 'usehooks-ts'
import { HoverCardPanel } from '../../components/chrome/HoverCard'
import PathText from '../../components/git/PathText'
import { useCurrentClientRoot } from '../../lib/currentClientRoot'
import { canvasPainter } from '../../lib/picture'
import { drawTimeline, readTimelineColors } from './drawTimeline'
import type { Bubble, Entry, Lane } from './timelineLanes'
import { TICK_ROW, layoutTimeline, type TimelineLayout } from './timelineLayout'
import './TimelineChart.scss'

const GAP = 6
// A small bubble is pointed at from this far away.
const REACH = 5

interface TimelineChartProps {
  lanes: Lane[]
  entries: Entry[]
  tags: { name: string; sha: string }[]
  range: { since: number | null; now: number | null }
  folder: string
  describe: (entry: Entry) => string
  card: (entry: Entry) => ReactNode
  onOpen: (entry: Entry, bubble: Bubble | null) => void
  onFolder: (folder: string) => void
  onActive: (entry: Entry | null) => void
  onLayout: (layout: TimelineLayout) => void
}

interface Active {
  key: string
  // null for the commit's tick.
  lane: number | null
}

const keyOf = (entry: Entry) => entry.sha ?? 'now'

// Thousands of bubbles, so they're drawn on a canvas, with one element over it for the
// bubble that's pointed at or focused.
function TimelineChart({
  lanes,
  entries,
  tags,
  range,
  folder,
  describe,
  card,
  onOpen,
  onFolder,
  onActive,
  onLayout,
}: TimelineChartProps) {
  const box = useRef<HTMLDivElement>(null)
  const canvas = useRef<HTMLCanvasElement>(null)
  const { element: themed } = useCurrentClientRoot()
  const [width, setWidth] = useState(0)
  const [active, setActive] = useState<Active | null>(null)
  const [focused, setFocused] = useState(false)
  const [pointed, setPointed] = useState(false)

  useLayoutEffect(() => {
    const el = box.current!
    const measure = () => {
      if (el.clientWidth > 0) setWidth(el.clientWidth)
    }
    measure()
    if (typeof ResizeObserver === 'undefined') return
    const observer = new ResizeObserver(measure)
    observer.observe(el)
    return () => observer.disconnect()
  }, [])

  const layout = useMemo(() => layoutTimeline(lanes, entries, tags, range, width), [lanes, entries, tags, range, width])
  const index = active ? entries.findIndex((entry) => keyOf(entry) === active.key) : -1
  const shown = focused || pointed ? entries[index] : undefined
  const bubble =
    shown && active?.lane !== null
      ? layout.bubbles.find((b) => b.entry === index && b.lane === active?.lane)
      : undefined

  const reportLayout = useEventCallback(onLayout)
  const reportActive = useEventCallback(onActive)
  useEffect(() => {
    reportLayout(layout)
  }, [layout, reportLayout])
  useEffect(() => {
    reportActive(shown ?? null)
  }, [shown, reportActive])

  useLayoutEffect(() => {
    const draw = () => {
      const el = canvas.current
      const context = el?.getContext('2d')
      if (!el || !context || layout.width === 0) return
      const ratio = window.devicePixelRatio || 1
      el.width = Math.round(layout.width * ratio)
      el.height = Math.round(layout.height * ratio)
      context.scale(ratio, ratio)
      drawTimeline(canvasPainter(context), layout, readTimelineColors(el), false)
    }
    draw()
    const observer = new MutationObserver(draw)
    if (themed) observer.observe(themed, { attributes: true, attributeFilter: ['data-theme'] })
    let current = true
    void document.fonts?.ready.then(() => current && draw())
    return () => {
      current = false
      observer.disconnect()
    }
  }, [layout, themed])

  const at = (e: MouseEvent): Active | null => {
    const bounds = e.currentTarget.getBoundingClientRect()
    const x = e.clientX - bounds.left
    const y = e.clientY - bounds.top
    if (y >= layout.top && y < layout.top + TICK_ROW) {
      let nearest = -1
      layout.xs.forEach((at, i) => {
        if (
          entries[i].sha !== null &&
          Math.abs(at - x) <= REACH &&
          (nearest < 0 || Math.abs(at - x) < Math.abs(layout.xs[nearest] - x))
        )
          nearest = i
      })
      return nearest < 0 ? null : { key: keyOf(entries[nearest]), lane: null }
    }
    for (let i = layout.bubbles.length - 1; i >= 0; i--) {
      const b = layout.bubbles[i]
      if (Math.hypot(b.x - x, b.y - y) <= Math.max(b.radius, REACH))
        return { key: keyOf(entries[b.entry]), lane: b.lane }
    }
    return null
  }

  const open = (found: Active | null) => {
    const entry = found && entries.find((e) => keyOf(e) === found.key)
    if (entry) onOpen(entry, entry.bubbles.find((b) => b.lane === found.lane) ?? null)
  }

  const largest = (entry: Entry) => entry.bubbles.reduce((a, b) => (b.lines > a.lines ? b : a)).lane
  const goTo = (i: number, lane: number | null) => {
    const entry = entries[Math.max(0, Math.min(entries.length - 1, i))]
    if (!entry) return
    const kept = lane !== null && entry.bubbles.some((b) => b.lane === lane)
    setActive({ key: keyOf(entry), lane: kept ? lane : largest(entry) })
  }
  const goToLane = (step: number) => {
    const x = layout.xs[index] ?? 0
    for (
      let lane = (active?.lane ?? (step > 0 ? -1 : lanes.length)) + step;
      lane >= 0 && lane < lanes.length;
      lane += step
    ) {
      const nearest = layout.bubbles
        .filter((b) => b.lane === lane)
        .sort((a, b) => Math.abs(a.x - x) - Math.abs(b.x - x))[0]
      if (nearest) return setActive({ key: keyOf(entries[nearest.entry]), lane })
    }
  }

  const onKeyDown = (e: KeyboardEvent) => {
    if (e.altKey || e.ctrlKey || e.metaKey) return
    if (e.key === 'ArrowRight') goTo(index + 1, active?.lane ?? null)
    else if (e.key === 'ArrowLeft') goTo(index < 0 ? 0 : index - 1, active?.lane ?? null)
    else if (e.key === 'ArrowDown') goToLane(1)
    else if (e.key === 'ArrowUp') goToLane(-1)
    else if (e.key === 'Home') goTo(0, active?.lane ?? null)
    else if (e.key === 'End') goTo(entries.length - 1, active?.lane ?? null)
    else if (e.key === 'Enter' && shown) open(active)
    else return
    e.preventDefault()
  }

  const ring = shown && {
    x: bubble?.x ?? layout.xs[index],
    y: bubble?.y ?? layout.top + 14,
    radius: bubble ? bubble.radius : 8,
  }
  const place = () => {
    if (!ring || !box.current) return null
    const bounds = box.current.getBoundingClientRect()
    const left = Math.max(GAP, Math.min(bounds.left + ring.x - ring.radius, window.innerWidth - 320 - GAP))
    const below = bounds.top + ring.y + ring.radius + GAP
    return below < window.innerHeight * 0.6
      ? { top: below, left }
      : { bottom: window.innerHeight - (bounds.top + ring.y - ring.radius - GAP), left }
  }
  const placed = place()

  return (
    <div ref={box} className="timeline-chart">
      <div className="timeline-plot-box" style={{ height: layout.height }}>
        <div
          className={clsx('timeline-plot', pointed && active && 'timeline-plot-pointing')}
          role="listbox"
          aria-label="Timeline"
          aria-activedescendant={shown ? 'timeline-active' : undefined}
          tabIndex={0}
          onMouseMove={(e) => {
            setPointed(true)
            setActive(at(e) ?? (focused ? active : null))
          }}
          onMouseLeave={() => setPointed(false)}
          onClick={(e) => open(at(e))}
          onKeyDown={onKeyDown}
          onFocus={() => {
            setFocused(true)
            if (index < 0 && entries.length > 0) goTo(entries.length - 1, null)
          }}
          onBlur={() => setFocused(false)}
        >
          <canvas
            ref={canvas}
            className="timeline-canvas"
            style={{ width: layout.width, height: layout.height }}
            aria-hidden="true"
          />
          {ring && (
            <div
              id="timeline-active"
              className="timeline-active"
              role="option"
              aria-selected="true"
              aria-label={describe(shown)}
              aria-posinset={index + 1}
              aria-setsize={entries.length}
              style={{
                left: ring.x - ring.radius - 2,
                top: ring.y - ring.radius - 2,
                width: 2 * ring.radius + 4,
                height: 2 * ring.radius + 4,
              }}
            />
          )}
        </div>
        <ul className="timeline-lanes" aria-label="Folders">
          {layout.lanes.map(({ lane, label, y }) => (
            <li key={lane.path ?? '\0'} style={{ top: y - 11, width: layout.gutter - 8 }}>
              <button
                type="button"
                className="timeline-lane"
                disabled={lane.path === null || lane.path === folder}
                onClick={() => lane.path !== null && onFolder(lane.path)}
              >
                <span className={`timeline-dot timeline-dot-${lane.slot || 'other'}`} />
                <PathText path={label} />
              </button>
            </li>
          ))}
        </ul>
      </div>
      {shown && placed && (
        <HoverCardPanel {...placed} root={themed}>
          {card(shown)}
        </HoverCardPanel>
      )}
    </div>
  )
}

export default TimelineChart
