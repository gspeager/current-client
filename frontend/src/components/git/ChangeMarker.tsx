import type { MouseEvent } from 'react'
import { PlatformService, type BranchChange } from '@current-client-bindings/app'
import { ignoreDecorativeFailure } from '../../lib/errors'
import { relativeTime } from '../../lib/relativeTime'
import HoverCard from '../chrome/HoverCard'
import './ChangeMarker.scss'

interface ChangeMarkerProps {
  change: BranchChange | undefined
}

const STATE_LABELS: Record<string, string> = { open: 'open', merged: 'merged', closed: 'closed' }
const CI_LABELS: Record<string, string> = {
  success: 'CI passed',
  failed: 'CI failed',
  running: 'CI running',
  pending: 'CI pending',
  canceled: 'CI canceled',
  skipped: 'CI skipped',
}

// The square marker for a branch's merge request / pull request: number,
// state and CI result when the host is on, the number alone when only Git
// supplied it. Read from the cache; nothing here makes a request.
function ChangeMarker({ change }: ChangeMarkerProps) {
  if (!change) return null
  const number = `${change.prefix}${change.number}`
  const state = change.draft ? 'draft' : (STATE_LABELS[change.state] ?? '')
  const title = [number, change.title, state, CI_LABELS[change.ci]].filter(Boolean).join(' · ')
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
        {change.ci && <span className={`change-marker-ci change-marker-ci-${change.ci}`} />}
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
        {change.ci && <span>{CI_LABELS[change.ci] ?? change.ci}</span>}
        {change.unresolvedThreads > 0 && <span>{change.unresolvedThreads} unresolved</span>}
        {change.updatedAt && (
          <span className="hover-card-muted">updated {relativeTime(new Date(change.updatedAt))}</span>
        )}
      </p>
    </>
  )
}

export default ChangeMarker
