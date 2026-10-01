import { HistoryService, type RefInfo } from '@current-client-bindings/app'
import { useAsyncData } from '../../lib/useAsyncData'

function groupBySha(refs: RefInfo[]): Map<string, RefInfo[]> {
  const bySha = new Map<string, RefInfo[]>()
  for (const r of refs) {
    bySha.set(r.sha, [...(bySha.get(r.sha) ?? []), r])
  }
  return bySha
}

export function useRefBadges(repoPath: string) {
  const { data } = useAsyncData(() => HistoryService.GetRefs(repoPath).then(groupBySha), [repoPath])
  return data ?? new Map<string, RefInfo[]>()
}
