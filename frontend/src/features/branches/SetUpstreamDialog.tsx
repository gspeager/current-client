import { useState, type FormEvent } from 'react'
import { BranchService } from '@current-client-bindings/app'
import Modal from '../../components/chrome/Modal'
import RefSelect from '../../components/git/RefSelect'
import { errorMessage } from '../../lib/errors'
import { useAsyncData } from '../../lib/useAsyncData'

interface SetUpstreamDialogProps {
  repoPath: string
  branch: string
  upstream: string
  onClose: () => void
  onSet: () => void
}

function SetUpstreamDialog({ repoPath, branch, upstream, onClose, onSet }: SetUpstreamDialogProps) {
  const { data: remoteBranches } = useAsyncData(() => BranchService.ListRemote(repoPath), [repoPath])
  const [choice, setChoice] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(null)
  const selected = choice ?? (upstream || remoteBranches?.[0] || '')

  const submit = (e: FormEvent) => {
    e.preventDefault()
    setError(null)
    BranchService.SetUpstream(repoPath, branch, selected)
      .then(onSet)
      .catch((err: unknown) => setError(errorMessage(err)))
  }

  return (
    <Modal title="Set upstream" onClose={onClose} className="dialog-panel">
      <form className="dialog-body" onSubmit={submit}>
        {remoteBranches?.length === 0 ? (
          <p className="dialog-message">No remote branches. Fetch or push first.</p>
        ) : (
          <label className="dialog-field">
            <span className="dialog-message">Remote branch for {branch} to track</span>
            <RefSelect
              groups={[{ label: 'Remote branches', names: remoteBranches ?? [] }]}
              value={selected}
              onChange={setChoice}
              label="Upstream"
            />
          </label>
        )}
        {error && <p className="dialog-message dialog-error">{error}</p>}
        <div className="dialog-actions">
          <button type="button" className="dialog-button" onClick={onClose}>
            Cancel
          </button>
          <button
            type="submit"
            className="dialog-button dialog-button-primary"
            disabled={!selected || selected === upstream}
          >
            Set upstream
          </button>
        </div>
      </form>
    </Modal>
  )
}

export default SetUpstreamDialog
