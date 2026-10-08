import type { CSSProperties, KeyboardEvent, MouseEvent } from 'react'
import { Ellipsis, X } from 'lucide-react'
import { isMenuKey } from '../../components/controls/ContextMenu'
import Checkbox from '../../components/forms/Checkbox'
import DiffStat from '../../components/git/DiffStat'
import PathText from '../../components/git/PathText'
import StatusBadge from '../../components/git/StatusBadge'
import './FileRow.scss'

interface FileRowProps {
  status: string
  label: string
  selected: boolean
  checked: boolean
  checkLabel?: string
  onToggleChecked: () => void
  onSelect: (e: MouseEvent) => void
  onSelectMouseDown?: (e: MouseEvent) => void
  onDiscard?: () => void
  onContextMenu?: (e: MouseEvent | KeyboardEvent) => void
  style?: CSSProperties
  indent?: number
  added?: number
  removed?: number
  binary?: boolean
  // A short label for what kind of entry it is, such as "submodule".
  tag?: string
}

function FileRow({
  status,
  label,
  selected,
  checked,
  checkLabel,
  onToggleChecked,
  onSelect,
  onSelectMouseDown,
  onDiscard,
  onContextMenu,
  style,
  indent = 0,
  added = 0,
  removed = 0,
  binary = false,
  tag,
}: FileRowProps) {
  const rowStyle = indent > 0 ? { ...style, paddingLeft: `calc(var(--space-sm) + ${indent}px)` } : style

  return (
    <div
      className={selected ? 'file-row file-row-selected' : 'file-row'}
      onContextMenu={onContextMenu}
      style={rowStyle}
    >
      <Checkbox
        checked={checked}
        onChange={onToggleChecked}
        onClick={(e) => e.stopPropagation()}
        ariaLabel={checkLabel ?? (checked ? `Unstage ${label}` : `Stage ${label}`)}
      />
      <button
        type="button"
        className="file-row-path"
        onClick={onSelect}
        onMouseDown={onSelectMouseDown}
        onKeyDown={(e) => onContextMenu && isMenuKey(e) && onContextMenu(e)}
      >
        <PathText path={label} />
      </button>
      <span className="file-row-stats">
        {tag && <span className="file-row-tag">{tag}</span>}
        <DiffStat added={added} removed={removed} binary={binary} />
      </span>
      <StatusBadge status={status} />
      {(onContextMenu || onDiscard) && (
        <span className="file-row-actions">
          {onContextMenu && (
            <button type="button" onClick={onContextMenu} aria-label={`More actions for ${label}`} title="More actions">
              <Ellipsis size={14} strokeWidth={1.5} />
            </button>
          )}
          {onDiscard && (
            <button
              type="button"
              className="file-row-discard"
              onClick={onDiscard}
              aria-label={`Discard ${label}`}
              title="Discard"
            >
              <X size={14} strokeWidth={1.5} />
            </button>
          )}
        </span>
      )}
    </div>
  )
}

export default FileRow
