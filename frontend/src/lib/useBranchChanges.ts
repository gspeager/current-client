import { HostService, type BranchChange } from '@current-client-bindings/app'
import { useAsyncData } from './useAsyncData'

export type BranchChanges = Record<string, BranchChange | undefined>

const NONE: BranchChanges = {}

// Each remote branch's merge request / pull request, keyed "origin/feature",
// from the cache the last fetch or refresh filled: drawing never asks the
// host. Empty in a build without host integration.
export function useBranchChanges(repoPath: string, refreshKey?: unknown): BranchChanges {
  const { data } = useAsyncData(() => HostService.BranchChanges(repoPath), [repoPath], { refreshKey, keepData: true })
  return data ?? NONE
}
