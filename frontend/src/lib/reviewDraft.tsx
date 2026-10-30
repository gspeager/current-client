import { createContext, useCallback, useContext, useMemo, useState, type ReactNode } from 'react'
import type { NewComment } from '@current-client-bindings/core/hostapi'

export interface PendingComment extends NewComment {
  id: string
}

// Comments drafted for a review, kept until they are submitted together or
// discarded; nothing here reaches the host. target names the change:
// `${repoPath}#${number}`. version counts review writes, so views that
// show threads read them again.
interface ReviewDraftState {
  comments: Record<string, PendingComment[]>
  version: number
  add: (target: string, comment: NewComment) => void
  remove: (target: string, id: string) => void
  clear: (target: string) => void
  wrote: () => void
}

const NONE: ReviewDraftState = {
  comments: {},
  version: 0,
  add: () => undefined,
  remove: () => undefined,
  clear: () => undefined,
  wrote: () => undefined,
}

export const ReviewDraftContext = createContext<ReviewDraftState>(NONE)

export function reviewTarget(repoPath: string, number: number): string {
  return `${repoPath}#${number}`
}

export function ReviewDraftProvider({ children }: { children: ReactNode }) {
  const [comments, setComments] = useState<Record<string, PendingComment[]>>({})
  const [version, setVersion] = useState(0)
  const add = useCallback((target: string, comment: NewComment) => {
    const id = `${Date.now()}-${Math.random().toString(36).slice(2, 8)}`
    setComments((c) => ({ ...c, [target]: [...(c[target] ?? []), { ...comment, id }] }))
  }, [])
  const remove = useCallback((target: string, id: string) => {
    setComments((c) => ({ ...c, [target]: (c[target] ?? []).filter((p) => p.id !== id) }))
  }, [])
  const clear = useCallback((target: string) => {
    setComments((c) => ({ ...c, [target]: [] }))
  }, [])
  const wrote = useCallback(() => setVersion((v) => v + 1), [])
  const value = useMemo(
    () => ({ comments, version, add, remove, clear, wrote }),
    [comments, version, add, remove, clear, wrote],
  )
  return <ReviewDraftContext.Provider value={value}>{children}</ReviewDraftContext.Provider>
}

const NO_COMMENTS: PendingComment[] = []

export function useReviewDraft(target?: string) {
  const state = useContext(ReviewDraftContext)
  const comments = target ? (state.comments[target] ?? NO_COMMENTS) : NO_COMMENTS
  return { ...state, comments }
}
