import { textWidth } from '../../lib/picture'
import type { Entry, Lane } from './timelineLanes'

export const LANE = 52
export const MAX_RADIUS = 24
export const MIN_RADIUS = 2.5
const TAG_ROW = 22
const MOST_TAG_ROWS = 2
export const TICK_ROW = 28
const AXIS = 28
const GUTTER = { least: 120, most: 260 }
const RIGHT = 16
const HOUR = 3_600_000
const DAY = 24 * HOUR
// The fewest pixels between two times on the axis.
const TICK_ROOM = 96

export interface PlacedBubble {
  entry: number
  lane: number
  x: number
  y: number
  radius: number
}

export interface PlacedTag {
  name: string
  x: number
  y: number
  width: number
}

export interface TimelineLayout {
  width: number
  height: number
  gutter: number
  lanes: { lane: Lane; label: string; y: number }[]
  entries: Entry[]
  // Each entry's x.
  xs: number[]
  // The largest drawn first, so a smaller one over it can still be pointed at.
  bubbles: PlacedBubble[]
  ticks: { x: number; label: string }[]
  tags: PlacedTag[]
  // Where the tick row and the lanes start and end.
  top: number
  bottom: number
  now: number | null
}

// A bubble's area is in proportion to its lines up to a reference, the size of all
// but the largest few, so one huge commit doesn't shrink the rest to dots.
export function radiusFinder(lines: number[]): (lines: number) => number {
  const sorted = [...lines].sort((a, b) => a - b)
  const reference = Math.max(1, sorted[Math.round((sorted.length - 1) * 0.95)] ?? 1)
  return (n) => Math.max(MIN_RADIUS, MAX_RADIUS * Math.sqrt(Math.min(n, reference) / reference))
}

const STEPS = [HOUR, 2 * HOUR, 3 * HOUR, 6 * HOUR, 12 * HOUR, DAY, 2 * DAY, 7 * DAY, 14 * DAY]
const MONTHS = [1, 3, 6, 12]

// Times on the hour, day, Monday or first of a month, local time, as far apart as
// the width needs.
export function timeTicks(start: number, end: number, width: number): { time: number; label: string }[] {
  const most = Math.max(1, Math.floor(width / TICK_ROOM))
  const span = end - start
  const step = STEPS.find((s) => span / s <= most)
  const times: number[] = []
  if (step !== undefined) {
    const at = new Date(start)
    if (step < DAY) at.setMinutes(0, 0, 0)
    else at.setHours(0, 0, 0, 0)
    if (step >= 7 * DAY) at.setDate(at.getDate() - ((at.getDay() + 6) % 7))
    for (; at.getTime() <= end; step < DAY ? at.setHours(at.getHours() + 1) : at.setDate(at.getDate() + step / DAY)) {
      if (at.getTime() >= start && (step >= DAY || at.getHours() % (step / HOUR) === 0)) times.push(at.getTime())
    }
  } else {
    const months = MONTHS.find((m) => span / (m * 30 * DAY) <= most) ?? Math.ceil(span / (most * 365 * DAY)) * 12
    const at = new Date(start)
    at.setHours(0, 0, 0, 0)
    at.setDate(1)
    at.setMonth(at.getMonth() - (at.getMonth() % months))
    for (; at.getTime() <= end; at.setMonth(at.getMonth() + months)) {
      if (at.getTime() >= start) times.push(at.getTime())
    }
  }
  const options: Intl.DateTimeFormatOptions =
    step === undefined
      ? { month: 'short', year: 'numeric' }
      : step < DAY
        ? { hour: 'numeric', minute: '2-digit' }
        : { month: 'short', day: 'numeric' }
  return times.map((time) => ({ time, label: new Date(time).toLocaleString(undefined, options) }))
}

export const laneName = (lane: Lane) => (lane.path === null ? 'Other folders' : lane.path || 'Top folder')

const DATA = { face: 'data' } as const

// From since, or the first entry, to now when the range reaches it, or else the last.
export function layoutTimeline(
  lanes: Lane[],
  entries: Entry[],
  tags: { name: string; sha: string }[],
  { since, now }: { since: number | null; now: number | null },
  width: number,
): TimelineLayout {
  const times = entries.map((entry) => entry.time)
  const end = now ?? Math.max(...times, since ?? 0)
  const start = Math.min(since ?? Math.min(...times, end), end - HOUR)
  const labels = lanes.map(laneName)
  const gutter = Math.round(
    Math.min(Math.max(GUTTER.least, ...labels.map((label) => textWidth(label, DATA) + 36)), GUTTER.most, width * 0.35),
  )
  const left = gutter + MAX_RADIUS
  const right = Math.max(left + 1, width - RIGHT - MAX_RADIUS)
  const xOf = (time: number) => left + ((time - start) / (end - start)) * (right - left)
  const xs = entries.map((entry) => xOf(entry.time))
  const nowX = now === null ? null : xOf(now)

  // Each row's taken stretches, as left and right edges; the first keeps room for "now".
  const rows: number[][] = [nowX === null ? [] : [nowX - 20, nowX + 20]]
  const placed: PlacedTag[] = []
  const tagged = new Map(tags.map((tag) => [tag.sha, tag.name]))
  entries.forEach((entry, i) => {
    const name = entry.sha && tagged.get(entry.sha)
    if (!name) return
    const tagWidth = textWidth(name, DATA) + 8
    const x = Math.min(Math.max(xs[i] - tagWidth / 2, gutter), width - tagWidth)
    for (let row = 0; row < MOST_TAG_ROWS; row++) {
      const taken = (rows[row] ??= [])
      let free = true
      for (let k = 0; k < taken.length; k += 2) free &&= x + tagWidth + 4 < taken[k] || x > taken[k + 1] + 4
      if (!free) continue
      taken.push(x, x + tagWidth)
      placed.push({ name, x, y: row * TAG_ROW + 3, width: tagWidth })
      return
    }
  })

  const top = Math.max(1, rows.filter((row) => row.length > 0).length) * TAG_ROW
  const first = top + TICK_ROW
  const bottom = first + lanes.length * LANE
  const radius = radiusFinder(entries.flatMap((entry) => entry.bubbles.map((bubble) => bubble.lines)))
  const bubbles = entries
    .flatMap((entry, i) =>
      entry.bubbles.map((bubble) => ({
        entry: i,
        lane: bubble.lane,
        x: xs[i],
        y: first + bubble.lane * LANE + LANE / 2,
        radius: radius(bubble.lines),
      })),
    )
    .sort((a, b) => b.radius - a.radius)
  return {
    width,
    height: bottom + AXIS,
    gutter,
    lanes: lanes.map((lane, i) => ({ lane, label: labels[i], y: first + i * LANE + LANE / 2 })),
    entries,
    xs,
    bubbles,
    ticks: timeTicks(start, end, right - left).map(({ time, label }) => ({ x: xOf(time), label })),
    tags: placed,
    top,
    bottom,
    now: nowX,
  }
}
