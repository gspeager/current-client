import { IdentityService } from '@current-client-bindings/app'
import { useAsyncData } from './useAsyncData'

export function useCurrentUser(repoPath: string | null) {
  return useAsyncData(() => (repoPath ? IdentityService.GetCurrentUser(repoPath) : null), [repoPath]).data
}
