import type { MouseEvent } from 'react'
import { PlatformService, type BranchChange } from '@current-client-bindings/app'
import { ignoreDecorativeFailure } from '../../lib/errors'
import { relativeTime } from '../../lib/relativeTime'
import HoverCard from '../chrome/HoverCard'
import { CIIcon, ciLabel } from './CIMark'
import './ChangeMarker.scss'

interface ChangeMarkerProps {
  change: BranchChange | undefined
}

const STATE_LABELS: Record<string, string> = { open: 'open', merged: 'merged', closed: 'closed' }

// The square marker for a branch's merge request / pull request: number,
// state and CI result when the host is on, the number alone when only Git
// supplied it. Read from the cache; nothing here makes a request.
function ChangeMarker({ change }: ChangeMarkerProps) {
  if (!change) return null
  const number = `${change.prefix}${change.number}`
  const state = change.draft ? 'draft' : (STATE_LABELS[change.state] ?? '')
  const title = [number, change.title, state, change.ci && ciLabel(change.ci)].filter(Boolean).join(' · ')
  const open = (e: MouseEvent) => {
    e.stopPropagation()
    if (change.webUrl) PlatformService.OpenWebPage(change.webUrl).catch(ignoreDecorativeFailure)
  }
  return (
    <HoverCard card={<ChangeCard change={change} number={number} state={state} />}>
      <button
        type="button"
        className={change.webUrl ? 'change-marker' : 'change-marker change-marker-static'}
        data-state={state || 'unknown'}
        aria-label={title}
        onClick={open}
      >
        <span className="change-marker-square" />
        <span className="change-marker-number">{number}</span>
        {change.ci && <CIIcon status={change.ci} size={12} />}
      </button>
    </HoverCard>
  )
}

// Everything the cache holds for the change; with the host off that is the
// number alone, and the card says so.
function ChangeCard({ change, number, state }: { change: BranchChange; number: string; state: string }) {
  if (!change.state) {
    return (
      <>
        <p className="hover-card-title">{number}</p>
        <p className="hover-card-line hover-card-muted">
          Found through Git. Turn integration on for its title and state.
        </p>
      </>
    )
  }
  return (
    <>
      <p className="hover-card-title">{change.title}</p>
      <p className="hover-card-line">
        <span className="hover-card-data">{number}</span>
        <span>{state}</span>
        {change.author && <span>by {change.author}</span>}
      </p>
      {change.sourceBranch && (
        <p className="hover-card-line hover-card-data">
          {change.sourceBranch} → {change.targetBranch}
        </p>
      )}
      <p className="hover-card-line">
        {change.ci && <span>{ciLabel(change.ci)}</span>}
        {change.unresolvedThreads > 0 && <span>{change.unresolvedThreads} unresolved</span>}
        {change.updatedAt && (
          <span className="hover-card-muted">updated {relativeTime(new Date(change.updatedAt))}</span>
        )}
      </p>
    </>
  )
}

export default ChangeMarker
