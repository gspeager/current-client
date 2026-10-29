import { useState, type ReactNode } from 'react'
import {
  Cherry,
  Copy,
  FileDown,
  GitBranchPlus,
  GitCommitVertical,
  RotateCcw,
  Undo2,
  type LucideIcon,
} from 'lucide-react'
import {
  BranchService,
  type CommitInfo,
  CommitService,
  PatchService,
  PlatformService,
} from '@current-client-bindings/app'
import type { Links } from '@current-client-bindings/core/hostlink'
import { toBranchName } from '../../lib/branchName'
import { errorMessage } from '../../lib/errors'
import { commitPage } from '../../lib/hostPages'
import { suggestedPatchFilename } from '../../lib/patchFilename'
import { useDialogs } from '../../lib/useDialogs'
import ResetDialog, { type ResetMode } from './ResetDialog'
import { hostLogo } from '../../components/git/hostLogos'

export interface CommitAction {
  id: string
  label: string
  icon: LucideIcon
  run: () => void
  destructive?: boolean
}

// The number of actions the inspector shows as buttons; the rest go under More.
export const PRIMARY_COMMIT_ACTIONS = 4

// The one ordered list of things to do with a commit, for the inspector and the
// row's menu alike. Refreshes on failure too, since a conflict leaves the repo
// mid-operation.
export function useCommitActions(repoPath: string, onChanged?: () => void, hostLinks: Links | null = null) {
  const { confirm, prompt } = useDialogs()
  const [error, setError] = useState<string | null>(null)
  const [resetTarget, setResetTarget] = useState<CommitInfo | null>(null)

  const run = (action: Promise<unknown>, failure: string, refreshOnFailure = false) => {
    setError(null)
    action
      .then(() => onChanged?.())
      .catch((err: unknown) => {
        setError(`${failure}: ${errorMessage(err)}`)
        if (refreshOnFailure) onChanged?.()
      })
  }

  const checkoutCommit = async (sha: string) => {
    const confirmed = await confirm({
      title: 'Check out commit',
      message: `Check out ${sha.slice(0, 7)}? HEAD will be detached: new commits won't be on any branch unless one is created from them. Switch to a branch to leave.`,
      confirmLabel: 'Check out',
    })
    if (confirmed) run(BranchService.CheckoutCommit(repoPath, sha), 'Could not check out')
  }

  const createBranchFrom = async (sha: string) => {
    const name = await prompt({
      title: 'New branch',
      label: `Branch name (from ${sha.slice(0, 7)})`,
      confirmLabel: 'Create',
      transform: toBranchName,
    })
    if (name) run(BranchService.CreateBranchAt(repoPath, name, sha), 'Could not create branch')
  }

  const exportPatch = (commit: CommitInfo) => {
    setError(null)
    PatchService.ExportCommit(repoPath, commit.sha, suggestedPatchFilename(commit.sha, commit.subject)).catch(
      (err: unknown) => setError(`Could not export patch: ${errorMessage(err)}`),
    )
  }

  const reset = (sha: string, mode: ResetMode) => {
    setResetTarget(null)
    run(CommitService.Reset(repoPath, sha, mode), 'Could not reset')
  }

  const actionsFor = (commit: CommitInfo): CommitAction[] => [
    { id: 'checkout', label: 'Check out', icon: GitCommitVertical, run: () => void checkoutCommit(commit.sha) },
    { id: 'branch', label: 'Branch here', icon: GitBranchPlus, run: () => void createBranchFrom(commit.sha) },
    {
      id: 'cherry-pick',
      label: 'Cherry-pick',
      icon: Cherry,
      run: () => run(CommitService.CherryPick(repoPath, commit.sha), 'Could not cherry-pick', true),
    },
    {
      id: 'revert',
      label: 'Revert',
      icon: RotateCcw,
      run: () => run(CommitService.Revert(repoPath, commit.sha), 'Could not revert', true),
    },
    { id: 'reset', label: 'Reset to here…', icon: Undo2, run: () => setResetTarget(commit) },
    { id: 'export-patch', label: 'Export patch…', icon: FileDown, run: () => exportPatch(commit) },
    { id: 'copy-sha', label: 'Copy SHA', icon: Copy, run: () => void navigator.clipboard.writeText(commit.sha) },
    ...(hostLinks?.commit
      ? [
          {
            id: 'open-on-host',
            label: 'Open on host',
            icon: hostLogo(hostLinks.style),
            run: () => void PlatformService.OpenWebPage(commitPage(hostLinks, commit.sha)),
          },
        ]
      : []),
  ]

  const overlays: ReactNode = resetTarget && (
    <ResetDialog
      repoPath={repoPath}
      commit={resetTarget}
      onClose={() => setResetTarget(null)}
      onReset={(mode) => reset(resetTarget.sha, mode)}
    />
  )

  return { actionsFor, error, overlays }
}
