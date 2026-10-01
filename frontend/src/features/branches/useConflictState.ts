import { ConflictService } from '@current-client-bindings/app'
import { useAsyncData } from '../../lib/useAsyncData'

export function useConflictState(repoPath: string | null, repoVersion: number) {
  const { data, reload } = useAsyncData(
    () => (repoPath ? ConflictService.GetConflictState(repoPath) : null),
    [repoPath],
    { refreshKey: repoVersion },
  )
  return { conflictState: data?.operation ? data : null, reloadConflictState: reload }
}
