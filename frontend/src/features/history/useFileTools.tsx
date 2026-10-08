import { useState, type KeyboardEvent, type MouseEvent } from 'react'
import ContextMenu, {
  menuAnchor,
  type ContextMenuItem,
  type ContextMenuState,
} from '../../components/controls/ContextMenu'
import BlameView from './BlameView'
import FileHistoryPanel from './FileHistoryPanel'

// The context menu, file history and blame that file lists share. pathItems
// leaves out history and blame for a file git doesn't track yet.
export function useFileTools(repoPath: string) {
  const [contextMenu, setContextMenu] = useState<ContextMenuState | null>(null)
  const [fileHistoryPath, setFileHistoryPath] = useState<string | null>(null)
  const [blamePath, setBlamePath] = useState<string | null>(null)

  const openMenu = (e: MouseEvent | KeyboardEvent, items: ContextMenuItem[]) => {
    e.preventDefault()
    setContextMenu({ ...menuAnchor(e), items })
  }

  const pathItems = (path: string, tracked = true): ContextMenuItem[] => [
    { label: 'Copy path', onClick: () => void navigator.clipboard.writeText(path) },
    ...(tracked
      ? [
          { label: 'File history', onClick: () => setFileHistoryPath(path) },
          { label: 'Blame', onClick: () => setBlamePath(path) },
        ]
      : []),
  ]

  const overlays = (
    <>
      {contextMenu && <ContextMenu state={contextMenu} onClose={() => setContextMenu(null)} />}
      {fileHistoryPath && (
        <FileHistoryPanel repoPath={repoPath} path={fileHistoryPath} onClose={() => setFileHistoryPath(null)} />
      )}
      {blamePath && <BlameView repoPath={repoPath} path={blamePath} onClose={() => setBlamePath(null)} />}
    </>
  )

  return { openMenu, pathItems, overlays }
}
