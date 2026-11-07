import { RepositoryService } from '@current-client-bindings/app'
import { useAsyncData } from '../../lib/useAsyncData'

// keepData leaves the current rows up while the list refreshes, and the
// filter drops a removed one at once, so the list never empties and redraws.
export function useRecentSummaries(recentRepos: string[]) {
  const { data } = useAsyncData(() => RepositoryService.GetRepoSummaries(recentRepos).catch(() => []), [recentRepos], {
    keepData: true,
  })
  return data?.filter((s) => recentRepos.includes(s.path)) ?? null
}
