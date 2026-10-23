import type { ReactNode } from 'react'
import Modal from './Modal'
import './DiffSheet.scss'

interface DiffSheetProps {
  title: string
  headerContent?: ReactNode
  onClose: () => void
  children: ReactNode
}

// The one shape for browsing a set of changes: full height, a file list and a diff.
function DiffSheet({ title, headerContent, onClose, children }: DiffSheetProps) {
  return (
    <Modal title={title} onClose={onClose} className="diff-sheet" headerContent={headerContent}>
      {children}
    </Modal>
  )
}

export default DiffSheet
