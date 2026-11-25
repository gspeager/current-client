import { useLaneColors } from '../../lib/laneColor'
import { GRAPH_LINE, graphShapes, LANE_WIDTH, type GraphRow, type GraphShape } from './graphShapes'

interface CommitGraphProps extends GraphRow {
  laneCount: number
}

function shapeElement(shape: GraphShape, i: number) {
  switch (shape.kind) {
    case 'line': {
      const { x1, y1, x2, y2, color } = shape
      return <line key={i} x1={x1} y1={y1} x2={x2} y2={y2} style={{ stroke: color }} strokeWidth={GRAPH_LINE} />
    }
    case 'path':
      return <path key={i} d={shape.d} fill="none" style={{ stroke: shape.color }} strokeWidth={GRAPH_LINE} />
    case 'circle': {
      const { x, y, radius, fill = 'none', stroke, dash } = shape
      return (
        <circle
          key={i}
          cx={x}
          cy={y}
          r={radius}
          style={{ fill, stroke }}
          strokeWidth={stroke && GRAPH_LINE}
          strokeDasharray={dash && `${dash} ${dash}`}
        />
      )
    }
  }
}

function CommitGraph({ laneCount, ...row }: CommitGraphProps) {
  const { laneColor } = useLaneColors()
  const { node, rowHeight, laneWidth = LANE_WIDTH } = row
  return (
    <svg width={Math.max(laneCount, node.lane + 1) * laneWidth} height={rowHeight} style={{ display: 'block' }}>
      {graphShapes(row, laneColor).map(shapeElement)}
    </svg>
  )
}

export default CommitGraph
