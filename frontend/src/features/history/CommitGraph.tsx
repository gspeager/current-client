import type { GraphNode } from '@current-client-bindings/app'
import { useLaneColors } from '../../lib/laneColor'

interface CommitGraphProps {
  node: GraphNode
  laneCount: number
  rowHeight: number
  laneWidth?: number
  // The working tree's node: outlined and dashed, not a commit.
  dashed?: boolean
}

function laneX(lane: number, laneWidth: number): number {
  return lane * laneWidth + laneWidth / 2
}

const NODE_RADIUS = 4

function CommitGraph({ node, laneCount, rowHeight, laneWidth = 18, dashed = false }: CommitGraphProps) {
  const { laneColor } = useLaneColors()
  const mid = rowHeight / 2
  const ownX = laneX(node.lane, laneWidth)

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
            style={{ stroke: laneColor(lane) }}
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
            style={{ stroke: laneColor(lane) }}
            strokeWidth={1.5}
          />
        )
      })}

      {node.hasChild && (
        <line x1={ownX} y1={0} x2={ownX} y2={mid} style={{ stroke: laneColor(node.lane) }} strokeWidth={1.5} />
      )}

      {node.parentLanes.map((lane, i) => {
        if (lane === node.lane) {
          return (
            <line
              key={`pl-${i}`}
              x1={ownX}
              y1={mid}
              x2={ownX}
              y2={rowHeight}
              style={{ stroke: laneColor(lane) }}
              strokeWidth={1.5}
            />
          )
        }
        const x = laneX(lane, laneWidth)
        return (
          <path
            key={`pl-${i}`}
            d={`M ${ownX} ${mid} C ${ownX} ${mid}, ${x} ${mid}, ${x} ${rowHeight}`}
            fill="none"
            style={{ stroke: laneColor(lane) }}
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
          style={{ stroke: laneColor(node.lane) }}
        />
      ) : (
        <circle cx={ownX} cy={mid} r={NODE_RADIUS} style={{ fill: laneColor(node.lane) }} />
      )}
    </svg>
  )
}

export default CommitGraph
