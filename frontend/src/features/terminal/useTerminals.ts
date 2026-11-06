import { useCallback, useRef, useState } from 'react'
import { TerminalService } from '@current-client-bindings/app'
import { useDialogs } from '../../lib/useDialogs'
import { errorMessage } from '../../lib/errors'

export interface Shell {
  id: string
  number: number
}

export type PanelView = 'shells' | 'tasks'

interface RepoTerminals {
  shells: Shell[]
  active: string | null
  open: boolean
  view: PanelView
}

const NONE: RepoTerminals = { shells: [], active: null, open: false, view: 'shells' }

// Each repository tab has its own shells, which keep running in the background.
export function useTerminals(repoPath: string | null) {
  const { confirm } = useDialogs()
  const [byRepo, setByRepo] = useState<Record<string, RepoTerminals>>({})
  const [error, setError] = useState<string | null>(null)
  const counter = useRef(0)

  const update = useCallback(
    (repo: string, change: (prev: RepoTerminals) => RepoTerminals) =>
      setByRepo((prev) => ({ ...prev, [repo]: change(prev[repo] ?? NONE) })),
    [],
  )

  const current = (repoPath && byRepo[repoPath]) || NONE

  const newShell = useCallback(
    (repo: string) => {
      setError(null)
      // The size follows from the panel once it has drawn.
      TerminalService.Start(repo, 80, 24)
        .then((id) => {
          counter.current += 1
          const shell = { id, number: counter.current }
          update(repo, (prev) => ({ shells: [...prev.shells, shell], active: id, open: true, view: 'shells' }))
        })
        .catch((err: unknown) => setError(errorMessage(err)))
    },
    [update],
  )

  const removeShell = useCallback(
    (repo: string, id: string) =>
      update(repo, (prev) => {
        const shells = prev.shells.filter((shell) => shell.id !== id)
        const active = prev.active === id ? (shells[shells.length - 1]?.id ?? null) : prev.active
        return { ...prev, shells, active, open: prev.open && (shells.length > 0 || prev.view === 'tasks') }
      }),
    [update],
  )

  const closeShell = async (repo: string, id: string) => {
    if (
      (await TerminalService.Busy(id).catch(() => false)) &&
      !(await confirm({
        title: 'Close terminal',
        message: 'A process is still running in this terminal. Closing ends it.',
        confirmLabel: 'Close terminal',
        destructive: true,
      }))
    ) {
      return
    }
    await TerminalService.Close(id).catch(() => undefined)
    removeShell(repo, id)
  }

  // For closing a repository's tab: false when the user would rather keep it open.
  const confirmCloseRepo = async (repo: string): Promise<boolean> => {
    if (
      (await TerminalService.RepoBusy(repo).catch(() => false)) &&
      !(await confirm({
        title: 'Close repository tab',
        message: 'A process is still running in one of its terminals. Closing the tab ends it.',
        confirmLabel: 'Close tab',
        destructive: true,
      }))
    ) {
      return false
    }
    await TerminalService.CloseRepo(repo).catch(() => undefined)
    setByRepo((prev) => {
      const next = { ...prev }
      delete next[repo]
      return next
    })
    return true
  }

  // Opens the panel, starting a shell if there isn't one.
  const show = (repo: string) => {
    update(repo, (prev) => ({ ...prev, open: true, view: 'shells' }))
    if (current.shells.length === 0) newShell(repo)
  }
  const showTasks = (repo: string) => update(repo, (prev) => ({ ...prev, open: true, view: 'tasks' }))
  const setView = (repo: string, view: PanelView) => update(repo, (prev) => ({ ...prev, view }))
  const hide = (repo: string) => update(repo, (prev) => ({ ...prev, open: false }))
  const select = (repo: string, id: string) => update(repo, (prev) => ({ ...prev, active: id }))

  const running = Object.entries(byRepo)
    .filter(([, repo]) => repo.shells.length > 0)
    .map(([path]) => path)

  return {
    byRepo,
    current,
    error,
    running,
    newShell,
    closeShell,
    removeShell,
    confirmCloseRepo,
    show,
    showTasks,
    setView,
    hide,
    select,
  }
}
