import type { GraphNode } from '@current-client-bindings/app'
import type { RowLaneColors } from './laneOwners'

// A row of the graph, on screen and in a saved picture.
export type GraphShape =
  | { kind: 'line'; x1: number; y1: number; x2: number; y2: number; color: string }
  | { kind: 'path'; d: string; color: string }
  | { kind: 'circle'; x: number; y: number; radius: number; fill?: string; stroke?: string; dash?: number }

export const LANE_WIDTH = 18
export const GRAPH_LINE = 1.5
const NODE_RADIUS = 4

export interface GraphRow {
  node: GraphNode
  rowHeight: number
  laneWidth?: number
  // The working tree's node: outlined and dashed, not a commit.
  dashed?: boolean
  // Who owns each lane at this row; without it, lanes color by position.
  colors?: RowLaneColors
  // HEAD's node: ringed, the graph's one cyan mark.
  head?: boolean
}

export function graphShapes(
  { node, rowHeight, laneWidth = LANE_WIDTH, dashed = false, colors, head = false }: GraphRow,
  laneColor: (lane: number) => string,
): GraphShape[] {
  const above = (lane: number) => colors?.before.get(lane) ?? laneColor(lane)
  const beneath = (lane: number) => colors?.after.get(lane) ?? laneColor(lane)
  const laneX = (lane: number) => lane * laneWidth + laneWidth / 2
  const nodeColor = colors?.node ?? laneColor(node.lane)
  const mid = rowHeight / 2
  const ownX = laneX(node.lane)
  // An outlined node has no fill to hide the line, so it starts at the rim.
  const below = dashed ? mid + NODE_RADIUS : mid
  const down = (x: number, y1: number, y2: number, color: string): GraphShape => ({
    kind: 'line',
    x1: x,
    y1,
    x2: x,
    y2,
    color,
  })
  const shapes: GraphShape[] = [
    ...node.passThrough.map((lane) => down(laneX(lane), 0, rowHeight, above(lane))),
    ...node.convergingLanes.map((lane): GraphShape => {
      const x = laneX(lane)
      return { kind: 'path', d: `M ${x} 0 C ${x} ${mid}, ${ownX} ${mid}, ${ownX} ${mid}`, color: above(lane) }
    }),
  ]
  if (node.hasChild) shapes.push(down(ownX, 0, mid, above(node.lane)))
  for (const lane of node.parentLanes) {
    const x = laneX(lane)
    shapes.push(
      lane === node.lane
        ? down(ownX, below, rowHeight, nodeColor)
        : {
            kind: 'path',
            d: `M ${ownX} ${below} C ${ownX} ${mid}, ${x} ${mid}, ${x} ${rowHeight}`,
            color: beneath(lane),
          },
    )
  }
  if (dashed) return [...shapes, { kind: 'circle', x: ownX, y: mid, radius: NODE_RADIUS, stroke: nodeColor, dash: 2 }]
  if (head) shapes.push({ kind: 'circle', x: ownX, y: mid, radius: NODE_RADIUS + 3, stroke: 'var(--primary)' })
  return [...shapes, { kind: 'circle', x: ownX, y: mid, radius: NODE_RADIUS, fill: nodeColor }]
}
