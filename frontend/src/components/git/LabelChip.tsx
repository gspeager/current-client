import type { CSSProperties } from 'react'
import { useLaneColors } from '../../lib/laneColor'
import './LabelChip.scss'

// A host label, tinted the same way each time from its name. Square, so it
// never reads as a branch pill.
function LabelChip({ label }: { label: string }) {
  const { branchColor } = useLaneColors()
  return (
    <span className="label-chip" style={{ '--lane': branchColor(label, false) } as CSSProperties}>
      {label}
    </span>
  )
}

export default LabelChip
