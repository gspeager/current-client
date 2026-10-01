import { useState } from 'react'
import { StatusService, type FileStatus } from '@current-client-bindings/app'
import { errorMessage } from '../../lib/errors'
import { useDialogs } from '../../lib/useDialogs'

export function useWorkingTreeActions(repoPath: string, loadStatus: () => void) {
  const { confirm } = useDialogs()
  const [actionError, setActionError] = useState<string | null>(null)

  const run = (action: Promise<unknown>) => {
    setActionError(null)
    action.then(loadStatus).catch((err: unknown) => setActionError(errorMessage(err)))
  }

  const discardFile = async (f: FileStatus) => {
    const untracked = f.indexStatus === '?'
    const confirmed = await confirm({
      title: untracked ? 'Delete file' : 'Discard changes',
      message: untracked
        ? `Delete ${f.path}? This cannot be undone.`
        : `Discard changes to ${f.path}? This cannot be undone.`,
      confirmLabel: untracked ? 'Delete' : 'Discard',
      destructive: true,
    })
    if (confirmed) run(StatusService.DiscardFile(repoPath, f.path))
  }

  const discardPaths = async (paths: string[]) => {
    const confirmed = await confirm({
      title: 'Discard changes',
      message: `Discard changes to ${paths.length} files? This cannot be undone.`,
      confirmLabel: 'Discard',
      destructive: true,
    })
    if (confirmed) run(StatusService.DiscardFiles(repoPath, paths))
  }

  return {
    actionError,
    run,
    stage: (f: FileStatus) => run(StatusService.StageFile(repoPath, f.path)),
    unstage: (f: FileStatus) => run(StatusService.UnstageFile(repoPath, f.path)),
    stagePaths: (paths: string[]) => run(StatusService.StageFiles(repoPath, paths)),
    unstagePaths: (paths: string[]) => run(StatusService.UnstageFiles(repoPath, paths)),
    addToGitignore: (pattern: string) => run(StatusService.AddToGitignore(repoPath, pattern)),
    discardFile,
    discardPaths,
  }
}

export type WorkingTreeActions = ReturnType<typeof useWorkingTreeActions>
