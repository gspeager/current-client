import { cut, TEXT_SIZE, textWidth, type Painter } from '../../lib/picture'
import { HEADER, type MapLayout, type Tile } from './mapLayout'

export interface MapColors {
  ground: string
  folder: string
  name: string
  muted: string
  onFolder: string
  warning: string
  // By slot, 0 for other, then by step from 1.
  steps: string[][]
}

const DATA = { face: 'data' } as const
const PAD = 6
// Less of a name than this says nothing.
const SHORTEST = 8

export function readMapColors(element: Element): MapColors {
  const style = getComputedStyle(element)
  const token = (name: string) => style.getPropertyValue(name).trim()
  const steps = (folder: string) => [1, 2, 3, 4].map((step) => token(`--folder-${folder}-${step}`))
  return {
    ground: token('--surface-lowest'),
    folder: token('--surface-container'),
    name: token('--on-surface'),
    muted: token('--on-surface-variant'),
    onFolder: token('--on-folder'),
    warning: token('--warning'),
    steps: [steps('other'), ...[1, 2, 3, 4, 5, 6, 7, 8].map((slot) => steps(String(slot)))],
  }
}

function drawTile(painter: Painter, tile: Tile, colors: MapColors) {
  const { x0, y0, item } = tile
  const width = tile.x1 - x0
  const height = tile.y1 - y0
  if (width <= 0 || height <= 0) return
  const color = colors.steps[item.slot][item.step - 1]
  const deleted = item.status === 'D'
  if (deleted) {
    painter.frame(x0, y0, width, height, color, 1)
    if (width > 6 && height > 6) painter.line(x0, tile.y1, tile.x1, y0, color)
  } else {
    painter.rect(x0, y0, width, height, color)
  }
  if (item.status === 'R' && width > 10 && height > 10) {
    painter.frame(x0 + 2, y0 + 2, width - 4, height - 4, colors.onFolder, 1)
  }
  if (item.conflict && width > 4 && height > 4) painter.frame(x0, y0, width, height, colors.warning, 3)
  if (height < TEXT_SIZE + 6 || width < 24) return
  const ink = deleted ? colors.muted : colors.onFolder
  const mark = item.status ? `${item.status}${item.conflict ? '!' : ''}` : ''
  const baseline = y0 + TEXT_SIZE + 3
  const room = width - 2 * PAD
  if (mark && textWidth(mark, DATA) <= room) painter.text(tile.x1 - PAD, baseline, mark, { color: ink, align: 'right' })
  const name = cut(tile.name, mark ? room - textWidth(`${mark} `, DATA) : room, DATA, 'right', SHORTEST)
  if (name) painter.text(x0 + PAD, baseline, name, { color: ink })
}

export function drawMap(painter: Painter, layout: MapLayout, colors: MapColors) {
  painter.rect(0, 0, layout.width, layout.height, colors.ground)
  for (const folder of layout.folders) {
    if (!folder.header) continue
    painter.rect(folder.x0, folder.y0, folder.x1 - folder.x0, folder.y1 - folder.y0, colors.folder)
    const name = cut(folder.name, folder.x1 - folder.x0 - 2 * PAD, DATA, 'left')
    if (name) painter.text(folder.x0 + PAD, folder.y0 + HEADER - 6, name, { color: colors.name })
  }
  for (const tile of layout.tiles) drawTile(painter, tile, colors)
}
