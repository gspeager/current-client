import { useRef, useState } from 'react'
import type { AppTab } from './TabBar'

interface ViewMemory {
  selectedFilePath: string | null
  selectedCommitSha: string | null
  commitDraft: string
  historyTopSha: string | null
}

const EMPTY_MEMORY: ViewMemory = {
  selectedFilePath: null,
  selectedCommitSha: null,
  commitDraft: '',
  historyTopSha: null,
}

// Per-repo UI state that survives tab switches. View memory lives in a ref:
// views write it on every keystroke without needing a re-render, and read it
// when they mount — which also happens on every repoVersion bump, so the read
// must see the latest write (an unsaved draft has to survive the remount a
// commit triggers), not a snapshot from an earlier render.
export function useTabMemory(repoPath: string | null) {
  const [activeTabs, setActiveTabs] = useState<Record<string, AppTab>>({})
  const memory = useRef(new Map<string, ViewMemory>())

  const activeTab: AppTab = (repoPath && activeTabs[repoPath]) || 'activity'
  const setActiveTab = (tab: AppTab) => {
    if (repoPath) setActiveTabs((prev) => ({ ...prev, [repoPath]: tab }))
  }

  const remember = (partial: Partial<ViewMemory>) => {
    if (!repoPath) return
    memory.current.set(repoPath, { ...(memory.current.get(repoPath) ?? EMPTY_MEMORY), ...partial })
  }

  const restored = (repoPath && memory.current.get(repoPath)) || EMPTY_MEMORY

  return { activeTab, setActiveTab, restored, remember }
}
