import { useState } from 'react'
import type { CommitInfo } from '@current-client-bindings/app'
import Modal from '../../components/chrome/Modal'
import { useBranchStatus } from '../../lib/useBranchStatus'
import './ResetDialog.scss'

export type ResetMode = 'soft' | 'mixed' | 'hard'

interface ResetDialogProps {
  repoPath: string
  commit: CommitInfo
  onClose: () => void
  onReset: (mode: ResetMode) => void
}

const MODES: { mode: ResetMode; label: string; effect: string }[] = [
  { mode: 'soft', label: 'Soft', effect: 'The later commits leave the branch; their changes stay staged.' },
  {
    mode: 'mixed',
    label: 'Mixed',
    effect: 'The later commits leave the branch; their changes stay in the working tree, unstaged.',
  },
  {
    mode: 'hard',
    label: 'Hard',
    effect: 'The later commits leave the branch and their changes are discarded, along with any uncommitted work.',
  },
]

// Moves the checked-out branch to a commit, one of three ways; the Undo dialog's
// shape: what happens, then the git it is the same as.
function ResetDialog({ repoPath, commit, onClose, onReset }: ResetDialogProps) {
  const [mode, setMode] = useState<ResetMode>('mixed')
  const status = useBranchStatus(repoPath, 0)
  const sha = commit.sha.slice(0, 7)

  return (
    <Modal title={`Reset to ${sha}`} onClose={onClose} className="dialog-panel reset-dialog">
      <div className="dialog-body">
        <p className="dialog-message">
          {status?.current || 'HEAD'} moves to {sha} <span className="reset-dialog-subject">{commit.subject}</span>.
        </p>
        <div className="reset-dialog-modes" role="radiogroup" aria-label="Reset mode">
          {MODES.map((m) => (
            <label
              key={m.mode}
              className={mode === m.mode ? 'reset-dialog-mode reset-dialog-mode-on' : 'reset-dialog-mode'}
            >
              <input
                type="radio"
                name="reset-mode"
                value={m.mode}
                checked={mode === m.mode}
                onChange={() => setMode(m.mode)}
              />
              <span className="reset-dialog-mode-text">
                <span className="reset-dialog-mode-label">{m.label}</span>
                <span className="reset-dialog-mode-effect">{m.effect}</span>
                <span className="reset-dialog-mode-git">
                  git reset --{m.mode} {sha}
                </span>
              </span>
            </label>
          ))}
        </div>
        <div className="dialog-actions">
          <button type="button" className="dialog-button" onClick={onClose}>
            Cancel
          </button>
          <button
            type="button"
            className={
              mode === 'hard' ? 'dialog-button dialog-button-destructive' : 'dialog-button dialog-button-primary'
            }
            onClick={() => onReset(mode)}
          >
            Reset ({mode})
          </button>
        </div>
      </div>
    </Modal>
  )
}

export default ResetDialog
