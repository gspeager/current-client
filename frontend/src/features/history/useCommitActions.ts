import { useState } from 'react'
import { CommitService } from '@current-client-bindings/app'
import { errorMessage } from '../../lib/errors'

// Refreshes on failure too, since a conflict leaves the repo mid-operation.
export function useCommitActions(repoPath: string, onChanged?: () => void) {
  const [actionError, setActionError] = useState<string | null>(null)

  const cherryPick = (sha: string) => {
    setActionError(null)
    CommitService.CherryPick(repoPath, sha)
      .catch((err: unknown) => setActionError(errorMessage(err)))
      .finally(() => onChanged?.())
  }

  const revert = (sha: string) => {
    setActionError(null)
    CommitService.Revert(repoPath, sha)
      .catch((err: unknown) => setActionError(errorMessage(err)))
      .finally(() => onChanged?.())
  }

  return { cherryPick, revert, actionError }
}
