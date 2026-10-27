import { useMemo } from 'react'
import { HostService, type EnvironmentInfo } from '@current-client-bindings/app'
import { useAsyncData } from './useAsyncData'

// The environments the last refresh found, by the commit each runs. Empty
// while the host is off or in a build without host integration; reading it
// asks no host.
export function useDeploymentsBySha(repoPath: string, refreshKey?: unknown): Map<string, EnvironmentInfo[]> {
  const { data } = useAsyncData(() => HostService.Deployments(repoPath).catch(() => null), [repoPath], {
    refreshKey,
    keepData: true,
  })
  return useMemo(() => {
    const bySha = new Map<string, EnvironmentInfo[]>()
    for (const env of data?.environments ?? []) bySha.set(env.sha, [...(bySha.get(env.sha) ?? []), env])
    return bySha
  }, [data])
}
