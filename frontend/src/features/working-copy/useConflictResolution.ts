import { ConflictService, PlatformService, StatusService, type FileStatus } from '@current-client-bindings/app'
import type { ContextMenuItem } from '../../components/controls/ContextMenu'
import { useAsyncData } from '../../lib/useAsyncData'
import { useDialogs } from '../../lib/useDialogs'

type Side = 'ours' | 'theirs'

// A side that deleted the file shows D in its half of the status: the index
// letter for ours, the working tree's for theirs.
function deletedBy(f: FileStatus, side: Side): boolean {
  return (side === 'ours' ? f.indexStatus : f.worktreeStatus) === 'D'
}

// Sides are named by branch: during a rebase Git's "ours" is the branch being
// rebased onto, which reads backwards.
export function useConflictResolution(repoPath: string, active: boolean, run: (action: Promise<unknown>) => void) {
  const { confirm } = useDialogs()
  const { data: sides } = useAsyncData(
    () => (active ? ConflictService.GetConflictSides(repoPath) : null),
    [repoPath, active],
  )
  const name = (side: Side) => sides?.[side] ?? side

  const keepLabel = (f: FileStatus, side: Side) =>
    deletedBy(f, side) ? `Keep it deleted, as in ${name(side)}` : `Keep ${name(side)}'s version`

  const keep = async (f: FileStatus, side: Side) => {
    const other = side === 'ours' ? 'theirs' : 'ours'
    const confirmed = await confirm({
      title: keepLabel(f, side),
      message: `${keepLabel(f, side)} of ${f.path}? Changes to it from ${name(other)} are discarded.`,
      confirmLabel: 'Keep',
      destructive: true,
    })
    if (confirmed) run(ConflictService.KeepSide(repoPath, f.path, side))
  }

  const openInEditor = (f: FileStatus) =>
    run(
      ConflictService.ConflictMarkerLine(repoPath, f.path).then((line) =>
        PlatformService.OpenFileInEditor(repoPath, f.path, Math.max(line, 1)),
      ),
    )

  const markResolved = async (f: FileStatus) => {
    const line = await ConflictService.ConflictMarkerLine(repoPath, f.path).catch(() => 0)
    if (line > 0) {
      const confirmed = await confirm({
        title: 'Mark resolved',
        message: `${f.path} still has conflict markers, from line ${line}. Mark it resolved anyway?`,
        confirmLabel: 'Mark resolved',
      })
      if (!confirmed) return
    }
    run(StatusService.StageFile(repoPath, f.path))
  }

  const menuItems = (f: FileStatus): ContextMenuItem[] => [
    { label: 'Mark resolved', onClick: () => void markResolved(f) },
    { label: keepLabel(f, 'ours'), onClick: () => void keep(f, 'ours') },
    { label: keepLabel(f, 'theirs'), onClick: () => void keep(f, 'theirs') },
    { label: 'Open in editor', onClick: () => openInEditor(f) },
  ]

  return { keepLabel, keep, openInEditor, markResolved, menuItems }
}

export type ConflictResolution = ReturnType<typeof useConflictResolution>
