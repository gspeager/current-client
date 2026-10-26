import type { MouseEvent } from 'react'
import { Cloud } from 'lucide-react'
import type { BranchChange, RefInfo } from '@current-client-bindings/app'
import ChangeMarker from '../../components/git/ChangeMarker'
import './RefBadge.scss'

interface RefBadgeProps {
  refInfo: RefInfo
  color: string
  onClick?: (sha: string) => void
  active?: boolean
  change?: BranchChange
  upstream?: RefInfo
}

// With `upstream`, the pill stands for the branch and its remote-tracking
// branch together, marked with a cloud and both named on hover.
function RefBadge({ refInfo, color, onClick, active, change, upstream }: RefBadgeProps) {
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
  const names = upstream ? `${refInfo.name} · ${upstream.name}` : refInfo.name
  const badge = (
    <span
      className={classes.join(' ')}
      onClick={handleClick}
      title={onClick ? `${names}\nHighlight ${refInfo.name}'s history in the graph` : names}
      style={{ background: color }}
    >
      {refInfo.name}
      {upstream && <Cloud size={10} strokeWidth={2} className="ref-badge-upstream" aria-label={upstream.name} />}
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
