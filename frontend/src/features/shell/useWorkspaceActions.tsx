import { useState, type ReactNode } from 'react'
import {
  Code2,
  FolderOpen,
  History,
  Import,
  ShieldCheck,
  Sparkles,
  Split,
  SquareTerminal,
  Undo2,
  Webhook,
  type LucideIcon,
  LifeBuoy,
} from 'lucide-react'
import { DashboardService, PatchService, PlatformService, UndoService } from '@current-client-bindings/app'
import type { Links } from '@current-client-bindings/core/hostlink'
import { hostName, repositoryPage } from '../../lib/hostPages'
import HooksPanel from '../hooks/HooksPanel'
import BisectWizard from '../history/BisectWizard'
import RecoverPanel from '../history/RecoverPanel'
import ReflogPanel from '../history/ReflogPanel'
import { undoActionLabel, undoConfirmOptions } from '../history/undoPreview'
import type { ActionResult } from '../../lib/actionResult'
import { errorMessage } from '../../lib/errors'
import { fileManagerName, revealInFileManagerLabel } from '../../lib/fileManager'
import { operationNoun } from '../../lib/conflictWords'
import { baseName } from '../../lib/paths'
import { useDialogs } from '../../lib/useDialogs'
import { hostLogo } from '../../components/git/hostLogos'

export type WorkspaceGroup = 'open' | 'tools' | 'maintenance'

export interface WorkspaceAction {
  id: string
  // The full name, for the palette and tooltips.
  label: string
  // The word the sidebar row shows; absent for actions that aren't in the sidebar.
  short?: string
  group?: WorkspaceGroup
  icon: LucideIcon
  // Other words the palette finds it by.
  keywords?: string[]
  shortcut?: string
  run: () => void
  // Set while the action can't run, with the reason for its tooltip.
  disabled?: boolean
  reason?: string
}

export interface WorkspaceFeedback {
  kind: 'error' | 'notice'
  text: string
  // The failure's own message, for its Git output.
  failure?: string
}

interface WorkspaceActionsOptions {
  repoPath: string | null
  // The repository's host pages, when a remote points at one.
  hostLinks?: Links | null
  // The editor executable from Settings; empty means VS Code.
  editorPath?: string
  // The merge, rebase, cherry-pick or revert in progress; undo and bisect wait for it.
  operation?: string | null
  // Opens the embedded terminal; absent where it's turned off.
  onOpenTerminal?: () => void
  onBranchChanged: () => void
  onOpenCommit: (sha: string) => void
  onFailure: () => void
  onUndone?: (result: ActionResult) => void
}

interface WorkspaceActions {
  actions: WorkspaceAction[]
  feedback: WorkspaceFeedback | null
  dismissFeedback: () => void
  modals: ReactNode
}

function editorName(editorPath: string): string {
  if (!editorPath) return 'VS Code'
  return baseName(editorPath).replace(/\.(exe|app|cmd)$/i, '')
}

// The Workspace actions the sidebar and the command palette both offer. One list,
// and one place for their feedback and windows, so the two can't drift apart.
export function useWorkspaceActions({
  repoPath,
  onBranchChanged,
  onOpenCommit,
  onFailure,
  hostLinks = null,
  editorPath = '',
  operation = null,
  onOpenTerminal,
  onUndone,
}: WorkspaceActionsOptions): WorkspaceActions {
  const { confirm } = useDialogs()
  const [feedback, setFeedback] = useState<(WorkspaceFeedback & { repoPath: string }) | null>(null)
  const [open, setOpen] = useState<'reflog' | 'recover' | 'bisect' | 'hooks' | null>(null)
  const dismissFeedback = () => setFeedback(null)

  if (!repoPath) return { actions: [], feedback: null, dismissFeedback, modals: null }

  const notice = (text: string) => setFeedback({ repoPath, kind: 'notice', text })

  const run = (action: Promise<unknown>, failure: string) => {
    setFeedback(null)
    action.catch((err: unknown) => {
      const message = errorMessage(err)
      setFeedback({ repoPath, kind: 'error', text: `${failure}: ${message}`, failure: message })
      onFailure()
    })
  }

  const undoLast = async () => {
    const plan = await UndoService.Preview(repoPath)
    if (!(await confirm(undoConfirmOptions(plan)))) return
    await UndoService.Apply(repoPath, plan)
    onBranchChanged()
    onUndone?.({ text: undoActionLabel(plan).replace(/^Undo/, 'Undid') })
  }

  const importPatch = () =>
    PatchService.ImportPatch(repoPath)
      .then((text) => text && notice(text))
      .finally(onBranchChanged)

  const checkIntegrity = () =>
    DashboardService.RunFsckCheck(repoPath).then((issues) =>
      notice(issues.length === 0 ? 'Repository is healthy.' : issues.join('\n')),
    )

  const runCleanup = () =>
    DashboardService.RunGC(repoPath).then(() => {
      notice('Cleanup complete.')
      onBranchChanged()
    })

  const blocked = operation ? { disabled: true, reason: `Finish the ${operationNoun(operation)} first` } : {}

  const actions: WorkspaceAction[] = [
    ...(onOpenTerminal
      ? [
          {
            id: 'terminal-panel',
            label: 'Terminal panel',
            short: 'Terminal',
            group: 'open' as const,
            icon: SquareTerminal,
            keywords: ['shell', 'console', 'command line'],
            shortcut: 'ctrl+`',
            run: onOpenTerminal,
          },
        ]
      : []),
    {
      id: 'reveal-file-manager',
      label: revealInFileManagerLabel(),
      short: fileManagerName(),
      group: 'open',
      icon: FolderOpen,
      run: () => run(PlatformService.RevealInFileManager(repoPath), 'Could not open file manager'),
    },
    {
      id: 'open-editor',
      label: `Open in ${editorName(editorPath)}`,
      short: editorName(editorPath),
      group: 'open',
      icon: Code2,
      run: () => run(PlatformService.OpenInEditor(repoPath), 'Could not open editor'),
    },
    ...(hostLinks?.project
      ? [
          {
            id: 'open-on-host',
            label: `Open repository on ${hostName(hostLinks)}`,
            short: hostName(hostLinks),
            group: 'open' as const,
            icon: hostLogo(hostLinks.style),
            run: () => run(PlatformService.OpenWebPage(repositoryPage(hostLinks)), 'Could not open'),
          },
        ]
      : []),
    {
      id: 'undo',
      label: 'Undo last operation',
      short: 'Undo',
      group: 'tools',
      icon: Undo2,
      run: () => run(undoLast(), 'Could not undo'),
      ...blocked,
    },
    {
      id: 'recover',
      label: 'Recover discarded work',
      short: 'Recover',
      group: 'tools',
      icon: LifeBuoy,
      run: () => setOpen('recover'),
    },
    { id: 'reflog', label: 'Reflog', short: 'Reflog', group: 'tools', icon: History, run: () => setOpen('reflog') },
    {
      id: 'bisect',
      label: 'Bisect',
      short: 'Bisect',
      group: 'tools',
      icon: Split,
      run: () => setOpen('bisect'),
      ...blocked,
    },
    {
      id: 'import-patch',
      label: 'Apply patch…',
      short: 'Apply patch',
      group: 'tools',
      icon: Import,
      run: () => run(importPatch(), 'Could not import patch'),
    },
    {
      id: 'hooks',
      label: 'Git hooks',
      short: 'Hooks',
      group: 'tools',
      icon: Webhook,
      keywords: ['pre-commit', 'husky', 'hooksPath'],
      run: () => setOpen('hooks'),
    },
    {
      id: 'check-integrity',
      label: 'Check integrity',
      short: 'Check integrity',
      group: 'maintenance',
      icon: ShieldCheck,
      run: () => run(checkIntegrity(), 'Could not check integrity'),
    },
    {
      id: 'run-cleanup',
      label: 'Run cleanup',
      short: 'Run cleanup',
      group: 'maintenance',
      icon: Sparkles,
      run: () => run(runCleanup(), 'Could not run cleanup'),
    },
  ]

  const modals = (
    <>
      {open === 'recover' && (
        <RecoverPanel repoPath={repoPath} onClose={() => setOpen(null)} onBranchChanged={onBranchChanged} />
      )}
      {open === 'hooks' && <HooksPanel repoPath={repoPath} onClose={() => setOpen(null)} />}
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

  return { actions, feedback: feedback?.repoPath === repoPath ? feedback : null, dismissFeedback, modals }
}
