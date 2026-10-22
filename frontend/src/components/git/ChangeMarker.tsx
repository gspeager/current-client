import type { MouseEvent } from 'react'
import { PlatformService, type BranchChange } from '@current-client-bindings/app'
import { ignoreDecorativeFailure } from '../../lib/errors'
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
    <button
      type="button"
      className="change-marker"
      data-state={state || 'unknown'}
      title={title}
      aria-label={title}
      onClick={open}
      disabled={!change.webUrl}
    >
      <span className="change-marker-square" />
      <span className="change-marker-number">{number}</span>
      {change.ci && <span className={`change-marker-ci change-marker-ci-${change.ci}`} />}
    </button>
  )
}

export default ChangeMarker
