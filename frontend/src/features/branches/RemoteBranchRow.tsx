import type { KeyboardEvent, MouseEvent } from 'react'
import { Cloud } from 'lucide-react'
import type { OverlapInfo } from '@current-client-bindings/app'
import { rowKeyDown } from '../../components/controls/ContextMenu'
import ConflictsBadge from './ConflictsBadge'

interface RemoteBranchRowProps {
  name: string
  // Why a checkout can't happen now, such as an operation in progress.
  blocked: string | null
  incoming: boolean
  local: boolean
  overlap?: OverlapInfo
  onCheckout: () => void
  onMenu: (e: MouseEvent | KeyboardEvent) => void
}

function RemoteBranchRow({ name, blocked, incoming, local, overlap, onCheckout, onMenu }: RemoteBranchRowProps) {
  const checkout = () => !blocked && onCheckout()
  return (
    <li
      className={blocked ? 'branch-row branch-row-remote branch-row-blocked' : 'branch-row branch-row-remote'}
      onDoubleClick={checkout}
      onContextMenu={onMenu}
      role="button"
      tabIndex={0}
      aria-disabled={blocked ? true : undefined}
      title={blocked ?? 'Double-click to check out'}
      onKeyDown={rowKeyDown(checkout, onMenu)}
    >
      <Cloud size={14} strokeWidth={1.5} className="branch-row-remote-icon" />
      <span className="branch-row-remote-name">{name}</span>
      {incoming && <span className="branch-row-mode branch-row-mode-incoming">incoming</span>}
      <ConflictsBadge overlap={overlap} />
      {!local && <span className="branch-row-meta">not local</span>}
    </li>
  )
}

export default RemoteBranchRow
