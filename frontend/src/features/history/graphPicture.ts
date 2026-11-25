import type { CommitInfo, GraphNode, IdentityInfo, RefInfo } from '@current-client-bindings/app'
import { cut, textWidth, type Painter, type Picture } from '../../lib/picture'
import { shortSha } from '../../lib/text'
import { GRAPH_LINE, graphShapes, LANE_WIDTH } from './graphShapes'
import type { RowLaneColors } from './laneOwners'
import { pairUpstreams } from './refPills'

export interface GraphPictureRow {
  // Without a commit it's the working tree's, and text says what it holds.
  commit?: CommitInfo
  text?: string
  node?: GraphNode
  colors?: RowLaneColors
  head?: boolean
  dimmed?: boolean
  refs?: RefInfo[]
  identity?: IdentityInfo
  time?: string
}

export interface GraphPictureOptions {
  rows: GraphPictureRow[]
  width: number
  rowHeight: number
  laneCount: number
  // Leaves out who made each commit.
  names: boolean
  upstreams: Record<string, string>
  laneColor: (lane: number) => string
  refColor: (ref: RefInfo) => string
  // A color, or a token's value.
  color: (color: string) => string
}

const PAD = 12
const GAP = 8
const AUTHOR = 124
const HASH = 64
const TIME = 72
const PILL = 18
const UI = { face: 'ui', size: 13 } as const
const DATA = { face: 'data' } as const
const SMALL = { face: 'data', size: 10, weight: 600 } as const
// The subject keeps at least this much of its column.
const REFS_SHARE = 0.36
const MOST_REFS = 6
// As IdentityBadge.scss has it, which doesn't follow the theme.
const BADGE_INK = '#0a0e16'

const isBranch = (ref: RefInfo) => ref.kind === 'branch' || ref.kind === 'remote'

// The commit list as one picture: what's a small SVG and HTML for each row on screen.
export function graphPicture(options: GraphPictureOptions): Picture {
  const { rows, width, rowHeight } = options
  return {
    width,
    height: rows.length * rowHeight,
    draw: (painter) =>
      rows.forEach((row, i) =>
        painter.within(0, i * rowHeight, row.dimmed ? 0.4 : 1, () => drawRow(painter, row, options)),
      ),
  }
}

function drawRow(painter: Painter, row: GraphPictureRow, options: GraphPictureOptions) {
  const { width, rowHeight, laneCount, names, color } = options
  const baseline = rowHeight / 2 + 4
  const graph = laneCount * LANE_WIDTH
  if (row.node) drawGraph(painter, row, options)
  const right = width - PAD
  const time = right - TIME
  const hash = time - GAP - HASH
  const author = names ? hash - GAP - AUTHOR : hash
  let x = PAD + graph + GAP
  const end = author - GAP
  if (!row.commit) {
    const text = cut(row.text ?? '', end - x, UI, 'right')
    painter.text(x, baseline, text, { ...UI, color: color('var(--on-surface-variant)') })
    return
  }
  if (row.head) x += drawHead(painter, x, rowHeight, options) + 4
  x = drawRefs(painter, row.refs ?? [], x, x + (end - x) * REFS_SHARE, options)
  const subject = cut(row.commit.subject, end - x, UI, 'right')
  painter.text(x, baseline, subject, { ...UI, color: color('var(--on-surface)') })
  const quiet = color('var(--on-surface-variant)')
  if (names) {
    if (row.identity) {
      painter.circle(author + 10, rowHeight / 2, 10, { fill: row.identity.color })
      painter.text(author + 10, rowHeight / 2 + 3.5, row.identity.initials, {
        ...SMALL,
        align: 'center',
        color: BADGE_INK,
      })
    }
    painter.text(author + 26, baseline, cut(row.commit.authorName, AUTHOR - 26, UI, 'right'), { ...UI, color: quiet })
  }
  painter.text(hash, baseline, shortSha(row.commit.sha), { ...DATA, color: quiet })
  painter.text(right, baseline, row.time ?? '', { face: 'ui', color: quiet, align: 'right' })
}

function drawGraph(painter: Painter, row: GraphPictureRow, { rowHeight, laneColor, color }: GraphPictureOptions) {
  const shapes = graphShapes(
    { node: row.node!, rowHeight, colors: row.colors, head: row.head, dashed: !row.commit },
    laneColor,
  )
  painter.within(PAD, 0, 1, () => {
    for (const shape of shapes) {
      if (shape.kind === 'line') {
        painter.line(shape.x1, shape.y1, shape.x2, shape.y2, color(shape.color), GRAPH_LINE)
      } else if (shape.kind === 'path') {
        painter.path(shape.d, { stroke: color(shape.color), width: GRAPH_LINE })
      } else {
        const { x, y, radius, fill, stroke, dash } = shape
        painter.circle(x, y, radius, {
          fill: fill && color(fill),
          stroke: stroke && color(stroke),
          width: GRAPH_LINE,
          dash,
        })
      }
    }
  })
}

function drawHead(painter: Painter, x: number, rowHeight: number, { color }: GraphPictureOptions): number {
  const width = textWidth('HEAD', SMALL) + 8
  painter.rect(x, (rowHeight - PILL) / 2 + 1, width, PILL - 2, color('var(--primary)'), 3)
  painter.text(x + 4, rowHeight / 2 + 3.5, 'HEAD', { ...SMALL, color: color('var(--on-primary)') })
  return width
}

// Branches and tags up to an edge, with a count of any there's no room for. Answers
// where they end.
function drawRefs(
  painter: Painter,
  refs: RefInfo[],
  from: number,
  edge: number,
  { rowHeight, upstreams, refColor, color }: GraphPictureOptions,
): number {
  const pills = pairUpstreams(refs, upstreams)
  const top = (rowHeight - PILL) / 2
  const baseline = rowHeight / 2 + 4
  let x = from
  let left = pills.length
  for (const { ref } of pills.slice(0, MOST_REFS)) {
    const branch = isBranch(ref)
    const before = branch ? 17 : 6
    const room = edge - x - before - 8
    const name = cut(ref.name, room, DATA, 'right', 4)
    if (!name) break
    const width = before + textWidth(name, DATA) + 8
    painter.rect(x, top, width, PILL, color('var(--surface-high)'), branch ? PILL / 2 : 3)
    if (branch) painter.circle(x + 9, rowHeight / 2, 3, { fill: color(refColor(ref)) })
    painter.text(x + before, baseline, name, {
      ...DATA,
      color: color(branch ? 'var(--on-surface)' : 'var(--on-surface-variant)'),
    })
    x += width + 4
    left--
  }
  if (left > 0 && pills.length > 0) {
    const more = `+${left}`
    painter.text(x, baseline, more, { ...DATA, color: color('var(--outline)') })
    x += textWidth(more, DATA) + 6
  }
  return x === from ? from : x + 2
}
