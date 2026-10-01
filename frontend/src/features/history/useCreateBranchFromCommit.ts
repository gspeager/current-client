import { useState } from 'react'
import { BranchService } from '@current-client-bindings/app'
import { errorMessage } from '../../lib/errors'
import { useDialogs } from '../../lib/useDialogs'

export function useCreateBranchFromCommit(repoPath: string, onBranchChanged?: () => void) {
  const { prompt } = useDialogs()
  const [branchError, setBranchError] = useState<string | null>(null)

  const createBranchFrom = async (sha: string) => {
    const name = await prompt({
      title: 'New branch',
      label: `Branch name (from ${sha.slice(0, 7)})`,
      confirmLabel: 'Create',
    })
    if (!name) return
    setBranchError(null)
    BranchService.CreateBranchAt(repoPath, name, sha)
      .then(() => onBranchChanged?.())
      .catch((err: unknown) => setBranchError(errorMessage(err)))
  }

  return { createBranchFrom, branchError }
}
