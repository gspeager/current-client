import './DiffStat.scss'

interface DiffStatProps {
  added: number
  removed: number
  binary: boolean
}

// Renders nothing without line counts rather than a misleading "+0 -0".
function DiffStat({ added, removed, binary }: DiffStatProps) {
  if (binary) {
    return <span className="diff-stat diff-stat-binary">Binary</span>
  }
  if (added === 0 && removed === 0) {
    return null
  }
  return (
    <span className="diff-stat">
      <span className="diff-stat-added">{added > 0 && `+${added}`}</span>
      <span className="diff-stat-removed">{removed > 0 && `-${removed}`}</span>
    </span>
  )
}

export default DiffStat
