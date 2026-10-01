import type { IdentityInfo } from '@current-client-bindings/app'
interface IdentityBadgeProps {
  identity: IdentityInfo
  name: string
  email: string
  size?: number
}

function IdentityBadge({ identity, name, email, size = 22 }: IdentityBadgeProps) {
  return (
    <span
      title={email ? `${name} <${email}>` : name}
      style={{
        display: 'inline-flex',
        alignItems: 'center',
        justifyContent: 'center',
        width: size,
        height: size,
        borderRadius: '50%',
        background: identity.color,
        color: '#0a0e16',
        fontFamily: 'ui-monospace, SFMono-Regular, Menlo, Consolas, "Liberation Mono", monospace',
        fontSize: Math.round(size * 0.42),
        fontWeight: 600,
        lineHeight: 1,
        flexShrink: 0,
      }}
    >
      {identity.initials}
    </span>
  )
}

export default IdentityBadge
