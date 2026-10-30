import { HistoryService, type CommitInfo, type GraphNode } from '@current-client-bindings/app'
import { useAsyncData } from '../../lib/useAsyncData'

const WORKING_SHA = 'working-tree'

// With firstParent the layout sees only each commit's first parent, so the
// main line is one lane and no lane waits for a parent the log left out.
// With workingTreeParent, a node for the working tree goes first, so the
// layout joins it to HEAD and carries the lane through the rows between.
export function useCommitGraph(commits: CommitInfo[], firstParent = false, workingTreeParent: string | null = null) {
  const { data } = useAsyncData(
    () => {
      if (commits.length === 0) return null
      const refs = commits.map((c) => ({
        sha: c.sha,
        parentShas: firstParent ? c.parentShas.slice(0, 1) : c.parentShas,
      }))
      if (workingTreeParent) refs.unshift({ sha: WORKING_SHA, parentShas: [workingTreeParent] })
      return HistoryService.ComputeGraphLayout(refs)
    },
    [commits, firstParent, workingTreeParent],
    { keepData: true },
  )
  const nodes: GraphNode[] = data ?? []
  if (!workingTreeParent || nodes[0]?.sha !== WORKING_SHA) return { graphNodes: nodes, workingNode: undefined }
  return { graphNodes: nodes.slice(1), workingNode: nodes[0] }
}
