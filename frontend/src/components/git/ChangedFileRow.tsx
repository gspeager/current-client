import clsx from 'clsx'
import type { KeyboardEvent, MouseEvent } from 'react'
import { isMenuKey } from '../controls/ContextMenu'
import type { ChangedFile as ChangedFileInfo } from '@current-client-bindings/app'
import Checkbox from '../forms/Checkbox'
import DiffStat from './DiffStat'
import PathText from './PathText'
import StatusBadge from './StatusBadge'
import ThreadCount from './ThreadCount'
import './ChangedFileRow.scss'

interface ChangedFileRowProps {
  file: ChangedFileInfo
  selected: boolean
  onSelect: () => void
  onContextMenu?: (e: MouseEvent | KeyboardEvent) => void
  // Unresolved review threads on the file.
  threads?: number
  // A folder the list already names.
  relativeTo?: string
  // With onRead, whether the file is ticked as read.
  read?: boolean
  onRead?: (read: boolean) => void
  // Why it's marked as changed again.
  again?: string
}

function ChangedFileRow({
  file,
  selected,
  onSelect,
  onContextMenu,
  threads,
  relativeTo,
  read = false,
  onRead,
  again,
}: ChangedFileRowProps) {
  const path = relativeTo && file.path.startsWith(`${relativeTo}/`) ? file.path.slice(relativeTo.length + 1) : file.path
  return (
    <li
      className={onRead && clsx('read-through-row changed-file-tickable', selected && 'changed-file-tickable-selected')}
    >
      {onRead && <Checkbox checked={read} onChange={onRead} ariaLabel={`Read ${file.path}`} />}
      <button
        type="button"
        className={clsx('changed-file', read && 'changed-file-read', selected && !onRead && 'changed-file-selected')}
        aria-current={onRead && selected ? 'true' : undefined}
        onClick={onSelect}
        onContextMenu={onContextMenu}
        onKeyDown={(e) => onContextMenu && isMenuKey(e) && onContextMenu(e)}
      >
        <PathText path={file.origPath ? `${file.origPath} → ${path}` : path} className="changed-file-path" />
        {again && (
          <span className="changed-file-again" title={again}>
            again
          </span>
        )}
        <ThreadCount count={threads} />
        <DiffStat added={file.added} removed={file.removed} binary={file.binary} />
        <StatusBadge status={file.status} />
      </button>
    </li>
  )
}

export default ChangedFileRow
