import { useMemo } from 'react'
import { HostService, type ReviewThread, type ReviewThreads } from '@current-client-bindings/app'
import { useAsyncData } from './useAsyncData'

export interface ReviewThreadsView {
  enabled: boolean
  number: number
  prefix: string
  // Threads pinned to a line, by path, still current and in line order.
  byPath: Record<string, ReviewThread[] | undefined>
  // Threads whose code has changed since, by path.
  outdatedByPath: Record<string, ReviewThread[] | undefined>
  // Unresolved, still-current threads per path.
  counts: Record<string, number | undefined>
  total: number
}

const NONE: ReviewThreadsView = {
  enabled: false,
  number: 0,
  prefix: '',
  byPath: {},
  outdatedByPath: {},
  counts: {},
  total: 0,
}

// The checked-out branch's review threads, from the last refresh. Reading
// them never contacts the host. With at, only those on files that are the same
// at that commit as at the change's head, where their lines still match.
export function useReviewThreads(repoPath: string, refreshKey?: unknown, at?: string): ReviewThreadsView {
  const { data } = useAsyncData(
    () => (at === undefined ? HostService.ReviewThreads(repoPath) : HostService.ReviewThreadsAt(repoPath, at)),
    [repoPath, at],
    { refreshKey, keepData: at === undefined },
  )
  return useMemo(() => (data ? groupThreads(data) : NONE), [data])
}

export function groupThreads(data: ReviewThreads): ReviewThreadsView {
  const byPath: Record<string, ReviewThread[]> = {}
  const outdatedByPath: Record<string, ReviewThread[]> = {}
  for (const thread of data.threads ?? []) {
    if (!thread.path) continue
    const bucket = thread.outdated ? outdatedByPath : byPath
    ;(bucket[thread.path] ??= []).push(thread)
  }
  for (const threads of Object.values(byPath)) threads.sort((a, b) => a.line - b.line)
  return {
    enabled: data.enabled,
    number: data.number,
    prefix: data.prefix,
    byPath,
    outdatedByPath,
    counts: data.unresolved ?? {},
    total: data.total,
  }
}
