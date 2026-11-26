import clsx from 'clsx'
import { hostUserName } from '../../lib/useNameIdentities'
import IdentityBadge from '../../components/git/IdentityBadge'
import { type IdentityInfo, BranchChange } from '@current-client-bindings/app'
import type { Change } from '@current-client-bindings/core/hostapi'
import ChangeMarker from '../../components/git/ChangeMarker'
import ReviewSummary from '../../components/git/ReviewSummary'
import { relativeTime } from '../../lib/relativeTime'
import './ChangeRow.scss'

interface ChangeRowProps {
  identity?: IdentityInfo
  change: Change
  prefix: string
  selected: boolean
  onSelect: () => void
  // The signed-in username, so a request for review reads as yours or someone else's.
  me: string
  displayName: (name: string) => string
}

function ChangeRow({ change, prefix, selected, onSelect, me, identity, displayName }: ChangeRowProps) {
  const author = hostUserName(change.author)
  return (
    <button
      type="button"
      className={clsx('host-change-row', selected && 'host-change-row-selected')}
      onClick={onSelect}
    >
      <span className="host-change-row-head">
        <ChangeMarker
          change={
            new BranchChange({ number: change.number, prefix, state: change.state, draft: change.draft, ci: change.ci })
          }
        />
        <span className="host-change-row-meta">
          <ReviewSummary reviews={change.reviews} open={change.state === 'open'} me={me} displayName={displayName} />
          <span className="host-change-row-time">{relativeTime(new Date(change.updatedAt))}</span>
        </span>
      </span>
      <span className="host-change-row-title">
        {identity && <IdentityBadge identity={identity} name={author} email="" size={20} />}
        <span className="host-change-row-title-text">{change.title}</span>
      </span>
      <span className="host-change-row-branches">
        {change.sourceProject ? `${change.sourceProject}:` : ''}
        {change.sourceBranch} → {change.targetBranch}
      </span>
    </button>
  )
}

export default ChangeRow
