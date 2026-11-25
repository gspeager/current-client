import type { KeyboardEvent, MouseEvent } from 'react'
import { isMenuKey } from '../../components/controls/ContextMenu'
import type { ChangedFile as ChangedFileInfo } from '@current-client-bindings/app'
import Checkbox from '../../components/forms/Checkbox'
import DiffStat from '../../components/git/DiffStat'
import PathText from '../../components/git/PathText'
import StatusBadge from '../../components/git/StatusBadge'
import ThreadCount from '../../components/git/ThreadCount'
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
}: ChangedFileRowProps) {
  const path = relativeTo && file.path.startsWith(`${relativeTo}/`) ? file.path.slice(relativeTo.length + 1) : file.path
  return (
    <li className={onRead ? 'read-through-row changed-file-tickable' : undefined}>
      {onRead && <Checkbox checked={read} onChange={onRead} ariaLabel={`Read ${file.path}`} />}
      <button
        type="button"
        className={`changed-file${selected ? ' changed-file-selected' : ''}${read ? ' changed-file-read' : ''}`}
        onClick={onSelect}
        onContextMenu={onContextMenu}
        onKeyDown={(e) => onContextMenu && isMenuKey(e) && onContextMenu(e)}
      >
        <PathText path={file.origPath ? `${file.origPath} → ${path}` : path} className="changed-file-path" />
        <ThreadCount count={threads} />
        <DiffStat added={file.added} removed={file.removed} binary={file.binary} />
        <StatusBadge status={file.status} />
      </button>
    </li>
  )
}

export default ChangedFileRow
