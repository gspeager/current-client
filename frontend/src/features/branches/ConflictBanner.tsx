import { useState } from 'react'
import { TriangleAlert } from 'lucide-react'
import { ConflictService, type ConflictStateInfo } from '@current-client-bindings/app'
import { errorMessage } from '../../lib/errors'
import { baseName } from '../../lib/paths'
import { useDialogs } from '../../lib/useDialogs'
import './ConflictBanner.scss'

interface ConflictBannerProps {
  repoPath: string
  state: ConflictStateInfo
  onResolved: () => void
  onOpenFile: (path: string) => void
}

const OPERATION_LABEL: Record<string, string> = {
  merge: 'Merge',
  'cherry-pick': 'Cherry-pick',
  revert: 'Revert',
  rebase: 'Rebase',
  am: 'Apply patch',
}

function ConflictBanner({ repoPath, state, onResolved, onOpenFile }: ConflictBannerProps) {
  const { confirm } = useDialogs()
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const label = OPERATION_LABEL[state.operation] ?? state.operation
  const remaining = state.conflictedPaths.length

  const continueOp = () => {
    setError(null)
    setBusy(true)
    ConflictService.Continue(repoPath)
      .then(onResolved)
      .catch((err: unknown) => setError(errorMessage(err)))
      .finally(() => setBusy(false))
  }

  const abort = async () => {
    const confirmed = await confirm({
      title: `Abort ${label.toLowerCase()}`,
      message: `Abort the ${label.toLowerCase()} and return to the state before it started?`,
      confirmLabel: 'Abort',
      destructive: true,
    })
    if (!confirmed) return
    setError(null)
    setBusy(true)
    ConflictService.Abort(repoPath)
      .then(onResolved)
      .catch((err: unknown) => setError(errorMessage(err)))
      .finally(() => setBusy(false))
  }

  return (
    <div className="conflict-banner">
      <TriangleAlert size={14} strokeWidth={1.75} className="conflict-banner-icon" />
      <span className="conflict-banner-text">
        {label} in progress
        {remaining > 0 ? ` — ${remaining} file${remaining === 1 ? '' : 's'} conflicted` : ' — all conflicts resolved'}
        {state.conflictedPaths.map((path) => (
          <button
            key={path}
            type="button"
            className="conflict-banner-file"
            onClick={() => onOpenFile(path)}
            title={path}
          >
            {baseName(path)}
          </button>
        ))}
      </span>
      <button
        type="button"
        className="conflict-banner-continue"
        onClick={continueOp}
        disabled={busy || remaining > 0}
        title={remaining > 0 ? 'Resolve all conflicted files first' : 'Continue'}
      >
        Continue
      </button>
      <button type="button" className="conflict-banner-abort" onClick={abort} disabled={busy}>
        Abort
      </button>
      {error && <span className="conflict-banner-error">{error}</span>}
    </div>
  )
}

export default ConflictBanner
