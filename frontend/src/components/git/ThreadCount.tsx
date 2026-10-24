import { plural } from '../../lib/plural'
import '../../features/diff/ReviewThreadBlock.scss'

// Unresolved review threads on a file; nothing when there are none.
function ThreadCount({ count }: { count?: number }) {
  if (!count) return null
  return (
    <span className="thread-count" title={plural(count, 'unresolved review thread')}>
      {count}
    </span>
  )
}

export default ThreadCount
