import './StatusBadge.scss'

const COLOR_BY_STATUS: Record<string, string> = {
  A: 'var(--tertiary)',
  D: 'var(--error)',
  R: 'var(--secondary)',
  '?': 'var(--outline)',
  '!': 'var(--outline)',
}

// Renames and copies arrive with a similarity score ("R100"); only the letter matters here.
function StatusBadge({ status }: { status: string }) {
  const letter = status.charAt(0)
  const color = COLOR_BY_STATUS[letter] ?? 'var(--warning)'
  return (
    <span className="status-badge" style={{ color, background: `color-mix(in srgb, ${color} 18%, transparent)` }}>
      {letter}
    </span>
  )
}

export default StatusBadge
