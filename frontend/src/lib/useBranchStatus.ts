import { BranchService } from '@current-client-bindings/app'
import { useAsyncData } from './useAsyncData'

export function useBranchStatus(repoPath: string, refreshKey: number) {
  return useAsyncData(() => BranchService.CurrentBranchStatus(repoPath), [repoPath], { refreshKey }).data
}
