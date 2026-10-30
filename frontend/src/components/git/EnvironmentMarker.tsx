import type { MouseEvent } from 'react'
import { Rocket } from 'lucide-react'
import { PlatformService, type EnvironmentInfo, type Reach } from '@current-client-bindings/app'
import HoverCard from '../chrome/HoverCard'
import { CIIcon, ciLabel } from './CIMark'
import { ignoreDecorativeFailure } from '../../lib/errors'
import { relativeTime } from '../../lib/relativeTime'
import './EnvironmentMarker.scss'

// The environments whose latest deployment is this commit: one rocket at the
// end of the subject, each environment and how its deploy went in the card.
// Clicking opens the first one on the host.
export function EnvironmentMarker({ environments }: { environments: EnvironmentInfo[] }) {
  const names = environments.map((e) => e.environment).join(', ')
  const open = (e: MouseEvent) => {
    e.stopPropagation()
    const url = environments.find((env) => env.webUrl)?.webUrl
    if (url) PlatformService.OpenWebPage(url).catch(ignoreDecorativeFailure)
  }
  const card = environments.map((env) => {
    const by = env.deployedBy.name || env.deployedBy.username
    return (
      <div key={env.environment}>
        <p className="hover-card-title">{env.environment}</p>
        <p className="hover-card-line">
          <CIIcon status={env.status} />
          <span>{ciLabel(env.status).replace('CI ', 'Deploy ')}</span>
        </p>
        <p className="hover-card-line hover-card-muted">
          {[env.ref && `from ${env.ref}`, env.deployedAt && relativeTime(new Date(env.deployedAt)), by && `by ${by}`]
            .filter(Boolean)
            .join(' · ')}
        </p>
      </div>
    )
  })
  return (
    <HoverCard card={<>{card}</>}>
      <button type="button" className="environment-marker" onClick={open} aria-label={`Deployed to ${names}`}>
        <Rocket size={14} strokeWidth={1.5} />
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
