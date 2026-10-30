import { useMemo } from 'react'
import { HostService } from '@current-client-bindings/app'
import type { ReviewCapabilities } from '@current-client-bindings/core/hostapi'
import { useAsyncData } from '../../lib/useAsyncData'
import { useReviewDraft } from '../../lib/reviewDraft'

// What a thread block can do on the host for one change.
export interface ThreadActions {
  canResolve: boolean
  reply: (threadId: string, body: string) => Promise<void>
  resolve: (threadId: string, resolved: boolean) => Promise<void>
}

export interface ReviewActions {
  capabilities: ReviewCapabilities
  threadActions?: ThreadActions
  // Comments on a line now, against the head the last refresh saw.
  comment: (path: string, line: number, oldLine: number, body: string) => Promise<void>
}

// The review writes for a change, each one request on a click, followed by
// a bump of the draft's version so every view's threads read again. Nothing
// when the host is off or its API takes no review writes.
export function useReviewActions(repoPath: string | undefined, number: number | undefined): ReviewActions | undefined {
  const { wrote } = useReviewDraft()
  const { data: capabilities } = useAsyncData(
    () => (repoPath && number ? HostService.ReviewCapabilities(repoPath).catch(() => null) : null),
    [repoPath, number],
  )
  return useMemo(() => {
    if (!repoPath || !number || !capabilities) return undefined
    const any = capabilities.reply || capabilities.resolve || capabilities.lineComments || capabilities.approve
    if (!any) return undefined
    const threadActions: ThreadActions | undefined =
      capabilities.reply || capabilities.resolve
        ? {
            canResolve: capabilities.resolve,
            reply: (threadId, body) => HostService.ReplyToThread(repoPath, number, threadId, body).then(wrote),
            resolve: (threadId, resolved) =>
              HostService.ResolveThread(repoPath, number, threadId, resolved).then(wrote),
          }
        : undefined
    return {
      capabilities,
      threadActions,
      comment: (path, line, oldLine, body) =>
        HostService.CommentOnLine(repoPath, number, { path, line, oldLine, body }).then(wrote),
    }
  }, [repoPath, number, capabilities, wrote])
}
