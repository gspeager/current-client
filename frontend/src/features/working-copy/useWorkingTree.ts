import { StatusService } from '@current-client-bindings/app'
import { isStagedStatus, isUnstagedStatus } from './fileStatus'
import { useAsyncData } from '../../lib/useAsyncData'

export type WorkingTreeSection = 'conflicted' | 'staged' | 'unstaged'

export function useWorkingTree(repoPath: string | null) {
  const {
    data: files,
    error,
    reload: loadStatus,
  } = useAsyncData(() => (repoPath ? StatusService.GetStatus(repoPath) : null), [repoPath])

  // A conflicted file shows as changed on both sides, so it gets its own list.
  const conflicted = files?.filter((f) => f.conflicted) ?? []
  const staged = files?.filter((f) => !f.conflicted && isStagedStatus(f.indexStatus)) ?? []
  const unstaged = files?.filter((f) => !f.conflicted && isUnstagedStatus(f.worktreeStatus)) ?? []

  return { files, conflicted, staged, unstaged, error, loadStatus }
}
