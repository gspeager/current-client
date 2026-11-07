import './StatusBadge.scss'

const COLOR_BY_STATUS: Record<string, string> = {
  A: 'var(--tertiary)',
  M: 'var(--on-surface-variant)',
  D: 'var(--error)',
  U: 'var(--error)',
  R: 'var(--secondary)',
  '?': 'var(--outline)',
  '!': 'var(--outline)',
}

const LABEL_BY_STATUS: Record<string, string> = {
  A: 'Added',
  M: 'Modified',
  D: 'Deleted',
  R: 'Renamed',
  C: 'Copied',
  U: 'Conflicted',
  '?': 'Untracked',
  '!': 'Ignored',
}

// Renames and copies arrive with a similarity score ("R100"); only the letter matters here.
function StatusBadge({ status }: { status: string }) {
  const letter = status.charAt(0)
  const color = COLOR_BY_STATUS[letter] ?? 'var(--warning)'
  return (
    <span
      className="status-badge"
      title={LABEL_BY_STATUS[letter]}
      style={{ color, background: `color-mix(in srgb, ${color} 18%, transparent)` }}
    >
      {letter}
    </span>
  )
}

export default StatusBadge
