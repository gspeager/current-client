import { TriangleAlert } from 'lucide-react'
import type { OverlapInfo } from '@current-client-bindings/app'
import './BranchRow.scss'

function ConflictsBadge({ overlap }: { overlap?: OverlapInfo }) {
  if (!overlap) return null
  return (
    <span
      className="branch-conflicts-badge"
      title={`Would conflict with the current branch: ${overlap.files.join(', ')}`}
    >
      <TriangleAlert size={12} strokeWidth={1.75} aria-label="Conflicts" />
    </span>
  )
}

export default ConflictsBadge
