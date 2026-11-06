import { useEffect, useRef } from 'react'
import { Events } from '@wailsio/runtime'
import { RepositoryService } from '@current-client-bindings/app'
import { useDialogs } from '../../lib/useDialogs'

const OPEN_REQUESTED_EVENT = 'repo:open-requested'

interface OpenTarget {
  restored: boolean
  openFromOutside: (path: string) => void
  showOpenProblem: (problem: string) => void
}

// Repositories asked for from outside the app: a path argument, a link or the
// file manager.
export function useOpenRequests(repo: OpenTarget) {
  const { confirm } = useDialogs()
  const latest = useRef(repo)
  latest.current = repo

  useEffect(() => {
    if (!repo.restored) return
    const handle = async () => {
      const requests = await RepositoryService.TakeOpenRequests().catch(() => [])
      for (const request of requests) {
        if (request.problem) {
          latest.current.showOpenProblem(request.problem)
          continue
        }
        const allowed =
          !request.confirm ||
          (await confirm({
            title: 'Open repository from a link',
            message: `A link asks to open ${request.path}. Git runs in a repository once it's open, and a repository's settings can run programs. Only open one that's trusted.`,
            confirmLabel: 'Open',
          }))
        if (allowed) latest.current.openFromOutside(request.path)
      }
    }
    void handle()
    return Events.On(OPEN_REQUESTED_EVENT, () => void handle())
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [repo.restored])
}
