import type { KeyboardEvent, MouseEvent } from 'react'
import { Ellipsis, Pencil, X } from 'lucide-react'
import type {
  BranchChange,
  BranchInfo,
  CIMark as CIMarkInfo,
  ConflictStateInfo,
  OverlapInfo,
} from '@current-client-bindings/app'
import type { Links } from '@current-client-bindings/core/hostlink'
import { rowKeyDown } from '../../components/controls/ContextMenu'
import BranchPill from '../../components/git/BranchPill'
import ChangeMarker from '../../components/git/ChangeMarker'
import { ciLabel } from '../../components/git/CIIcon'
import CIMark from '../host-changes/CIMark'
import { NoteMark } from '../notes/notes'
import { operationBadge } from '../../lib/conflictWords'
import { relativeTime } from '../../lib/relativeTime'
import type { ReferenceCards } from '../../lib/useReferenceCards'
import ConflictsBadge from './ConflictsBadge'

interface BranchRowProps {
  repoPath: string
  branch: BranchInfo
  color: string
  dirty: boolean
  // Why a checkout can't happen now, such as an operation in progress.
  blocked: string | null
  conflict: ConflictStateInfo | null
  hostLinks: Links | null
  referenceCards: ReferenceCards
  change: BranchChange | undefined
  ciMark: CIMarkInfo | undefined
  // The environments the tip is deployed to.
  reached: string[]
  overlap?: OverlapInfo
  onCheckout: () => void
  onMenu: (e: MouseEvent | KeyboardEvent) => void
  onRename: () => void
  onDelete: () => void
}

function BranchRow({
  repoPath,
  branch: b,
  color,
  dirty,
  blocked,
  conflict,
  hostLinks,
  referenceCards,
  change,
  ciMark,
  reached,
  overlap,
  onCheckout,
  onMenu,
  onRename,
  onDelete,
}: BranchRowProps) {
  const isBlocked = !!blocked && !b.current
  const checkout = () => !b.current && !blocked && onCheckout()
  // What the row has no room for: where the tip is deployed, beside CI and age.
  const summary = [
    b.name,
    ciMark?.status && ciLabel(ciMark.status),
    reached.length > 0 && `on ${reached.join(', ')}`,
    b.lastCommitDate && relativeTime(new Date(b.lastCommitDate)),
  ]
    .filter(Boolean)
    .join(' · ')
  const incoming = !!conflict && b.name === conflict.theirs
  return (
    <li
      className={`branch-row${b.current ? ' branch-row-current' : ''}${isBlocked ? ' branch-row-blocked' : ''}`}
      onDoubleClick={checkout}
      onContextMenu={onMenu}
      role="button"
      tabIndex={0}
      aria-disabled={isBlocked ? true : undefined}
      title={isBlocked ? blocked : b.current ? summary : `${summary}\nDouble-click to check out`}
      onKeyDown={rowKeyDown(checkout, onMenu)}
    >
      <BranchPill
        name={b.name}
        color={color}
        current={b.current}
        dirty={dirty}
        hostLinks={hostLinks}
        referenceCards={referenceCards}
      />
      <NoteMark kind="branch" target={b.name} />
      {conflict && b.current && <span className="branch-row-mode">{operationBadge(conflict.operation)}</span>}
      {incoming && !b.current && <span className="branch-row-mode branch-row-mode-incoming">incoming</span>}
      {!b.current && <ChangeMarker change={change} />}
      {!change && <CIMark repoPath={repoPath} mark={ciMark} />}
      {!incoming && <ConflictsBadge overlap={overlap} />}
      {!b.current && b.worktreePath && (
        <span className="branch-row-worktree" title={`Checked out in the worktree at ${b.worktreePath}`}>
          worktree
        </span>
      )}
      <span className="branch-row-meta">
        {b.ahead > 0 && <span className="branch-row-ahead">↑{b.ahead}</span>}
        {b.behind > 0 && <span className="branch-row-behind">↓{b.behind}</span>}
        {b.lastCommitDate && <span className="branch-row-time">{relativeTime(new Date(b.lastCommitDate))}</span>}
      </span>
      <span className="branch-row-actions">
        <button type="button" onClick={onMenu} aria-label={`More actions for ${b.name}`} title="More actions">
          <Ellipsis size={16} strokeWidth={1.75} />
        </button>
        <button
          type="button"
          onClick={(e) => {
            e.stopPropagation()
            onRename()
          }}
          aria-label={`Rename ${b.name}`}
          title="Rename"
        >
          <Pencil size={16} strokeWidth={1.75} />
        </button>
        {!b.current && (
          <button
            type="button"
            onClick={(e) => {
              e.stopPropagation()
              onDelete()
            }}
            aria-label={`Delete ${b.name}`}
            title="Delete"
          >
            <X size={16} strokeWidth={1.75} />
          </button>
        )}
      </span>
    </li>
  )
}

export default BranchRow
