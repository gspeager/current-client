import { MessageSquare } from 'lucide-react'
import { plural } from '../../lib/plural'
import './ThreadCount.scss'

function ThreadCount({ count }: { count?: number }) {
  if (!count) return null
  return (
    <span className="thread-count" title={plural(count, 'unresolved review thread')}>
      <MessageSquare size={10} strokeWidth={2} aria-hidden="true" />
      {count}
    </span>
  )
}

export default ThreadCount
