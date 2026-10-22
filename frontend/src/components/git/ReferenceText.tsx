import { Fragment, type MouseEvent, type ReactNode } from 'react'
import { PlatformService } from '@current-client-bindings/app'
import type { Links } from '@current-client-bindings/core/hostlink'
import { splitReferences, type HostReference } from '../../lib/hostReferences'
import type { ReferenceCardInfo, ReferenceCards } from '../../lib/useReferenceCards'
import HoverCard from '../chrome/HoverCard'
import './ReferenceText.scss'

interface ReferenceLinkProps {
  reference: HostReference
  // What the host knows about it; undefined shows no card.
  card?: ReferenceCardInfo
  children?: ReactNode
}

// The page opens in the user's browser; the row underneath must not also act
// on the click.
export function ReferenceLink({ reference, card, children }: ReferenceLinkProps) {
  const open = (e: MouseEvent) => {
    e.preventDefault()
    e.stopPropagation()
    void PlatformService.OpenWebPage(reference.url)
  }
  const link = (
    <a className="reference-link" href={reference.url} title={card ? undefined : reference.url} onClick={open}>
      {children ?? reference.text}
    </a>
  )
  if (!card) return link
  return <HoverCard card={<ReferenceCard reference={reference} card={card} />}>{link}</HoverCard>
}

export function ReferenceCard({ reference, card }: { reference: HostReference; card: ReferenceCardInfo }) {
  return (
    <>
      <p className="hover-card-title">{card.known ? card.title || 'Not found on the host' : 'Title not fetched'}</p>
      <p className="hover-card-line">
        <span className="hover-card-data">{reference.text}</span>
        {card.state && <span>{card.state}</span>}
        {!card.known && <span className="hover-card-muted">Refresh, or turn integration on, for the title.</span>}
      </p>
      <p className="hover-card-line hover-card-muted">{reference.url}</p>
    </>
  )
}

interface ReferenceTextProps {
  text: string
  links: Links | null
  cards?: ReferenceCards
  // Decorates each run of text, linked or not.
  render?: (segment: string) => ReactNode
}

function ReferenceText({ text, links, cards, render = (segment) => segment }: ReferenceTextProps) {
  return (
    <>
      {splitReferences(text, links).map((segment, i) =>
        typeof segment === 'string' ? (
          <Fragment key={i}>{render(segment)}</Fragment>
        ) : (
          <ReferenceLink key={i} reference={segment} card={cards?.[segment.text]}>
            {render(segment.text)}
          </ReferenceLink>
        ),
      )}
    </>
  )
}

export default ReferenceText
