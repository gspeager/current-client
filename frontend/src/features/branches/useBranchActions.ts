import { useState } from 'react'
import { BranchService, GitFlowService, StackService, type BranchInfo } from '@current-client-bindings/app'
import type { DeletedBranch } from './CleanUpBranchesDialog'
import type { NewBranchKind } from './NewBranchForm'
import { toBranchName } from '../../lib/branchName'
import { errorMessage } from '../../lib/errors'
import { plural } from '../../lib/plural'
import { useAction } from '../../lib/useAction'
import { useDialogs } from '../../lib/useDialogs'

interface BranchActionsOptions {
  repoPath: string
  branches: BranchInfo[] | null
  currentBranch: string
  // Reloads the list alone, for a change that moves no ref the rest of the app shows.
  reloadBranches: () => void
  reloadStacks: () => void
  // Reloads the list and tells the app a ref moved.
  onBranchesChanged: () => void
}

// What the sidebar does to branches, with the failure, the squash notice and
// the undo for a delete they leave.
export function useBranchActions({
  repoPath,
  branches,
  currentBranch,
  reloadBranches,
  reloadStacks,
  onBranchesChanged,
}: BranchActionsOptions) {
  const { confirm, prompt } = useDialogs()
  const { error, setError, fail, run } = useAction()
  const [squashed, setSquashed] = useState<string | null>(null)
  const [deleted, setDeleted] = useState<DeletedBranch[] | null>(null)

  const checkout = (name: string) =>
    void run(() => BranchService.CheckoutBranch(repoPath, name).then(onBranchesChanged), `Could not check out ${name}`)

  const create = (kind: NewBranchKind, name: string, then: () => void) =>
    void run(() =>
      (kind === 'branch'
        ? BranchService.CreateBranch(repoPath, name).then(() => BranchService.CheckoutBranch(repoPath, name))
        : GitFlowService.StartBranch(repoPath, kind, name)
      ).then(() => {
        then()
        onBranchesChanged()
      }),
    )

  const stacksChanged = () => {
    reloadStacks()
    onBranchesChanged()
  }

  const newStacked = async (parent: string) => {
    const name = await prompt({
      title: `New branch on ${parent}`,
      label: 'New branch name',
      confirmLabel: 'Create',
      transform: toBranchName,
    })
    if (name) void run(() => StackService.NewLayer(repoPath, name, parent).then(stacksChanged))
  }

  const stackCurrentOn = (parent: string) =>
    void run(() => StackService.AddLayer(repoPath, currentBranch, parent).then(reloadStacks))

  const removeFromStack = (name: string) => void run(() => StackService.RemoveLayer(repoPath, name).then(reloadStacks))

  const rename = async (oldName: string) => {
    const newName = await prompt({
      title: 'Rename branch',
      label: 'New branch name',
      initialValue: oldName,
      confirmLabel: 'Rename',
      transform: toBranchName,
    })
    if (newName) void run(() => BranchService.RenameBranch(repoPath, oldName, newName).then(reloadBranches))
  }

  // mode is '' (fast-forward when possible), 'no-ff' or 'squash'.
  const merge = (name: string, mode = '') => {
    setSquashed(null)
    const merging = () =>
      BranchService.MergeBranch(repoPath, name, mode).then(() => {
        if (mode === 'squash') setSquashed(name)
      })
    // A failed merge leaves conflicts the banner must pick up.
    void run(merging).finally(onBranchesChanged)
  }

  const mergeDefault = async (name: string) => {
    const target = branches?.find((b) => b.name === name)
    if (target && target.behind > 0) {
      const confirmed = await confirm({
        title: `Merge ${name}`,
        message: `${name} is ${plural(target.behind, 'commit')} behind ${target.upstream}. Pull ${name} first to include the latest work, or merge it as it is.`,
        confirmLabel: 'Merge anyway',
      })
      if (!confirmed) return
    }
    merge(name)
  }

  const rebaseOnto = (name: string) =>
    void run(() => BranchService.RebaseOnto(repoPath, name)).finally(onBranchesChanged)

  const remove = (name: string, force: boolean) => {
    setError(null)
    setDeleted(null)
    BranchService.DeleteBranch(repoPath, name, force)
      .then((tip) => {
        setDeleted([{ name, tip }])
        reloadBranches()
      })
      .catch(async (err: unknown) => {
        if (!force && errorMessage(err) === 'This branch has unmerged changes.') {
          const confirmed = await confirm({
            title: 'Delete branch',
            message: `"${name}" has unmerged changes. Delete anyway? Its commits stay in Recover.`,
            confirmLabel: 'Delete',
            destructive: true,
          })
          if (confirmed) remove(name, true)
          return
        }
        fail(err)
      })
  }

  const unsetUpstream = (name: string) =>
    void run(() => BranchService.UnsetUpstream(repoPath, name).then(onBranchesChanged))

  const restore = (gone: DeletedBranch[]) => {
    setDeleted(null)
    void run(() =>
      Promise.all(
        gone.map((branch) =>
          BranchService.CreateBranchAt(repoPath, branch.name, branch.tip).catch((err: unknown) => {
            throw new Error(`Could not restore ${branch.name}: ${errorMessage(err)}`)
          }),
        ),
      ),
    ).finally(onBranchesChanged)
  }

  return {
    error,
    setError,
    squashed,
    deleted,
    setDeleted,
    checkout,
    create,
    stacksChanged,
    newStacked,
    stackCurrentOn,
    removeFromStack,
    rename,
    merge,
    mergeDefault,
    rebaseOnto,
    remove,
    unsetUpstream,
    restore,
  }
}
