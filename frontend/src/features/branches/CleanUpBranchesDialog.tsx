import { useEffect, useState } from 'react'
import { BranchService, type MergedBranchInfo, RemoteService } from '@current-client-bindings/app'
import Modal from '../../components/chrome/Modal'
import Checkbox from '../../components/forms/Checkbox'
import { errorMessage } from '../../lib/errors'
import { withSignIn } from '../../lib/signIn'
import { useAsyncData } from '../../lib/useAsyncData'
import { useDialogs } from '../../lib/useDialogs'
import './CleanUpBranchesDialog.scss'

export interface DeletedBranch {
  name: string
  tip: string
}

interface CleanUpBranchesDialogProps {
  repoPath: string
  onDeleted: (branches: DeletedBranch[]) => void
  onClose: () => void
}

// Local branches whose work is already merged: found with Git alone, which
// sees squash and rebase merges too, or merged on the host as a change the
// cache knows. Building the list makes no request.
function CleanUpBranchesDialog({ repoPath, onDeleted, onClose }: CleanUpBranchesDialogProps) {
  const { signIn } = useDialogs()
  const { data: report, error: loadError } = useAsyncData(() => BranchService.MergedBranches(repoPath), [repoPath])
  const [selected, setSelected] = useState<Set<string>>(new Set())
  const [running, setRunning] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const branches = report?.branches ?? []

  useEffect(() => {
    if (report) setSelected(new Set(report.branches.filter((b) => !b.newer).map((b) => b.name)))
  }, [report])

  const toggle = (name: string, checked: boolean) =>
    setSelected((prev) => {
      const next = new Set(prev)
      if (checked) next.add(name)
      else next.delete(name)
      return next
    })

  const current = branches.find((b) => b.current && selected.has(b.name))

  // The current branch can't be deleted while it is checked out, so the
  // default branch is switched to and brought up to date first.
  const run = async () => {
    if (!report) return
    setRunning(true)
    setError(null)
    const deleted: DeletedBranch[] = []
    try {
      if (current) {
        await BranchService.CheckoutBranch(repoPath, report.default)
        await withSignIn((auth) => RemoteService.Pull(repoPath, auth), signIn)
      }
      for (const b of branches.filter((b) => selected.has(b.name))) {
        deleted.push({ name: b.name, tip: await BranchService.DeleteBranch(repoPath, b.name, true) })
      }
    } catch (err: unknown) {
      setError(errorMessage(err))
      setRunning(false)
      if (deleted.length > 0) onDeleted(deleted)
      return
    }
    onDeleted(deleted)
  }

  const count = selected.size
  return (
    <Modal title="Clean up merged branches" onClose={onClose} className="dialog-panel clean-up-panel">
      <div className="dialog-body">
        {loadError && (
          <p className="dialog-error" role="alert">
            {loadError}
          </p>
        )}
        {report && branches.length === 0 && <p className="dialog-message">No merged branches to clean up.</p>}
        {branches.length > 0 && (
          <ul className="clean-up-list">
            {branches.map((b) => (
              <li key={b.name} className="clean-up-row">
                <Checkbox
                  checked={selected.has(b.name)}
                  onChange={(checked) => toggle(b.name, checked)}
                  ariaLabel={b.name}
                />
                <span className="clean-up-name">{b.name}</span>
                <span className="clean-up-reason">{reason(b, report?.base ?? '')}</span>
                {b.newer && <span className="clean-up-newer">Newer commits</span>}
              </li>
            ))}
          </ul>
        )}
        {current && (
          <p className="dialog-note">
            Switches to {report?.default} and pulls first, since {current.name} is checked out.
          </p>
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
          <button
            type="button"
            className="dialog-button dialog-button-destructive"
            disabled={count === 0 || running}
            onClick={() => void run()}
          >
            {running ? 'Deleting…' : count > 1 ? `Delete ${count} branches` : 'Delete branch'}
          </button>
        </div>
      </div>
    </Modal>
  )
}

function reason(b: MergedBranchInfo, base: string): string {
  return b.number ? `${b.prefix}${b.number} merged on the host` : `Merged into ${base}`
}

export default CleanUpBranchesDialog
