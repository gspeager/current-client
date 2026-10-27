import type { MouseEvent } from 'react'
import { PlatformService, type EnvironmentInfo, type Reach } from '@current-client-bindings/app'
import HoverCard from '../chrome/HoverCard'
import { CIIcon, ciLabel } from './CIMark'
import { ignoreDecorativeFailure } from '../../lib/errors'
import { relativeTime } from '../../lib/relativeTime'
import './EnvironmentMarker.scss'

// Where an environment's latest deployment runs: a square marker on that
// commit, like a tag, with how it went. Clicking opens the environment on
// the host.
export function EnvironmentMarker({ environment }: { environment: EnvironmentInfo }) {
  const open = (e: MouseEvent) => {
    e.stopPropagation()
    if (environment.webUrl) PlatformService.OpenWebPage(environment.webUrl).catch(ignoreDecorativeFailure)
  }
  const by = environment.deployedBy.name || environment.deployedBy.username
  const card = (
    <>
      <p className="hover-card-title">{environment.environment}</p>
      <p className="hover-card-line">
        <CIIcon status={environment.status} />
        <span>{ciLabel(environment.status).replace('CI ', 'Deploy ')}</span>
      </p>
      <p className="hover-card-line hover-card-muted">
        {[
          environment.ref && `from ${environment.ref}`,
          environment.deployedAt && relativeTime(new Date(environment.deployedAt)),
          by && `by ${by}`,
        ]
          .filter(Boolean)
          .join(' · ')}
      </p>
    </>
  )
  return (
    <HoverCard card={card}>
      <button
        type="button"
        className={`environment-marker${environment.production ? ' environment-marker-production' : ''}`}
        onClick={open}
        aria-label={`${environment.environment} is deployed here`}
      >
        <CIIcon status={environment.status} size={10} />
        {environment.environment}
      </button>
    </HoverCard>
  )
}

const REACH_WORDS: Record<string, string> = {
  in: 'Reached',
  out: 'Not yet',
  unknown: 'Unknown: the deployed commit isn’t fetched',
}

// Where a commit has reached, from the last refresh's deployments and Git.
export function EnvironmentReach({ reach }: { reach: Reach[] }) {
  return (
    <ul className="environment-reach" aria-label="Environments">
      {reach.map((r) => (
        <li key={r.environment} className={`environment-reach-${r.state}`}>
          <span className="environment-reach-name">{r.environment}</span>
          <span>{REACH_WORDS[r.state] ?? r.state}</span>
        </li>
      ))}
    </ul>
  )
}

// The environments a branch tip is in, as small squares beside the branch.
export function EnvironmentChips({ reach }: { reach: Reach[] | undefined }) {
  const reached = (reach ?? []).filter((r) => r.state === 'in')
  if (reached.length === 0) return null
  return (
    <span className="environment-chips">
      {reached.map((r) => (
        <span key={r.environment} className="environment-chip" title={`In ${r.environment}`}>
          {r.environment}
        </span>
      ))}
    </span>
  )
}
