import type { CSSProperties } from 'react'
import type { GraphNode } from '@current-client-bindings/app'
import { plural } from '../../lib/plural'
import CommitGraph from './CommitGraph'
import './WorkingChangesRow.scss'

interface WorkingChangesRowProps {
  node: GraphNode
  laneCount: number
  rowHeight: number
  changedFiles: number
  style: CSSProperties
  onOpen: () => void
}

// Uncommitted work above HEAD, where the next commit will go. Shares the
// commit row's grid so the graph stays aligned.
function WorkingChangesRow({ node, laneCount, rowHeight, changedFiles, style, onOpen }: WorkingChangesRowProps) {
  return (
    <div
      className="commit-row commit-row-working"
      role="button"
      tabIndex={0}
      title="Open Changes"
      style={style}
      onClick={onOpen}
      onKeyDown={(e) => {
        if (e.key === 'Enter' || e.key === ' ') {
          e.preventDefault()
          onOpen()
        }
      }}
    >
      <div className="commit-row-graph">
        <CommitGraph node={node} laneCount={laneCount} rowHeight={rowHeight} dashed />
      </div>
      <div className="commit-row-commit">
        <span className="commit-row-working-label">{plural(changedFiles, 'file')} changed</span>
      </div>
    </div>
  )
}

export default WorkingChangesRow
