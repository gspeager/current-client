import { CircleCheck, TriangleAlert } from 'lucide-react'
import { plural } from '../../lib/plural'

function MergeVerdict({ facts }: { facts: { level: string }[] }) {
  const blocking = facts.filter((f) => f.level === 'warn').length
  const Icon = blocking ? TriangleAlert : CircleCheck
  return (
    <div className={`host-change-verdict host-change-verdict-${blocking ? 'warn' : 'ok'}`}>
      <span className="host-change-verdict-text">
        <Icon size={16} strokeWidth={1.75} aria-hidden="true" />
        {blocking ? `${plural(blocking, 'item')} to resolve before merging` : 'Ready to merge'}
      </span>
    </div>
  )
}

export default MergeVerdict
