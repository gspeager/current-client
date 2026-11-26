import clsx from 'clsx'
import type { CSSProperties, MouseEvent } from 'react'
import type { IdentityInfo } from '@current-client-bindings/app'
import './IdentityBadge.scss'

interface IdentityBadgeProps {
  identity: IdentityInfo
  name: string
  email: string
  size?: number
  // With onClick the badge is a button and the click stays off the row.
  onClick?: () => void
  pressed?: boolean
  // Toned down where a column of them sits beside text.
  muted?: boolean
}

function IdentityBadge({ identity, name, email, size = 22, onClick, pressed, muted = false }: IdentityBadgeProps) {
  const className = clsx('identity-badge', onClick && 'identity-badge-button', muted && 'identity-badge-muted')
  const style = { '--badge-size': `${size}px`, background: identity.color } as CSSProperties
  const title = email ? `${name} <${email}>` : name
  if (!onClick) {
    return (
      <span className={className} title={title} style={style}>
        {identity.initials}
      </span>
    )
  }
  const click = (e: MouseEvent) => {
    e.stopPropagation()
    onClick()
  }
  return (
    <button
      type="button"
      className={className}
      title={`${title}\n${pressed ? 'Clear the highlight' : 'Highlight this author’s commits'}`}
      aria-label={`${pressed ? 'Clear the highlight of' : 'Highlight commits by'} ${name}`}
      aria-pressed={pressed}
      style={style}
      onClick={click}
    >
      {identity.initials}
    </button>
  )
}

export default IdentityBadge
