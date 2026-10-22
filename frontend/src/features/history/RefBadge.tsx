import type { MouseEvent } from 'react'
import type { BranchChange, RefInfo } from '@current-client-bindings/app'
import ChangeMarker from '../../components/git/ChangeMarker'
import './RefBadge.scss'

interface RefBadgeProps {
  refInfo: RefInfo
  color: string
  onClick?: (sha: string) => void
  active?: boolean
  change?: BranchChange
}

function RefBadge({ refInfo, color, onClick, active, change }: RefBadgeProps) {
  const classes = ['ref-badge']
  if (refInfo.kind === 'branch' || refInfo.kind === 'remote') classes.push('ref-badge-pill')
  if (onClick) classes.push('ref-badge-clickable')
  if (active) classes.push('ref-badge-active')
  const handleClick = onClick
    ? (e: MouseEvent) => {
        e.stopPropagation()
        onClick(refInfo.sha)
      }
    : undefined
  const badge = (
    <span
      className={classes.join(' ')}
      onClick={handleClick}
      title={onClick ? `Highlight ${refInfo.name}'s history in the graph` : refInfo.name}
      style={{ background: color }}
    >
      {refInfo.name}
    </span>
  )
  if (!change) return badge
  return (
    <span className="ref-badge-with-change">
      {badge}
      <ChangeMarker change={change} />
    </span>
  )
}

export default RefBadge
