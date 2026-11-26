import { cut, type Painter, type Picture } from '../../lib/picture'
import { plural } from '../../lib/plural'
import { readMapColors, type MapColors } from './drawMap'
import type { TimelineLayout } from './timelineLayout'

export interface TimelineColors extends MapColors {
  primary: string
  outline: string
  grid: string
  rule: string
}

const DATA = { face: 'data' } as const
const FILL_OPACITY = 0.38

export function readTimelineColors(element: Element): TimelineColors {
  const style = getComputedStyle(element)
  const token = (name: string) => style.getPropertyValue(name).trim()
  return {
    ...readMapColors(element),
    primary: token('--primary'),
    outline: token('--outline'),
    grid: token('--surface-high'),
    rule: token('--surface-container'),
  }
}

// The lane names are left out on screen, where they're buttons.
export function drawTimeline(painter: Painter, layout: TimelineLayout, colors: TimelineColors, names: boolean) {
  const { width, gutter, top, bottom, entries, xs, bubbles } = layout
  const slotOf = (lane: number) => layout.lanes[lane].lane.slot
  for (const { x, label } of layout.ticks) {
    painter.line(x, top, x, bottom, colors.grid)
    painter.text(x, bottom + 18, label, { ...DATA, size: 11, color: colors.muted, align: 'center' })
  }
  for (const { y } of layout.lanes) painter.line(gutter, y, width - 8, y, colors.rule)

  const committed = entries.filter((entry) => entry.sha !== null).length
  painter.text(12, top + 18, plural(committed, 'commit'), { ...DATA, color: colors.muted })
  entries.forEach((entry, i) => {
    if (entry.sha !== null) painter.rect(xs[i] - 1, top + 8, 2, 12, colors.outline)
  })

  for (const tag of layout.tags) {
    painter.path(`M${tag.x + tag.width / 2} ${tag.y + 16}V${bottom}`, { stroke: colors.outline, dash: 2 })
    painter.rect(tag.x, tag.y, tag.width, 16, colors.folder, 3)
    painter.text(tag.x + 4, tag.y + 12, tag.name, { ...DATA, color: colors.name })
  }

  painter.within(0, 0, FILL_OPACITY, () => {
    for (const bubble of bubbles) {
      if (entries[bubble.entry].sha !== null) {
        painter.circle(bubble.x, bubble.y, bubble.radius, { fill: colors.steps[slotOf(bubble.lane)][1] })
      }
    }
  })
  for (const bubble of bubbles) {
    const dash = entries[bubble.entry].sha === null ? 3 : undefined
    painter.circle(bubble.x, bubble.y, bubble.radius, { stroke: colors.steps[slotOf(bubble.lane)][2], dash })
  }

  if (layout.now !== null) {
    painter.line(layout.now, top - 4, layout.now, bottom, colors.primary)
    painter.text(layout.now, top - 8, 'now', { ...DATA, size: 11, color: colors.primary, align: 'center' })
  }

  if (!names) return
  for (const { lane, label, y } of layout.lanes) {
    painter.circle(16, y, 4, { fill: colors.steps[lane.slot][1] })
    painter.text(28, y + 4, cut(label, gutter - 36, DATA, 'left'), { ...DATA, color: colors.name })
  }
}

export function timelinePicture(layout: TimelineLayout, colors: TimelineColors): Picture {
  return {
    width: layout.width,
    height: layout.height,
    draw: (painter) => {
      painter.rect(0, 0, layout.width, layout.height, colors.ground)
      drawTimeline(painter, layout, colors, true)
    },
  }
}
