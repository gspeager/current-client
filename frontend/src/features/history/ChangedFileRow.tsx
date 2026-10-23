import type { KeyboardEvent, MouseEvent } from 'react'
import { isMenuKey } from '../../components/controls/ContextMenu'
import type { ChangedFile as ChangedFileInfo } from '@current-client-bindings/app'
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
}

function ChangedFileRow({ file, selected, onSelect, onContextMenu, threads }: ChangedFileRowProps) {
  return (
    <li>
      <button
        type="button"
        className={selected ? 'changed-file changed-file-selected' : 'changed-file'}
        onClick={onSelect}
        onContextMenu={onContextMenu}
        onKeyDown={(e) => onContextMenu && isMenuKey(e) && onContextMenu(e)}
      >
        <PathText path={file.origPath ? `${file.origPath} → ${file.path}` : file.path} className="changed-file-path" />
        <ThreadCount count={threads} />
        <DiffStat added={file.added} removed={file.removed} binary={file.binary} />
        <StatusBadge status={file.status} />
      </button>
    </li>
  )
}

export default ChangedFileRow
