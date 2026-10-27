import { useEffect, useState, type FormEvent } from 'react'
import { HostService, type MergeResult, type ReadinessFact } from '@current-client-bindings/app'
import type { Change } from '@current-client-bindings/core/hostapi'
import Modal from '../../components/chrome/Modal'
import Switch from '../../components/forms/Switch'
import { errorMessage } from '../../lib/errors'
import { useAsyncData } from '../../lib/useAsyncData'
import './MergeChangeDialog.scss'

interface MergeChangeDialogProps {
  repoPath: string
  change: Change
  prefix: string
  host: string
  facts: ReadinessFact[]
  onMerged: (result: MergeResult) => void
  onClose: () => void
}

const METHOD_LABELS: Record<string, string> = {
  merge: 'Merge commit',
  squash: 'Squash and merge',
  rebase: 'Rebase and merge',
}

// Merging on the host: one request for the host's merge settings when the
// dialog opens, the readiness facts from the last refresh, and the merge
// itself on the button. A refused merge shows the host's reason.
function MergeChangeDialog({ repoPath, change, prefix, host, facts, onMerged, onClose }: MergeChangeDialogProps) {
  const { data: options, error: optionsError } = useAsyncData(
    () => HostService.MergeOptions(repoPath, change.number),
    [repoPath, change.number],
  )
  const [method, setMethod] = useState<string | null>(null)
  const [deleteBranch, setDeleteBranch] = useState<boolean | null>(null)
  const [running, setRunning] = useState(false)
  const [error, setError] = useState<string | null>(null)
  useEffect(() => {
    if (options) {
      setMethod((m) => m ?? options.methods[0] ?? 'merge')
      setDeleteBranch((d) => d ?? options.deleteBranch)
    }
  }, [options])
  const title = `Merge ${prefix}${change.number}`
  const chosen = method ?? options?.methods[0] ?? 'merge'
  const warnings = facts.filter((f) => f.level === 'warn' || f.level === 'unknown')

  const submit = (e: FormEvent) => {
    e.preventDefault()
    if (!options || running) return
    setRunning(true)
    setError(null)
    HostService.MergeChange(repoPath, change.number, chosen, deleteBranch ?? options.deleteBranch)
      .then(onMerged)
      .catch((err: unknown) => {
        setError(errorMessage(err))
        setRunning(false)
      })
  }

  return (
    <Modal title={title} onClose={onClose} className="dialog-panel merge-change-panel">
      <form className="dialog-body" onSubmit={submit}>
        <p className="dialog-message">
          <span className="dialog-result">{change.sourceBranch}</span> into{' '}
          <span className="dialog-result">{change.targetBranch}</span> on {host}
        </p>
        {warnings.length > 0 && (
          <ul className="merge-change-warnings" aria-label="Before merging">
            {warnings.map((f) => (
              <li key={f.label} className={`merge-change-warning-${f.level}`}>
                {f.label}: {f.value}
              </li>
            ))}
          </ul>
        )}
        {optionsError && (
          <p className="dialog-error" role="alert">
            {optionsError}
          </p>
        )}
        {!options && !optionsError && <p className="dialog-note">Asking {host} how it merges…</p>}
        {options && (
          <>
            {options.methods.length > 1 && (
              <fieldset className="merge-change-methods">
                <legend className="dialog-message">Method</legend>
                {options.methods.map((m) => (
                  <label key={m} className="merge-change-method">
                    <input
                      type="radio"
                      name="merge-method"
                      value={m}
                      checked={chosen === m}
                      onChange={() => setMethod(m)}
                    />
                    {METHOD_LABELS[m] ?? m}
                  </label>
                ))}
              </fieldset>
            )}
            {options.deleteBranchFixed ? (
              <p className="dialog-note">
                {options.deleteBranch
                  ? `${host} deletes ${change.sourceBranch} after merging.`
                  : `${host} keeps ${change.sourceBranch} after merging.`}
              </p>
            ) : (
              <Switch
                checked={deleteBranch ?? options.deleteBranch}
                onChange={setDeleteBranch}
                label={`Delete ${change.sourceBranch} on ${host} after merging`}
              />
            )}
          </>
        )}
        {error && (
          <p className="dialog-error" role="alert">
            {error}
          </p>
        )}
        <div className="dialog-actions">
          <button type="button" className="dialog-button" onClick={onClose}>
            Cancel
          </button>
          <button type="submit" className="dialog-button dialog-button-primary" disabled={!options || running}>
            {running ? 'Merging…' : (METHOD_LABELS[chosen] ?? 'Merge')}
          </button>
        </div>
      </form>
    </Modal>
  )
}

export default MergeChangeDialog
