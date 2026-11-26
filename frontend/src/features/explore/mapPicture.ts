import { textWidth, type Painter, type Picture } from '../../lib/picture'
import { drawMap, type MapColors } from './drawMap'
import type { MapLayout } from './mapLayout'

export interface LegendEntry {
  label: string
  // Shown as a square of this color unless letter is set.
  color?: string
  letter?: string
}

const UI = { face: 'ui' } as const
const ROW = 22
const SQUARE = 10
const GAP = 16

const widthOf = ({ label, letter }: LegendEntry) =>
  (letter ? textWidth(letter, { face: 'data' }) : SQUARE) + 6 + textWidth(label, UI)

// The map as it's laid out on screen, over its legend.
export function mapPicture(layout: MapLayout, legend: LegendEntry[], colors: MapColors): Picture {
  const rows: LegendEntry[][] = [[]]
  let used = 0
  for (const entry of legend) {
    const width = widthOf(entry) + GAP
    if (used > 0 && used + width > layout.width - 24) {
      rows.push([])
      used = 0
    }
    rows[rows.length - 1].push(entry)
    used += width
  }
  return {
    width: layout.width,
    height: layout.height + rows.length * ROW + 12,
    draw: (painter) => {
      drawMap(painter, layout, colors)
      rows.forEach((entries, i) =>
        painter.within(12, layout.height + 8 + i * ROW, 1, () => drawLegend(painter, entries, colors)),
      )
    },
  }
}

function drawLegend(painter: Painter, entries: LegendEntry[], colors: MapColors) {
  let x = 0
  for (const entry of entries) {
    if (entry.color) painter.rect(x, 4, SQUARE, SQUARE, entry.color)
    if (entry.letter) painter.text(x, 13, entry.letter, { color: colors.name })
    const mark = entry.letter ? textWidth(entry.letter, { face: 'data' }) : SQUARE
    painter.text(x + mark + 6, 13, entry.label, { ...UI, color: colors.muted })
    x += widthOf(entry) + GAP
  }
}
