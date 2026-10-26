import { useMemo } from 'react'
import { CodeownersService } from '@current-client-bindings/app'
import { useAsyncData } from './useAsyncData'

export type FileOwners = Record<string, string[]>

// CODEOWNERS owners for paths, read from the working tree; empty when the
// repository has no file or the read fails.
export function useFileOwners(repoPath: string, paths: string[]): FileOwners {
  const key = useMemo(() => [...new Set(paths)].sort().join('\n'), [paths])
  const { data } = useAsyncData(
    () => (key ? CodeownersService.Owners(repoPath, key.split('\n')).catch(() => ({})) : null),
    [repoPath, key],
    { keepData: true },
  )
  return data ?? {}
}
