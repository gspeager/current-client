import { useEffect, useRef, useState } from 'react'
import { RepositoryService, type RepoSummaryInfo } from '@current-client-bindings/app'
import { useCancellableOperation } from '../../lib/cancellableOperation'
import { errorMessage } from '../../lib/errors'
import { cloneFolderName } from './cloneFolderName'

// initialRepoPath, when given, is focused instead of the saved focused tab.
export function useRepositoryLifecycle(initialRepoPath: string | null = null) {
  const [repoPath, setRepoPathState] = useState<string | null>(null)
  const [repoError, setRepoError] = useState<string | null>(null)
  const [cloneUrl, setCloneUrl] = useState('')
  const [cloneDest, setCloneDest] = useState('')
  // null follows the URL; set once the user types their own folder name.
  const [cloneFolderOverride, setCloneFolder] = useState<string | null>(null)
  const cloneFolder = cloneFolderOverride ?? cloneFolderName(cloneUrl)
  const [recentRepos, setRecentRepos] = useState<string[]>([])
  const [tabs, setTabs] = useState<string[]>([])
  const [tabSummaries, setTabSummaries] = useState<Record<string, RepoSummaryInfo>>({})
  const cloneOp = useCancellableOperation()
  const initStarted = useRef(false)

  const refreshRecentRepos = () => {
    RepositoryService.GetRecentRepositories()
      .then(setRecentRepos)
      .catch(() => undefined)
  }

  const refreshTabSummaries = (paths: string[]) => {
    if (paths.length === 0) return
    RepositoryService.GetRepoSummaries(paths)
      .then((results) => {
        setTabSummaries((prev) => {
          const next = { ...prev }
          for (const r of results) {
            next[r.path] = r
          }
          return next
        })
      })
      .catch(() => undefined)
  }

  const persistTabs = (nextTabs: string[], focused: string | null) => {
    RepositoryService.SetOpenTabs(nextTabs, focused ?? '').catch(() => undefined)
  }

  const addTab = (path: string) => {
    setRepoPathState(path)
    setTabs((prev) => {
      const next = prev.includes(path) ? prev : [...prev, path]
      persistTabs(next, path)
      return next
    })
    refreshTabSummaries([path])
  }

  const openPath = (open: Promise<string>) => {
    setRepoError(null)
    open
      .then((path) => {
        addTab(path)
        refreshRecentRepos()
      })
      .catch((err: unknown) => setRepoError(errorMessage(err)))
  }

  // Restores saved tabs, falling back to the most recent repo.
  useEffect(() => {
    if (initStarted.current) return
    initStarted.current = true
    Promise.all([
      RepositoryService.GetRecentRepositories().catch(() => []),
      RepositoryService.GetOpenTabs().catch(() => null),
    ]).then(([recentsResult, tabsResult]) => {
      const recentList = recentsResult
      setRecentRepos(recentList)
      const savedTabs = tabsResult?.tabs ?? []
      if (initialRepoPath) {
        setTabs(savedTabs)
        refreshTabSummaries(savedTabs)
        openPath(RepositoryService.OpenRepository(initialRepoPath))
      } else if (savedTabs.length > 0) {
        setTabs(savedTabs)
        refreshTabSummaries(savedTabs)
        const focused =
          tabsResult?.focused && savedTabs.includes(tabsResult.focused) ? tabsResult.focused : savedTabs[0]
        RepositoryService.OpenRepository(focused)
          .then((path) => setRepoPathState(path))
          .catch((err: unknown) => setRepoError(errorMessage(err)))
      } else if (recentList.length > 0) {
        openPath(RepositoryService.OpenRepository(recentList[0]))
      }
    })
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  const pickAndOpen = (run: (path: string) => Promise<string>) => {
    setRepoError(null)
    RepositoryService.PickRepositoryFolder()
      .then((path) => {
        if (path) {
          openPath(run(path))
        }
      })
      .catch((err: unknown) => setRepoError(errorMessage(err)))
  }

  const openRepository = () => pickAndOpen(RepositoryService.OpenRepository)
  const initRepository = () => pickAndOpen(RepositoryService.InitRepository)
  const openRecent = (path: string) => openPath(RepositoryService.OpenRepository(path))

  // Removed from the list straight away; the saved list is only re-read if saving fails.
  const removeRecent = (path: string) => {
    setRecentRepos((prev) => prev.filter((p) => p !== path))
    RepositoryService.RemoveRecentRepository(path).catch(refreshRecentRepos)
  }

  const closeTab = (path: string) => {
    setTabs((prev) => {
      const idx = prev.indexOf(path)
      if (idx === -1) return prev
      const next = prev.filter((p) => p !== path)
      let nextFocused = repoPath
      if (repoPath === path) {
        const candidate = next[idx] ?? next[idx - 1] ?? null
        nextFocused = candidate
        if (candidate) {
          RepositoryService.OpenRepository(candidate)
            .then((resolved) => setRepoPathState(resolved))
            .catch((err: unknown) => setRepoError(errorMessage(err)))
        } else {
          setRepoPathState(null)
        }
      }
      persistTabs(next, nextFocused)
      return next
    })
    setTabSummaries((prev) => {
      if (!(path in prev)) return prev
      const next = { ...prev }
      delete next[path]
      return next
    })
  }

  const reorderTabs = (nextOrder: string[]) => {
    setTabs(nextOrder)
    persistTabs(nextOrder, repoPath)
  }

  const chooseCloneDestination = () => {
    RepositoryService.PickDestinationFolder()
      .then((path) => {
        if (path) {
          setCloneDest(path)
        }
      })
      .catch((err: unknown) => setRepoError(errorMessage(err)))
  }

  const cloneRepository = () => {
    setRepoError(null)
    cloneOp.run<string>(
      (auth) => RepositoryService.CloneRepository(cloneUrl, cloneDest, cloneFolder.trim(), auth),
      (path) => {
        addTab(path)
        refreshRecentRepos()
        setCloneFolder(null)
      },
    )
  }

  return {
    repoPath,
    repoError,
    cloneUrl,
    setCloneUrl,
    cloneDest,
    cloneFolder,
    setCloneFolder,
    recentRepos,
    cloneOp,
    openRepository,
    initRepository,
    openRecent,
    removeRecent,
    chooseCloneDestination,
    cloneRepository,
    tabs,
    tabSummaries,
    closeTab,
    reorderTabs,
  }
}
