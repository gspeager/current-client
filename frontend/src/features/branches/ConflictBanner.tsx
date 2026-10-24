import { useState } from 'react'
import { TriangleAlert } from 'lucide-react'
import { ConflictService, type ConflictStateInfo } from '@current-client-bindings/app'
import { operationNoun, operationSentence } from '../../lib/conflictWords'
import { errorMessage } from '../../lib/errors'
import { baseName } from '../../lib/paths'
import { plural } from '../../lib/plural'
import { useDialogs } from '../../lib/useDialogs'
import './ConflictBanner.scss'

interface ConflictBannerProps {
  repoPath: string
  state: ConflictStateInfo
  onResolved: () => void
  onOpenFile: (path: string) => void
}

// Status and the way out; the operation is continued from the commit composer.
function ConflictBanner({ repoPath, state, onResolved, onOpenFile }: ConflictBannerProps) {
  const { confirm } = useDialogs()
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const noun = operationNoun(state.operation)
  const remaining = state.conflictedPaths.length

  const abort = async () => {
    const confirmed = await confirm({
      title: `Abort ${noun}`,
      message: `Abort the ${noun} and return to the state before it started?`,
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
    <div className="conflict-banner" role="status">
      <TriangleAlert size={14} strokeWidth={1.75} className="conflict-banner-icon" />
      <span className="conflict-banner-text">
        <span className="conflict-banner-status">
          {operationSentence(state)}
          {remaining > 0 ? ` · ${plural(remaining, 'file')} still conflicted` : ' · all conflicts resolved'}
        </span>
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
        <span className="conflict-banner-pointer">
          {remaining > 0
            ? `Resolve each file, then continue the ${noun} from the commit message.`
            : `Continue the ${noun} from the commit message.`}
        </span>
      </span>
      <button type="button" className="conflict-banner-abort" onClick={abort} disabled={busy}>
        Abort
      </button>
      {error && <span className="conflict-banner-error">{error}</span>}
    </div>
  )
}

export default ConflictBanner
