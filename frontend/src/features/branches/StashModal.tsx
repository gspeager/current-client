import { DiffService, StashService, type ChangedFile, type StashInfo } from '@current-client-bindings/app'
import ChangedFilesBrowser from '../../components/git/ChangedFilesBrowser'
import DiffSheet from '../../components/chrome/DiffSheet'
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
  const versions = (file: ChangedFile) =>
    file.status === '?' ? { from: EMPTY_TREE_SHA, to: `${ref}^3` } : { from: `${ref}^1`, to: ref }

  const loadDiff = (file: ChangedFile, force: boolean, ignoreWhitespace: boolean) => {
    const { from, to } = versions(file)
    return DiffService.GetRefDiff(repoPath, file.path, from, to, force, ignoreWhitespace)
  }

  const images = (file: ChangedFile) => {
    const { from, to } = versions(file)
    return { repoPath, before: { kind: 'commit', rev: from }, after: { kind: 'commit', rev: to } }
  }

  return (
    <DiffSheet
      title="Stash"
      onClose={onClose}
      headerContent={<span className="stash-modal-message">{stash.message}</span>}
    >
      <ChangedFilesBrowser
        files={files}
        error={error}
        emptyHint="This stash has no changes."
        loadDiff={loadDiff}
        images={images}
      />
    </DiffSheet>
  )
}

export default StashModal
