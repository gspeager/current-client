import { DiffService, type ChangedFile, type CommitInfo } from '@current-client-bindings/app'
import Modal from '../../components/chrome/Modal'
import ChangedFilesBrowser from '../../components/git/ChangedFilesBrowser'
import { diffBaseFor } from '../diff/diffMapping'
import './CommitFilesModal.scss'

interface CommitFilesModalProps {
  repoPath: string
  commit: CommitInfo
  files: ChangedFile[]
  initialPath: string
  onClose: () => void
}

// A commit's changes with room to read them; the Commit detail pane is too narrow.
function CommitFilesModal({ repoPath, commit, files, initialPath, onClose }: CommitFilesModalProps) {
  const base = diffBaseFor(commit.parentShas)
  return (
    <Modal
      title={`Commit ${commit.sha.slice(0, 7)}`}
      onClose={onClose}
      className="commit-files-panel"
      headerContent={<span className="commit-files-subject">{commit.subject}</span>}
    >
      <ChangedFilesBrowser
        files={files}
        error={null}
        emptyHint="No file changes."
        initialPath={initialPath}
        repoPath={repoPath}
        loadDiff={(file) => DiffService.GetRefDiff(repoPath, file.path, base, commit.sha, false)}
        images={() => ({
          repoPath,
          before: { kind: 'commit', rev: base },
          after: { kind: 'commit', rev: commit.sha },
        })}
      />
    </Modal>
  )
}

export default CommitFilesModal
