import { useState } from 'react'
import { HistoryService } from '@current-client-bindings/app'
import type { AppTab } from '../repositories/TabBar'
import type { useTabMemory } from '../repositories/useTabMemory'
import { usePerRepository } from '../../lib/usePerRepository'

// Each request remounts History so its initial selection, deep load, and scroll apply again.
interface CommitFocus {
  position: number | null
  ref: string | null
  request: number
}

type TabMemory = ReturnType<typeof useTabMemory>

interface NavigationOptions {
  repoPath: string | null
  setActiveTab: TabMemory['setActiveTab']
  remember: TabMemory['remember']
  // A tab chosen by hand closes what was opened over the last one.
  onChangeTab: () => void
}

// Going to a view, and to a commit, file or change in one.
export function useNavigation({ repoPath, setActiveTab, remember, onChangeTab }: NavigationOptions) {
  const [commitFocus, setCommitFocus] = usePerRepository<CommitFocus | null>(repoPath, null)
  // Bumped to remount Changes on a newly requested file, as commitFocus does for History.
  const [fileFocus, setFileFocus] = useState(0)
  const [changeFocus, setChangeFocus] = useState<number | null>(null)

  const changeTab = (tab: AppTab) => {
    onChangeTab()
    setActiveTab(tab)
    setCommitFocus(null)
  }

  // position is only known when the commit may be past History's first loaded page.
  const openCommit = (sha: string, position: number | null = null, ref: string | null = null) => {
    remember({ selectedCommitSha: sha })
    setActiveTab('history')
    setCommitFocus((prev) => ({ position, ref, request: (prev?.request ?? 0) + 1 }))
  }

  // Its position, so History can load that far back.
  const openCommitAnywhere = (sha: string) =>
    void HistoryService.GetCommitPosition(repoPath ?? '', sha)
      .then((position) => openCommit(sha, position))
      .catch(() => openCommit(sha))

  const openRevision = (path: string, rev: string, ref: string | null) =>
    HistoryService.ResolveCommit(path, rev).then((sha) => openCommit(sha, null, ref))

  const openWorkingFile = (path: string) => {
    remember({ selectedFilePath: path, diffScrollTop: 0 })
    setActiveTab('working-copy')
    setFileFocus((n) => n + 1)
  }

  const openChange = (number: number) => {
    setChangeFocus(number)
    setActiveTab('changes')
  }

  return {
    commitFocus,
    fileFocus,
    changeFocus,
    changeTab,
    openCommit,
    openCommitAnywhere,
    openRevision,
    openWorkingFile,
    openChange,
  }
}
