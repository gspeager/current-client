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

// What a picture is drawn with: a canvas on screen, and an SVG when it's saved.
export interface Painter {
  rect: (x: number, y: number, width: number, height: number, fill: string) => void
  frame: (x: number, y: number, width: number, height: number, color: string, line: number) => void
  line: (x0: number, y0: number, x1: number, y1: number, color: string) => void
  text: (x: number, baseline: number, text: string, color: string) => void
}

export const FONT_SIZE = 12
// Of the data font, which is monospaced.
const CHARACTER = FONT_SIZE * 0.6
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

const fits = (width: number) => Math.floor((width - 2 * PAD) / CHARACTER)

// A path keeps its end, where the name is.
function cutLeft(text: string, room: number): string {
  return text.length <= room ? text : room < 2 ? '' : `…${text.slice(text.length - room + 1)}`
}

function cutRight(text: string, room: number): string {
  return text.length <= room ? text : room < SHORTEST ? '' : `${text.slice(0, room - 1)}…`
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
  if (height < FONT_SIZE + 6 || width < 24) return
  const ink = deleted ? colors.muted : colors.onFolder
  const mark = item.status ? `${item.status}${item.conflict ? '!' : ''}` : ''
  const room = fits(width)
  const name = cutRight(tile.name, mark ? room - mark.length - 1 : room)
  const baseline = y0 + FONT_SIZE + 3
  if (mark && room >= mark.length) painter.text(tile.x1 - PAD - mark.length * CHARACTER, baseline, mark, ink)
  if (name) painter.text(x0 + PAD, baseline, name, ink)
}

export function drawMap(painter: Painter, layout: MapLayout, colors: MapColors) {
  painter.rect(0, 0, layout.width, layout.height, colors.ground)
  for (const folder of layout.folders) {
    if (!folder.header) continue
    painter.rect(folder.x0, folder.y0, folder.x1 - folder.x0, folder.y1 - folder.y0, colors.folder)
    const name = cutLeft(folder.name, fits(folder.x1 - folder.x0))
    if (name) painter.text(folder.x0 + PAD, folder.y0 + HEADER - 6, name, colors.name)
  }
  for (const tile of layout.tiles) drawTile(painter, tile, colors)
}

export function canvasPainter(context: CanvasRenderingContext2D, font: string): Painter {
  context.font = `${FONT_SIZE}px ${font}`
  return {
    rect: (x, y, width, height, fill) => {
      context.fillStyle = fill
      context.fillRect(x, y, width, height)
    },
    frame: (x, y, width, height, color, line) => {
      context.strokeStyle = color
      context.lineWidth = line
      context.strokeRect(x + line / 2, y + line / 2, Math.max(0, width - line), Math.max(0, height - line))
    },
    line: (x0, y0, x1, y1, color) => {
      context.strokeStyle = color
      context.lineWidth = 1
      context.beginPath()
      context.moveTo(x0, y0)
      context.lineTo(x1, y1)
      context.stroke()
    },
    text: (x, baseline, text, color) => {
      context.fillStyle = color
      context.fillText(text, x, baseline)
    },
  }
}
