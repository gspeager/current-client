import { useEffect, useRef, useState } from 'react'
import { HistoryService, type CommitInfo, type HistoryFilterInfo } from '@current-client-bindings/app'
import { errorMessage } from '../../lib/errors'

export const EMPTY_HISTORY_FILTER: HistoryFilterInfo = { ref: '', since: '', until: '', author: '', type: '' }

const PAGE_SIZE = 200

// initialLoadThrough enlarges the first page so a deep search match and everything above it load together.
export function useCommitHistory(
  repoPath: string,
  filter: HistoryFilterInfo = EMPTY_HISTORY_FILTER,
  initialLoadThrough?: number | null,
) {
  const [commits, setCommits] = useState<CommitInfo[]>([])
  const [loading, setLoading] = useState(false)
  const [done, setDone] = useState(false)
  const [error, setError] = useState<string | null>(null)
  // Pages requested before the latest repo/filter change are discarded.
  const generation = useRef(0)

  const fetchPage = (skip: number, limit: number) => {
    const requested = generation.current
    setLoading(true)
    HistoryService.GetHistory(repoPath, limit, skip, filter)
      .then((page) => {
        if (requested !== generation.current) return
        setCommits((prev) => [...prev, ...page])
        if (page.length < limit) setDone(true)
      })
      .catch((err: unknown) => {
        if (requested === generation.current) setError(errorMessage(err))
      })
      .finally(() => {
        if (requested === generation.current) setLoading(false)
      })
  }

  const loadMore = (skip: number, isDone: boolean) => {
    if (loading || isDone) return
    fetchPage(skip, PAGE_SIZE)
  }

  useEffect(() => {
    generation.current++
    setCommits([])
    setDone(false)
    setError(null)
    fetchPage(0, initialLoadThrough ? initialLoadThrough + 50 : PAGE_SIZE)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [repoPath, filter.ref, filter.since, filter.until, filter.author, filter.type])

  return { commits, loading, done, error, loadMore }
}
