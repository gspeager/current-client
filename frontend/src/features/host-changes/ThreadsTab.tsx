import { HostService } from '@current-client-bindings/app'
import ReviewThreadBlock from '../diff/ReviewThreadBlock'
import { useReviewActions } from '../review/useReviewActions'
import { useReviewDraft } from '../../lib/reviewDraft'
import { useAsyncData } from '../../lib/useAsyncData'
import useTabPlaceholder, { type TabProps } from './useTabPlaceholder'

function ThreadsTab({ repoPath, number, onLoaded }: TabProps) {
  const actions = useReviewActions(repoPath, number)
  const { version } = useReviewDraft()
  const { data, error } = useAsyncData(() => HostService.ChangeThreads(repoPath, number), [repoPath, number], {
    refreshKey: version,
    keepData: true,
  })
  const placeholder = useTabPlaceholder(data ? (data.threads?.length ?? 0) : undefined, error, onLoaded, 'No threads.')
  if (placeholder || !data?.threads) return placeholder
  return (
    <div className="host-change-threads">
      {data.threads.map((thread) => (
        <ReviewThreadBlock
          key={thread.id}
          thread={thread}
          where={thread.path ? `${thread.path}${thread.line ? `:${thread.line}` : ''}` : undefined}
          actions={actions?.threadActions}
        />
      ))}
    </div>
  )
}

export default ThreadsTab
