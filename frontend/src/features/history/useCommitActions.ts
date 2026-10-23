import { useState } from 'react'
import { BranchService, CommitService } from '@current-client-bindings/app'
import { errorMessage } from '../../lib/errors'
import { useDialogs } from '../../lib/useDialogs'

// Refreshes on failure too, since a conflict leaves the repo mid-operation.
export function useCommitActions(repoPath: string, onChanged?: () => void) {
  const { confirm } = useDialogs()
  const [actionError, setActionError] = useState<string | null>(null)

  const apply = (operation: (repoPath: string, sha: string) => Promise<void>) => (sha: string) => {
    setActionError(null)
    operation(repoPath, sha)
      .catch((err: unknown) => setActionError(errorMessage(err)))
      .finally(() => onChanged?.())
  }
  const cherryPick = apply(CommitService.CherryPick)
  const revert = apply(CommitService.Revert)

  const checkoutCommit = async (sha: string) => {
    const confirmed = await confirm({
      title: 'Check out commit',
      message: `Check out ${sha.slice(0, 7)}? HEAD will be detached: new commits won't be on any branch unless one is created from them. Switch to a branch to leave.`,
      confirmLabel: 'Check out',
    })
    if (!confirmed) return
    setActionError(null)
    BranchService.CheckoutCommit(repoPath, sha)
      .then(() => onChanged?.())
      .catch((err: unknown) => setActionError(errorMessage(err)))
  }

  return { cherryPick, revert, checkoutCommit, actionError }
}
