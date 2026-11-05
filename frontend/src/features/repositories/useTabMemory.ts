import { useRef, useState } from 'react'
import type { AppTab } from './TabBar'

interface RepoMemory {
  selectedCommitSha: string | null
  historyTopSha: string | null
}

interface BranchMemory {
  selectedFilePath: string | null
  commitDraft: string
  diffScrollTop: number
}

type ViewMemory = RepoMemory & BranchMemory

const EMPTY_REPO_MEMORY: RepoMemory = { selectedCommitSha: null, historyTopSha: null }
const EMPTY_BRANCH_MEMORY: BranchMemory = { selectedFilePath: null, commitDraft: '', diffScrollTop: 0 }

const branchKey = (repoPath: string, branch: string) => `${repoPath}\0${branch}`

// Per-repo UI state that survives tab switches, and per-branch state that
// survives switching branches. View memory lives in refs: views write it on
// every keystroke without needing a re-render, and read it when they mount —
// which also happens on every repoVersion bump, so the read must see the latest
// write (an unsaved draft has to survive the remount a commit triggers), not a
// snapshot from an earlier render.
//
// branch is null while it's loading, and "" on a detached HEAD. A branch not
// seen yet starts from the one before it, so a draft follows into a branch
// created to commit it on, and stays on the branch it was written on too.
export function useTabMemory(repoPath: string | null, branch: string | null = '') {
  const [activeTabs, setActiveTabs] = useState<Record<string, AppTab>>({})
  const repoMemory = useRef(new Map<string, RepoMemory>())
  const branchMemory = useRef(new Map<string, BranchMemory>())
  const lastBranch = useRef(new Map<string, string>())

  const known = repoPath ? lastBranch.current.get(repoPath) : undefined
  if (repoPath && branch !== null && branch !== known) {
    const key = branchKey(repoPath, branch)
    const previous = branchMemory.current.get(branchKey(repoPath, known ?? ''))
    if (previous && !branchMemory.current.has(key)) branchMemory.current.set(key, { ...previous, diffScrollTop: 0 })
    lastBranch.current.set(repoPath, branch)
  }
  const currentBranch = branch ?? known ?? ''

  const activeTab: AppTab = (repoPath && activeTabs[repoPath]) || 'working-copy'
  const setActiveTab = (tab: AppTab) => {
    if (repoPath) setActiveTabs((prev) => ({ ...prev, [repoPath]: tab }))
  }

  const remember = (partial: Partial<ViewMemory>) => {
    if (!repoPath) return
    const { selectedCommitSha, historyTopSha, ...forBranch } = partial
    const forRepo = {
      ...(selectedCommitSha !== undefined && { selectedCommitSha }),
      ...(historyTopSha !== undefined && { historyTopSha }),
    }
    repoMemory.current.set(repoPath, { ...(repoMemory.current.get(repoPath) ?? EMPTY_REPO_MEMORY), ...forRepo })
    const key = branchKey(repoPath, currentBranch)
    branchMemory.current.set(key, { ...(branchMemory.current.get(key) ?? EMPTY_BRANCH_MEMORY), ...forBranch })
  }

  // A deleted branch's context goes with it, so a new branch of that name starts clean.
  const forgetBranchesExcept = (branches: string[]) => {
    if (!repoPath) return
    const prefix = branchKey(repoPath, '')
    for (const key of branchMemory.current.keys()) {
      if (!key.startsWith(prefix)) continue
      const name = key.slice(prefix.length)
      if (name !== '' && name !== currentBranch && !branches.includes(name)) branchMemory.current.delete(key)
    }
  }

  const restored: ViewMemory = repoPath
    ? {
        ...(repoMemory.current.get(repoPath) ?? EMPTY_REPO_MEMORY),
        ...(branchMemory.current.get(branchKey(repoPath, currentBranch)) ?? EMPTY_BRANCH_MEMORY),
      }
    : { ...EMPTY_REPO_MEMORY, ...EMPTY_BRANCH_MEMORY }

  return { activeTab, setActiveTab, restored, remember, memoryBranch: currentBranch, forgetBranchesExcept }
}
