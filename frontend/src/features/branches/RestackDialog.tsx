import { StackService } from '@current-client-bindings/app'
import Modal from '../../components/chrome/Modal'
import { plural } from '../../lib/plural'
import { useAsyncData } from '../../lib/useAsyncData'
import { PreviewCommitRow } from './RebasePreviewDialog'
import DialogError from '../../components/chrome/DialogError'
import './RebasePreviewDialog.scss'

interface RestackDialogProps {
  repoPath: string
  root: string
  onRestack: () => void
  onClose: () => void
}

function RestackDialog({ repoPath, root, onRestack, onClose }: RestackDialogProps) {
  const { data: preview, error } = useAsyncData(() => StackService.Preview(repoPath, root), [repoPath, root])
  const layers = preview?.layers ?? []
  const conflict = layers.some((layer) => layer.commits.some((commit) => commit.outcome === 'conflict'))
  const changes = layers.some((layer) => layer.moves || layer.merged || layer.missing)
  const moves = layers.some((layer) => layer.moves)

  return (
    <Modal title={`Restack ${root}`} onClose={onClose} className="dialog-panel rebase-preview-panel">
      <div className="dialog-body">
        <DialogError>{error}</DialogError>
        {!preview && !error && <p className="dialog-message">Working it out…</p>}
        {preview && !changes && <p className="dialog-message">Nothing to move. Every layer is on its parent.</p>}
        {preview &&
          layers.map((layer) => (
            <section key={layer.branch} className="restack-layer">
              <h3 className="restack-layer-title">
                <span className="restack-layer-branch">{layer.branch}</span>
                {layer.moves && (
                  <span>
                    {plural(layer.commits.length, 'commit')} onto {layer.onto}
                  </span>
                )}
                {layer.merged && <span>Merged into {layer.onto}. Leaves the stack.</span>}
                {layer.missing && <span>Gone. Leaves the stack.</span>}
                {!layer.moves && !layer.merged && !layer.missing && <span>Already on {layer.onto}</span>}
              </h3>
              {layer.commits.length > 0 && (
                <ol className="rebase-preview-commits">
                  {layer.commits.map((commit) => (
                    <PreviewCommitRow key={commit.sha} commit={commit} />
                  ))}
                </ol>
              )}
              {layer.droppedMerges > 0 && (
                <p className="dialog-note">{plural(layer.droppedMerges, 'merge commit')} won't be kept.</p>
              )}
            </section>
          ))}
        {conflict && (
          <p className="dialog-note">
            Stops on the conflict, with Continue and Abort in the banner. Restack again afterwards for the layers above
            it.
          </p>
        )}
        {preview && !preview.predicted && (
          <p className="dialog-note">Conflicts can't be predicted on this Git. Predicting needs Git 2.40 or newer.</p>
        )}
        <DialogError>{preview?.blocked}</DialogError>
        {preview?.dirty && moves && <DialogError>Commit or stash the changes in the working tree first.</DialogError>}
        <div className="dialog-actions">
          <button type="button" className="dialog-button" onClick={onClose}>
            Cancel
          </button>
          <button
            type="button"
            className="dialog-button dialog-button-primary"
            disabled={!changes || !!preview?.blocked || (preview?.dirty && moves)}
            onClick={onRestack}
          >
            Restack
          </button>
        </div>
      </div>
    </Modal>
  )
}

export default RestackDialog
