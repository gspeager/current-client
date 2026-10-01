import { HistoryService, type CommitInfo } from '@current-client-bindings/app'
import { useAsyncData } from '../../lib/useAsyncData'

export function useCommitGraph(commits: CommitInfo[]) {
  const { data } = useAsyncData(
    () =>
      commits.length === 0
        ? null
        : HistoryService.ComputeGraphLayout(commits.map((c) => ({ sha: c.sha, parentShas: c.parentShas }))),
    [commits],
    { keepData: true },
  )
  return data ?? []
}
