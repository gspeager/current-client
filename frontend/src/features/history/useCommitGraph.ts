import { HistoryService, type CommitInfo } from '@current-client-bindings/app'
import { useAsyncData } from '../../lib/useAsyncData'

// With firstParent the layout sees only each commit's first parent, so the
// main line is one lane and no lane waits for a parent the log left out.
export function useCommitGraph(commits: CommitInfo[], firstParent = false) {
  const { data } = useAsyncData(
    () =>
      commits.length === 0
        ? null
        : HistoryService.ComputeGraphLayout(
            commits.map((c) => ({ sha: c.sha, parentShas: firstParent ? c.parentShas.slice(0, 1) : c.parentShas })),
          ),
    [commits, firstParent],
    { keepData: true },
  )
  return data ?? []
}
