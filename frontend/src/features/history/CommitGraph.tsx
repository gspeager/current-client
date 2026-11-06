import type { GraphNode } from '@current-client-bindings/app'
import { useLaneColors } from '../../lib/laneColor'
import type { RowLaneColors } from './laneOwners'

interface CommitGraphProps {
  node: GraphNode
  laneCount: number
  rowHeight: number
  laneWidth?: number
  // The working tree's node: outlined and dashed, not a commit.
  dashed?: boolean
  // Who owns each lane at this row; without it, lanes color by position.
  colors?: RowLaneColors
  // HEAD's node: ringed, the graph's one cyan mark.
  head?: boolean
}

function laneX(lane: number, laneWidth: number): number {
  return lane * laneWidth + laneWidth / 2
}

const NODE_RADIUS = 4

function CommitGraph({ node, laneCount, rowHeight, laneWidth = 18, dashed = false, colors, head = false }: CommitGraphProps) {
  const { laneColor } = useLaneColors()
  const above = (lane: number) => colors?.before.get(lane) ?? laneColor(lane)
  const beneath = (lane: number) => colors?.after.get(lane) ?? laneColor(lane)
  const nodeColor = colors?.node ?? laneColor(node.lane)
  const mid = rowHeight / 2
  const ownX = laneX(node.lane, laneWidth)
  // An outlined node has no fill to hide the line, so it starts at the rim.
  const below = dashed ? mid + NODE_RADIUS : mid

  return (
    <svg width={Math.max(laneCount, node.lane + 1) * laneWidth} height={rowHeight} style={{ display: 'block' }}>
      {node.passThrough.map((lane) => {
        const x = laneX(lane, laneWidth)
        return (
          <line
            key={`pt-${lane}`}
            x1={x}
            y1={0}
            x2={x}
            y2={rowHeight}
            style={{ stroke: above(lane) }}
            strokeWidth={1.5}
          />
        )
      })}

      {node.convergingLanes.map((lane) => {
        const x = laneX(lane, laneWidth)
        return (
          <path
            key={`conv-${lane}`}
            d={`M ${x} 0 C ${x} ${mid}, ${ownX} ${mid}, ${ownX} ${mid}`}
            fill="none"
            style={{ stroke: above(lane) }}
            strokeWidth={1.5}
          />
        )
      })}

      {node.hasChild && (
        <line x1={ownX} y1={0} x2={ownX} y2={mid} style={{ stroke: above(node.lane) }} strokeWidth={1.5} />
      )}

      {node.parentLanes.map((lane, i) => {
        if (lane === node.lane) {
          return (
            <line
              key={`pl-${i}`}
              x1={ownX}
              y1={below}
              x2={ownX}
              y2={rowHeight}
              style={{ stroke: nodeColor }}
              strokeWidth={1.5}
            />
          )
        }
        const x = laneX(lane, laneWidth)
        return (
          <path
            key={`pl-${i}`}
            d={`M ${ownX} ${below} C ${ownX} ${mid}, ${x} ${mid}, ${x} ${rowHeight}`}
            fill="none"
            style={{ stroke: beneath(lane) }}
            strokeWidth={1.5}
          />
        )
      })}

      {dashed ? (
        <circle
          cx={ownX}
          cy={mid}
          r={NODE_RADIUS}
          fill="none"
          strokeWidth={1.5}
          strokeDasharray="2 2"
          style={{ stroke: nodeColor }}
        />
      ) : (
        <>
          {head && (
            <circle cx={ownX} cy={mid} r={NODE_RADIUS + 3} fill="none" strokeWidth={1.5} style={{ stroke: 'var(--primary)' }} />
          )}
          <circle cx={ownX} cy={mid} r={NODE_RADIUS} style={{ fill: nodeColor }} />
        </>
      )}
    </svg>
  )
}

export default CommitGraph
