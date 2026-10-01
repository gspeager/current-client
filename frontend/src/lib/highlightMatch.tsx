import type { ReactNode } from 'react'

// Only the first case-insensitive match is marked.
export function highlightMatch(text: string, query: string, className: string): ReactNode {
  if (!query) return text
  const index = text.toLowerCase().indexOf(query.toLowerCase())
  if (index === -1) return text
  return (
    <>
      {text.slice(0, index)}
      <mark className={className}>{text.slice(index, index + query.length)}</mark>
      {text.slice(index + query.length)}
    </>
  )
}
