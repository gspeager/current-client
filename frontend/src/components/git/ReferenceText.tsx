import { Fragment, type MouseEvent, type ReactNode } from 'react'
import { PlatformService } from '@current-client-bindings/app'
import type { Links } from '@current-client-bindings/core/hostlink'
import { splitReferences, type HostReference } from '../../lib/hostReferences'
import './ReferenceText.scss'

interface ReferenceLinkProps {
  reference: HostReference
  children?: ReactNode
}

// The page opens in the user's browser; the row underneath must not also act
// on the click.
export function ReferenceLink({ reference, children }: ReferenceLinkProps) {
  const open = (e: MouseEvent) => {
    e.preventDefault()
    e.stopPropagation()
    void PlatformService.OpenWebPage(reference.url)
  }
  return (
    <a className="reference-link" href={reference.url} title={reference.url} onClick={open}>
      {children ?? reference.text}
    </a>
  )
}

interface ReferenceTextProps {
  text: string
  links: Links | null
  // Decorates each run of text, linked or not.
  render?: (segment: string) => ReactNode
}

function ReferenceText({ text, links, render = (segment) => segment }: ReferenceTextProps) {
  return (
    <>
      {splitReferences(text, links).map((segment, i) =>
        typeof segment === 'string' ? (
          <Fragment key={i}>{render(segment)}</Fragment>
        ) : (
          <ReferenceLink key={i} reference={segment}>
            {render(segment.text)}
          </ReferenceLink>
        ),
      )}
    </>
  )
}

export default ReferenceText
