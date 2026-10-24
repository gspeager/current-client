import type { MouseEvent } from 'react'
import { PlatformService } from '@current-client-bindings/app'
import { ignoreDecorativeFailure } from '../../lib/errors'
import './HostMarkdown.scss'

interface HostMarkdownProps {
  // Output of the Go renderer for host text (core/hostcontent); never raw
  // text from a host.
  html: string
  className?: string
}

// Shows a description or comment from a host. The page's Content-Security-
// Policy keeps anything in the markup from running or fetching, and links
// leave through the platform's browser opener, which takes only http(s).
function HostMarkdown({ html, className }: HostMarkdownProps) {
  const onClick = (e: MouseEvent<HTMLDivElement>) => {
    const link = (e.target as HTMLElement).closest('a')
    if (!link) return
    e.preventDefault()
    const href = link.getAttribute('href') ?? ''
    if (/^https?:\/\//i.test(href)) PlatformService.OpenWebPage(href).catch(ignoreDecorativeFailure)
  }

  return (
    <div
      className={className ? `host-markdown ${className}` : 'host-markdown'}
      onClick={onClick}
      dangerouslySetInnerHTML={{ __html: html }}
    />
  )
}

export default HostMarkdown
