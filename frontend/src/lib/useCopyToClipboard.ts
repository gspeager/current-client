import { useEffect, useState } from 'react'

export function useCopyToClipboard(resetMs = 1500) {
  const [copied, setCopied] = useState(false)

  useEffect(() => {
    if (!copied) return
    const timeout = setTimeout(() => setCopied(false), resetMs)
    return () => clearTimeout(timeout)
  }, [copied, resetMs])

  // With html, rich text for apps that paste formatting, and text for the rest.
  const copy = (text: string, html?: string) => {
    const written =
      html && typeof ClipboardItem !== 'undefined'
        ? navigator.clipboard.write([
            new ClipboardItem({
              'text/plain': new Blob([text], { type: 'text/plain' }),
              'text/html': new Blob([html], { type: 'text/html' }),
            }),
          ])
        : navigator.clipboard.writeText(text)
    void written.then(() => setCopied(true))
  }

  return { copied, copy }
}
