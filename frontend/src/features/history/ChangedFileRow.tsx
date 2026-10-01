import type { MouseEvent } from 'react'
import type { ChangedFile as ChangedFileInfo } from '@current-client-bindings/app'
import DiffStat from '../../components/git/DiffStat'
import PathText from '../../components/git/PathText'
import StatusBadge from '../../components/git/StatusBadge'
import './ChangedFileRow.scss'

interface ChangedFileRowProps {
  file: ChangedFileInfo
  selected: boolean
  onSelect: () => void
  onContextMenu?: (e: MouseEvent) => void
}

function ChangedFileRow({ file, selected, onSelect, onContextMenu }: ChangedFileRowProps) {
  return (
    <li>
      <button
        type="button"
        className={selected ? 'changed-file changed-file-selected' : 'changed-file'}
        onClick={onSelect}
        onContextMenu={onContextMenu}
      >
        <PathText path={file.origPath ? `${file.origPath} → ${file.path}` : file.path} className="changed-file-path" />
        <DiffStat added={file.added} removed={file.removed} binary={file.binary} />
        <StatusBadge status={file.status} />
      </button>
    </li>
  )
}

export default ChangedFileRow
