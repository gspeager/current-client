import { useState } from 'react'
import { ArrowDown, ArrowUp, X } from 'lucide-react'
import { useBranchStatus } from '../../lib/useBranchStatus'
import './ContextualNudges.scss'

interface ContextualNudgesProps {
  repoPath: string
  repoVersion: number
  onPush: () => void
  onPull: () => void
}

// Dismissal is keyed to the exact count, so a new commit or fetch re-shows the nudge.
function ContextualNudges({ repoPath, repoVersion, onPush, onPull }: ContextualNudgesProps) {
  const status = useBranchStatus(repoPath, repoVersion)
  const [dismissedAhead, setDismissedAhead] = useState<number | null>(null)
  const [dismissedBehind, setDismissedBehind] = useState<number | null>(null)

  if (!status) return null

  const showAhead = status.ahead > 0 && status.ahead !== dismissedAhead
  const showBehind = status.behind > 0 && status.behind !== dismissedBehind
  if (!showAhead && !showBehind) return null

  return (
    <div className="contextual-nudges">
      {showAhead && (
        <div className="contextual-nudge">
          <ArrowUp size={14} strokeWidth={1.75} className="contextual-nudge-icon" />
          <span className="contextual-nudge-text">
            {status.ahead} unpushed commit{status.ahead === 1 ? '' : 's'} — push now?
          </span>
          <button type="button" className="contextual-nudge-action" onClick={onPush}>
            Push
          </button>
          <button
            type="button"
            className="contextual-nudge-dismiss"
            onClick={() => setDismissedAhead(status.ahead)}
            aria-label="Dismiss"
          >
            <X size={12} strokeWidth={2} />
          </button>
        </div>
      )}
      {showBehind && (
        <div className="contextual-nudge">
          <ArrowDown size={14} strokeWidth={1.75} className="contextual-nudge-icon" />
          <span className="contextual-nudge-text">
            Behind {status.upstream || 'upstream'} by {status.behind} commit{status.behind === 1 ? '' : 's'} — pull now?
          </span>
          <button type="button" className="contextual-nudge-action" onClick={onPull}>
            Pull
          </button>
          <button
            type="button"
            className="contextual-nudge-dismiss"
            onClick={() => setDismissedBehind(status.behind)}
            aria-label="Dismiss"
          >
            <X size={12} strokeWidth={2} />
          </button>
        </div>
      )}
    </div>
  )
}

export default ContextualNudges
