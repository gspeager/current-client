import { useState, type ReactNode } from 'react'
import { Code2, ExternalLink, FolderOpen, History, Import, Split, Terminal, Undo2, type LucideIcon } from 'lucide-react'
import { PatchService, PlatformService, UndoService } from '@current-client-bindings/app'
import type { Links } from '@current-client-bindings/core/hostlink'
import { repositoryPage } from '../../lib/hostPages'
import BisectWizard from '../../features/history/BisectWizard'
import ReflogPanel from '../../features/history/ReflogPanel'
import { undoConfirmOptions } from '../../features/history/undoPreview'
import { errorMessage } from '../../lib/errors'
import { revealInFileManagerLabel } from '../../lib/fileManager'
import { useDialogs } from '../../lib/useDialogs'

export interface WorkspaceAction {
  id: string
  label: string
  icon: LucideIcon
  run: () => void
}

export interface WorkspaceFeedback {
  kind: 'error' | 'notice'
  text: string
}

interface WorkspaceActionsOptions {
  repoPath: string | null
  // The repository's host pages, when a remote points at one.
  hostLinks?: Links | null
  onBranchChanged: () => void
  onOpenCommit: (sha: string) => void
  onFailure: () => void
}

interface WorkspaceActions {
  actions: WorkspaceAction[]
  feedback: WorkspaceFeedback | null
  modals: ReactNode
}

// The Workspace actions the sidebar and the command palette both offer. One list,
// and one place for their feedback and windows, so the two can't drift apart.
export function useWorkspaceActions({
  repoPath,
  onBranchChanged,
  onOpenCommit,
  onFailure,
  hostLinks = null,
}: WorkspaceActionsOptions): WorkspaceActions {
  const { confirm } = useDialogs()
  const [feedback, setFeedback] = useState<(WorkspaceFeedback & { repoPath: string }) | null>(null)
  const [open, setOpen] = useState<'reflog' | 'bisect' | null>(null)

  if (!repoPath) return { actions: [], feedback: null, modals: null }

  const run = (action: Promise<unknown>, failure: string) => {
    setFeedback(null)
    action.catch((err: unknown) => {
      setFeedback({ repoPath, kind: 'error', text: `${failure}: ${errorMessage(err)}` })
      onFailure()
    })
  }

  const undoLast = async () => {
    const plan = await UndoService.Preview(repoPath)
    if (!(await confirm(undoConfirmOptions(plan)))) return
    await UndoService.Apply(repoPath, plan)
    onBranchChanged()
  }

  const importPatch = () =>
    PatchService.ImportPatch(repoPath)
      .then((notice) => {
        if (notice) setFeedback({ repoPath, kind: 'notice', text: notice })
      })
      .finally(onBranchChanged)

  const actions: WorkspaceAction[] = [
    {
      id: 'open-terminal',
      label: 'Open Terminal',
      icon: Terminal,
      run: () => run(PlatformService.OpenTerminal(repoPath), 'Could not open terminal'),
    },
    {
      id: 'reveal-file-manager',
      label: revealInFileManagerLabel(),
      icon: FolderOpen,
      run: () => run(PlatformService.RevealInFileManager(repoPath), 'Could not open file manager'),
    },
    {
      id: 'open-editor',
      label: 'Open in Editor',
      icon: Code2,
      run: () => run(PlatformService.OpenInEditor(repoPath), 'Could not open editor'),
    },
    ...(hostLinks?.project
      ? [
          {
            id: 'open-on-host',
            label: 'Open repository on host',
            icon: ExternalLink,
            run: () => run(PlatformService.OpenWebPage(repositoryPage(hostLinks)), 'Could not open'),
          },
        ]
      : []),
    { id: 'undo', label: 'Undo last operation', icon: Undo2, run: () => run(undoLast(), 'Could not undo') },
    { id: 'reflog', label: 'Reflog', icon: History, run: () => setOpen('reflog') },
    { id: 'bisect', label: 'Bisect', icon: Split, run: () => setOpen('bisect') },
    {
      id: 'import-patch',
      label: 'Import Patch…',
      icon: Import,
      run: () => run(importPatch(), 'Could not import patch'),
    },
  ]

  const modals = (
    <>
      {open === 'reflog' && (
        <ReflogPanel repoPath={repoPath} onClose={() => setOpen(null)} onBranchChanged={onBranchChanged} />
      )}
      {open === 'bisect' && (
        <BisectWizard
          repoPath={repoPath}
          onClose={() => setOpen(null)}
          onViewCommit={(sha) => {
            setOpen(null)
            onOpenCommit(sha)
          }}
        />
      )}
    </>
  )

  return { actions, feedback: feedback?.repoPath === repoPath ? feedback : null, modals }
}
