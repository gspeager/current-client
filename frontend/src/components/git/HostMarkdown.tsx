import { useLayoutEffect, useRef, type MouseEvent } from 'react'
import { openWebPage } from '../../lib/hostPages'
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
    if (/^https?:\/\//i.test(href)) openWebPage(href)
  }

  const root = useRef<HTMLDivElement>(null)
  // The renderer leaves a task's checkbox unlabelled; its item names it.
  useLayoutEffect(() => {
    root.current?.querySelectorAll<HTMLInputElement>('li > input[type="checkbox"]').forEach((box) => {
      box.setAttribute('aria-label', box.parentElement?.textContent?.trim() ?? 'Task')
    })
  }, [html])

  return (
    <div
      ref={root}
      className={className ? `host-markdown ${className}` : 'host-markdown'}
      onClick={onClick}
      dangerouslySetInnerHTML={{ __html: html }}
    />
  )
}

export default HostMarkdown
