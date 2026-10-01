import type { MouseEvent } from 'react'
import type { RefInfo } from '@current-client-bindings/app'
import './RefBadge.scss'

interface RefBadgeProps {
  refInfo: RefInfo
  color: string
  onClick?: (sha: string) => void
  active?: boolean
}

function RefBadge({ refInfo, color, onClick, active }: RefBadgeProps) {
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
  return (
    <span
      className={classes.join(' ')}
      onClick={handleClick}
      title={onClick ? `Highlight ${refInfo.name}'s history in the graph` : undefined}
      style={{ background: color }}
    >
      {refInfo.name}
    </span>
  )
}

export default RefBadge
