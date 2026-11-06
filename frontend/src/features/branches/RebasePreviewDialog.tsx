import { BranchService, type RebasePreviewCommitInfo } from '@current-client-bindings/app'
import Modal from '../../components/chrome/Modal'
import { plural } from '../../lib/plural'
import { useAsyncData } from '../../lib/useAsyncData'
import './RebasePreviewDialog.scss'

const OUTCOME_LABEL: Record<string, string> = {
  conflict: 'Conflict',
  empty: 'Already applied, dropped',
  unknown: '',
  clean: '',
}

export function PreviewCommitRow({ commit }: { commit: RebasePreviewCommitInfo }) {
  return (
    <li className="rebase-preview-commit">
      <div className="rebase-preview-commit-line">
        <span className="rebase-preview-sha">{commit.sha.slice(0, 7)}</span>
        <span className="rebase-preview-subject" title={commit.subject}>
          {commit.subject}
        </span>
        {OUTCOME_LABEL[commit.outcome] && (
          <span className={`rebase-preview-outcome rebase-preview-outcome-${commit.outcome}`}>
            {OUTCOME_LABEL[commit.outcome]}
          </span>
        )}
      </div>
      {commit.files.length > 0 && <p className="rebase-preview-files">{commit.files.join(', ')}</p>}
    </li>
  )
}

interface RebasePreviewDialogProps {
  repoPath: string
  onto: string
  onRebase: () => void
  onClose: () => void
}

function RebasePreviewDialog({ repoPath, onto, onRebase, onClose }: RebasePreviewDialogProps) {
  const { data: preview, error } = useAsyncData(() => BranchService.PreviewRebase(repoPath, onto), [repoPath, onto])
  const nothingToDo = preview?.commits.length === 0
  const conflict = preview?.commits.some((commit) => commit.outcome === 'conflict')

  return (
    <Modal
      title={`Rebase ${preview?.current || 'current'} onto ${onto}`}
      onClose={onClose}
      className="dialog-panel rebase-preview-panel"
    >
      <div className="dialog-body">
        {error && (
          <p className="dialog-error" role="alert">
            {error}
          </p>
        )}
        {!preview && !error && <p className="dialog-message">Working it out…</p>}
        {preview && nothingToDo && <p className="dialog-message">Nothing to move. Already on {onto}.</p>}
        {preview && !nothingToDo && (
          <>
            <p className="dialog-message">{plural(preview.commits.length, 'commit')} to move, oldest first.</p>
            <ol className="rebase-preview-commits">
              {preview.commits.map((commit) => (
                <PreviewCommitRow key={commit.sha} commit={commit} />
              ))}
            </ol>
            {conflict && (
              <p className="dialog-note">
                Stops on the conflict, with Continue and Abort in the banner. What follows it depends on how it's
                resolved.
              </p>
            )}
            {!preview.predicted && (
              <p className="dialog-note">
                Conflicts can't be predicted on this Git. Predicting needs Git 2.40 or newer.
              </p>
            )}
            {preview.droppedMerges > 0 && (
              <p className="dialog-note">
                {plural(preview.droppedMerges, 'merge commit')} won't be kept. Their changes stay in the commits above.
              </p>
            )}
            {preview.forcePush && (
              <p className="dialog-note">
                {preview.upstream} has these commits, so pushing afterwards needs a force push with --force-with-lease.
              </p>
            )}
            {preview.dirty && (
              <p className="dialog-error" role="alert">
                Commit or stash the changes in the working tree first.
              </p>
            )}
            <p className="rebase-preview-command">{preview.command}</p>
          </>
        )}
        <div className="dialog-actions">
          <button type="button" className="dialog-button" onClick={onClose}>
            Cancel
          </button>
          <button
            type="button"
            className="dialog-button dialog-button-primary"
            disabled={!preview || nothingToDo || preview.dirty}
            onClick={onRebase}
          >
            Rebase
          </button>
        </div>
      </div>
    </Modal>
  )
}

export default RebasePreviewDialog
