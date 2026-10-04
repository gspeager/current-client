import { DiffService, StashService, type ChangedFile, type StashInfo } from '@current-client-bindings/app'
import ChangedFilesBrowser from '../../components/git/ChangedFilesBrowser'
import Modal from '../../components/chrome/Modal'
import { EMPTY_TREE_SHA } from '../diff/diffMapping'
import { useAsyncData } from '../../lib/useAsyncData'
import './StashModal.scss'

interface StashModalProps {
  repoPath: string
  stash: StashInfo
  onClose: () => void
}

function StashModal({ repoPath, stash, onClose }: StashModalProps) {
  const ref = `stash@{${stash.index}}`
  const { data: files, error } = useAsyncData(
    () => StashService.GetChangedFiles(repoPath, stash.index),
    [repoPath, stash.index],
  )

  // Untracked files (status ?) live in the stash's parentless third parent.
  const loadDiff = (file: ChangedFile) =>
    file.status === '?'
      ? DiffService.GetRefDiff(repoPath, file.path, EMPTY_TREE_SHA, `${ref}^3`, false)
      : DiffService.GetRefDiff(repoPath, file.path, `${ref}^1`, ref, false)

  return (
    <Modal
      title="Stash"
      onClose={onClose}
      className="stash-modal"
      headerContent={<span className="stash-modal-message">{stash.message}</span>}
    >
      <ChangedFilesBrowser files={files} error={error} emptyHint="This stash has no changes." loadDiff={loadDiff} />
    </Modal>
  )
}

export default StashModal
