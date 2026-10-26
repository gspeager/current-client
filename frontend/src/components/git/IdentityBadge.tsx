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
}

function IdentityBadge({ identity, name, email, size = 22, onClick, pressed }: IdentityBadgeProps) {
  const style = { '--badge-size': `${size}px`, background: identity.color } as CSSProperties
  const title = email ? `${name} <${email}>` : name
  if (!onClick) {
    return (
      <span className="identity-badge" title={title} style={style}>
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
      className="identity-badge identity-badge-button"
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
