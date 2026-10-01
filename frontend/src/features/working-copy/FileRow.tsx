import type { CSSProperties, MouseEvent } from 'react'
import { X } from 'lucide-react'
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
  onDiscard?: () => void
  onContextMenu?: (e: MouseEvent) => void
  style?: CSSProperties
  indent?: number
  added?: number
  removed?: number
  binary?: boolean
}

function FileRow({
  status,
  label,
  selected,
  checked,
  checkLabel,
  onToggleChecked,
  onSelect,
  onDiscard,
  onContextMenu,
  style,
  indent = 0,
  added = 0,
  removed = 0,
  binary = false,
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
      <button type="button" className="file-row-path" onClick={onSelect}>
        <PathText path={label} />
      </button>
      <DiffStat added={added} removed={removed} binary={binary} />
      <StatusBadge status={status} />
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
    </div>
  )
}

export default FileRow
