import { HistoryService } from '@current-client-bindings/app'
import { useAsyncData } from '../../lib/useAsyncData'
import { EMPTY_HISTORY_FILTER } from '../history/useCommitHistory'

export function useHeadCommit(repoPath: string, refreshKey: number = 0) {
  const { data } = useAsyncData(() => HistoryService.GetHistory(repoPath, 1, 0, EMPTY_HISTORY_FILTER), [repoPath], {
    refreshKey,
  })
  return data?.[0] ?? null
}
