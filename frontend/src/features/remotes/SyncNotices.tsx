import type { PushResult } from '@current-client-bindings/app'
import GitOutput from '../../components/chrome/GitOutput'
import type { useCancellableOperation } from '../../lib/cancellableOperation'
import { DIVERGED } from './useRemoteSync'

type Operation = ReturnType<typeof useCancellableOperation>

interface SyncNoticesProps {
  pullOp: Operation
  pushOp: Operation
  onPullMerge: () => void
  onPullRebase: () => void
  // A push that left a branch with no change yet offers to create one.
  pushed: PushResult | null
  onCreateChange: (pushed: PushResult) => void
  onDismissPushed: () => void
}

function SyncNotices({
  pullOp,
  pushOp,
  onPullMerge,
  onPullRebase,
  pushed,
  onCreateChange,
  onDismissPushed,
}: SyncNoticesProps) {
  return (
    <>
      {pullOp.running && (
        <p className="app-inline-notice">
          Pulling… <button onClick={pullOp.cancel}>Cancel</button>
        </p>
      )}
      {pushOp.running && (
        <p className="app-inline-notice">
          Pushing… <button onClick={pushOp.cancel}>Cancel</button>
        </p>
      )}
      {pullOp.error && (
        <div className="app-inline-error">
          Could not pull: {pullOp.error}
          {pullOp.error === DIVERGED && (
            <>
              <button onClick={onPullMerge}>Pull (merge)</button>
              <button onClick={onPullRebase}>Pull (rebase)</button>
            </>
          )}
          <GitOutput message={pullOp.error} />
        </div>
      )}
      {pushOp.error && (
        <div className="app-inline-error">
          Could not push: {pushOp.error}
          <GitOutput message={pushOp.error} />
        </div>
      )}
      {pushed && (
        <p className="app-inline-notice" role="status">
          Pushed {pushed.branch}.{' '}
          <button type="button" onClick={() => onCreateChange(pushed)}>
            Create {pushed.kind}
          </button>
          <button type="button" onClick={onDismissPushed} aria-label="Dismiss">
            Dismiss
          </button>
        </p>
      )}
    </>
  )
}

export default SyncNotices
