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
  const addedShare = (added / (added + removed)) * 100
  return (
    <span className="diff-stat">
      <span className="diff-stat-added">+{added}</span>
      <span className="diff-stat-removed">-{removed}</span>
      <span className="diff-stat-bar" aria-hidden="true">
        <span className="diff-stat-bar-added" style={{ width: `${addedShare}%` }} />
        <span className="diff-stat-bar-removed" style={{ width: `${100 - addedShare}%` }} />
      </span>
    </span>
  )
}

export default DiffStat
