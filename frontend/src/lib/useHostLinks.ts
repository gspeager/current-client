import { RemoteService } from '@current-client-bindings/app'
import type { Links } from '@current-client-bindings/core/hostlink'
import { useAsyncData } from './useAsyncData'

// null until loaded, and when no remote points at a hosted project.
export function useHostLinks(repoPath: string): Links | null {
  const { data } = useAsyncData(() => RemoteService.HostLinks(repoPath), [repoPath])
  return data
}
