import type { CSSProperties } from 'react'
import './LabelChip.scss'

// Cool and neutral lanes only: a red label would read as an error.
const TINTS = ['--branch-integration', '--branch-feature', '--branch-spare', '--branch-sand', '--branch-experimental']

const tintFor = (label: string) => {
  let hash = 0
  for (const char of label) hash = (hash * 31 + char.charCodeAt(0)) >>> 0
  return `var(${TINTS[hash % TINTS.length]})`
}

// A host label, tinted the same way each time from its name. Square, so it
// never reads as a branch pill.
function LabelChip({ label }: { label: string }) {
  return (
    <span className="label-chip" style={{ '--tint': tintFor(label) } as CSSProperties}>
      {label}
    </span>
  )
}

export default LabelChip
