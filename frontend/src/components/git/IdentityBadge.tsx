import type { CSSProperties } from 'react'
import type { IdentityInfo } from '@current-client-bindings/app'
import './IdentityBadge.scss'

interface IdentityBadgeProps {
  identity: IdentityInfo
  name: string
  email: string
  size?: number
}

function IdentityBadge({ identity, name, email, size = 22 }: IdentityBadgeProps) {
  const style = { '--badge-size': `${size}px`, background: identity.color } as CSSProperties
  return (
    <span className="identity-badge" title={email ? `${name} <${email}>` : name} style={style}>
      {identity.initials}
    </span>
  )
}

export default IdentityBadge
