import { useState } from 'react'
import { GitBranchPlus } from 'lucide-react'
import { BranchService, ReflogService, type ReflogEntryInfo } from '@current-client-bindings/app'
import { toBranchName } from '../../lib/branchName'
import { errorMessage } from '../../lib/errors'
import { relativeTime } from '../../lib/relativeTime'
import { useClockTick } from '../../lib/useClockTick'
import { useAsyncData } from '../../lib/useAsyncData'
import { useDialogs } from '../../lib/useDialogs'
import Modal from '../../components/chrome/Modal'
import './ReflogPanel.scss'

interface ReflogPanelProps {
  repoPath: string
  onClose: () => void
  onBranchChanged?: () => void
}

const REFLOG_LIMIT = 200

function ReflogPanel({ repoPath, onClose, onBranchChanged }: ReflogPanelProps) {
  useClockTick()
  const { data: entries, error: loadError } = useAsyncData(
    () => ReflogService.GetReflog(repoPath, REFLOG_LIMIT),
    [repoPath],
  )
  const { prompt } = useDialogs()
  const [recoverError, setRecoverError] = useState<string | null>(null)
  const [recovering, setRecovering] = useState<string | null>(null)
  const error = loadError ?? recoverError

  const recover = async (entry: ReflogEntryInfo) => {
    const name = await prompt({
      title: 'Recover commit',
      label: `Branch name for ${entry.sha.slice(0, 7)} ("${entry.subject}")`,
      confirmLabel: 'Create branch',
      transform: toBranchName,
    })
    if (!name) return
    setRecoverError(null)
    setRecovering(entry.sha)
    BranchService.CreateBranchAt(repoPath, name, entry.sha)
      .then(() => BranchService.CheckoutBranch(repoPath, name))
      .then(() => {
        onBranchChanged?.()
        onClose()
      })
      .catch((err: unknown) => setRecoverError(errorMessage(err)))
      .finally(() => setRecovering(null))
  }

  return (
    <Modal title="Reflog" onClose={onClose} className="reflog-panel">
      {error && <p className="reflog-error">{error}</p>}

      <div className="reflog-body">
        {entries === null ? (
          <p className="reflog-hint">Loading reflog…</p>
        ) : entries.length === 0 ? (
          <p className="reflog-hint">No reflog entries.</p>
        ) : (
          <ul className="reflog-list">
            {entries.map((entry) => (
              <li key={entry.selector} className="reflog-row">
                <span className="reflog-row-selector">{entry.selector}</span>
                <span className="reflog-row-sha">{entry.sha.slice(0, 7)}</span>
                <span className="reflog-row-action">{entry.action}</span>
                <span className="reflog-row-time">{relativeTime(new Date(entry.date))}</span>
                <button
                  type="button"
                  className="reflog-row-recover"
                  onClick={() => recover(entry)}
                  disabled={recovering !== null}
                  aria-label={`Recover ${entry.sha}`}
                  title="Branch from here"
                >
                  <GitBranchPlus size={14} strokeWidth={1.75} />
                  Recover
                </button>
              </li>
            ))}
          </ul>
        )}
      </div>
    </Modal>
  )
}

export default ReflogPanel
