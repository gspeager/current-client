import { useEffect, useRef } from 'react'
import type { useRepositoryLifecycle } from './useRepositoryLifecycle'
import Modal from '../../components/chrome/Modal'
import RepoSwitcher from './RepoSwitcher'
import './RepoPickerModal.scss'

interface RepoPickerModalProps {
  repo: ReturnType<typeof useRepositoryLifecycle>
  onClose: () => void
}

function RepoPickerModal({ repo, onClose }: RepoPickerModalProps) {
  const openedWith = useRef(repo.repoPath)

  useEffect(() => {
    if (repo.repoPath !== openedWith.current) {
      onClose()
    }
  }, [repo.repoPath, onClose])

  return (
    <Modal title="Open repository" onClose={onClose} className="repo-picker-panel" placement="top">
      <div className="repo-picker-body">
        <RepoSwitcher repo={repo} />
      </div>
    </Modal>
  )
}

export default RepoPickerModal
