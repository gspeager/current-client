import { useState } from 'react'
import { BranchService, RecoverService, type SnapshotInfo } from '@current-client-bindings/app'
import Modal from '../../components/chrome/Modal'
import PathText from '../../components/git/PathText'
import { toBranchName } from '../../lib/branchName'
import { errorMessage } from '../../lib/errors'
import { plural } from '../../lib/plural'
import { relativeTime } from '../../lib/relativeTime'
import { useAsyncData } from '../../lib/useAsyncData'
import { useClockTick } from '../../lib/useClockTick'
import { useDialogs } from '../../lib/useDialogs'
import { shortSha } from '../../lib/text'
import DialogError from '../../components/chrome/DialogError'
import './RecoverPanel.scss'

const ACTION_LABEL: Record<string, string> = {
  discard: 'Before discard',
  checkout: 'Before checkout',
  stash: 'Before stash',
  reset: 'Before hard reset',
  undo: 'Before undo',
  restore: 'Before restore',
  'force-push': 'Before force-push',
  'branch delete': 'Before branch delete',
}

interface RecoverPanelProps {
  repoPath: string
  onClose: () => void
  onBranchChanged?: () => void
}

function RecoverPanel({ repoPath, onClose, onBranchChanged }: RecoverPanelProps) {
  useClockTick()
  const { prompt, confirm } = useDialogs()
  const { data: snapshots, error: loadError, reload } = useAsyncData(() => RecoverService.List(repoPath), [repoPath])
  const [open, setOpen] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [done, setDone] = useState<string | null>(null)

  const act = (run: Promise<unknown>, message: string) => {
    setError(null)
    setDone(null)
    run
      .then(() => {
        setDone(message)
        reload()
        onBranchChanged?.()
      })
      .catch((err: unknown) => setError(errorMessage(err)))
  }

  const restore = async (snapshot: SnapshotInfo, file?: string) => {
    if (
      !file &&
      !(await confirm({
        title: 'Restore snapshot',
        message: `Restores ${plural(snapshot.files.length, 'file')} from this snapshot over what's in the working tree. What it overwrites is saved as a new snapshot first.`,
        confirmLabel: 'Restore',
      }))
    ) {
      return
    }
    act(RecoverService.Restore(repoPath, snapshot.id, file ? [file] : []), file ? `Restored ${file}.` : 'Restored.')
  }

  const makeBranch = async (snapshot: SnapshotInfo) => {
    const name = await prompt({
      title: 'Recover commits',
      label: `Branch name for ${shortSha(snapshot.tip)}, the tip of ${snapshot.branch}`,
      confirmLabel: 'Create branch',
      transform: toBranchName,
    })
    if (name) act(BranchService.CreateBranchAt(repoPath, name, snapshot.tip), `Created ${name}.`)
  }

  const shownError = loadError ?? error
  return (
    <Modal title="Recover" onClose={onClose} className="dialog-panel recover-panel">
      <div className="dialog-body">
        <p className="dialog-note">
          Saved before actions that lose work. Kept on this computer, and never pushed. The number kept is in Settings.
        </p>
        <DialogError>{shownError}</DialogError>
        {done && (
          <p className="dialog-message" role="status">
            {done}
          </p>
        )}
        {snapshots?.length === 0 && <p className="dialog-message">Nothing saved yet.</p>}
        {snapshots && snapshots.length > 0 && (
          <ul className="recover-list">
            {snapshots.map((snapshot) => (
              <li key={snapshot.id} className="recover-row">
                <div className="recover-row-line">
                  <span className="recover-row-action">{ACTION_LABEL[snapshot.action] ?? snapshot.action}</span>
                  {snapshot.branch && <span className="recover-row-branch">{snapshot.branch}</span>}
                  <span className="recover-row-time">{relativeTime(new Date(snapshot.date))}</span>
                  <span className="recover-row-actions">
                    {snapshot.kind === 'tip' ? (
                      <button type="button" className="dialog-button" onClick={() => void makeBranch(snapshot)}>
                        Create branch
                      </button>
                    ) : (
                      <>
                        <button
                          type="button"
                          className="dialog-button"
                          aria-expanded={open === snapshot.id}
                          onClick={() => setOpen(open === snapshot.id ? null : snapshot.id)}
                        >
                          {plural(snapshot.files.length, 'file')}
                        </button>
                        <button type="button" className="dialog-button" onClick={() => void restore(snapshot)}>
                          Restore
                        </button>
                      </>
                    )}
                  </span>
                </div>
                {snapshot.skipped > 0 && (
                  <p className="recover-row-note">{plural(snapshot.skipped, 'untracked file')} over 10 MB left out.</p>
                )}
                {open === snapshot.id && (
                  <ul className="recover-files">
                    {snapshot.files.map((file) => (
                      <li key={file} className="recover-file">
                        <PathText path={file} />
                        <button type="button" className="dialog-button" onClick={() => void restore(snapshot, file)}>
                          Restore one file
                        </button>
                      </li>
                    ))}
                  </ul>
                )}
              </li>
            ))}
          </ul>
        )}
        <div className="dialog-actions">
          <button type="button" className="dialog-button" onClick={onClose}>
            Close
          </button>
        </div>
      </div>
    </Modal>
  )
}

export default RecoverPanel
