import type { CSSProperties } from 'react'
import type { CurrentUserInfo, GraphNode } from '@current-client-bindings/app'
import { plural } from '../../lib/plural'
import IdentityBadge from '../../components/git/IdentityBadge'
import CommitGraph from './CommitGraph'
import './WorkingChangesRow.scss'

interface WorkingChangesRowProps {
  node: GraphNode
  laneCount: number
  rowHeight: number
  changedFiles: number
  // Who the next commit will be by, from Git's user.name and user.email.
  user?: CurrentUserInfo | null
  style: CSSProperties
  onOpen: () => void
}

// Uncommitted work above HEAD, where the next commit will go. Shares the
// commit row's grid so the graph stays aligned.
function WorkingChangesRow({ node, laneCount, rowHeight, changedFiles, user, style, onOpen }: WorkingChangesRowProps) {
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
      {user?.name && (
        <div className="commit-row-author">
          <IdentityBadge identity={user} name={user.name} email={user.email} size={18} />
          <span className="commit-row-author-name">{user.name}</span>
        </div>
      )}
    </div>
  )
}

export default WorkingChangesRow
