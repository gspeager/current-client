import { useEffect, useRef, useState } from 'react'
import {
  ChevronDown,
  ChevronRight,
  Cloud,
  Code2,
  FolderOpen,
  GitBranch,
  History,
  Import,
  Package,
  PanelLeftClose,
  PanelLeftOpen,
  ShieldCheck,
  Sparkles,
  Split,
  Tag,
  Terminal,
  Undo2,
  Wrench,
  type LucideIcon,
} from 'lucide-react'
import { DashboardService, PatchService, PlatformService, UndoService } from '@current-client-bindings/app'
import BranchSidebar from '../../features/branches/BranchSidebar'
import HeadCommitPanel from '../../features/working-copy/HeadCommitPanel'
import RemotesPanel from '../../features/remotes/RemotesPanel'
import StashPanel from '../../features/branches/StashPanel'
import TagsPanel from '../../features/branches/TagsPanel'
import { errorMessage } from '../../lib/errors'
import { undoConfirmOptions } from '../../features/history/undoPreview'
import { useDialogs } from '../../lib/useDialogs'
import type { useWorkingTree } from '../../features/working-copy/useWorkingTree'
import BisectWizard from '../../features/history/BisectWizard'
import ReflogPanel from '../../features/history/ReflogPanel'
import './NavPane.scss'

type SectionId = 'maintenance' | 'branches' | 'stash' | 'tags' | 'remotes'

interface NavPaneProps {
  repoPath: string
  repoVersion: number
  workingTree: ReturnType<typeof useWorkingTree>
  onBranchChanged: () => void
  onFetched: () => void
  pruneOnFetch?: boolean
  onOpenHeadCommit: (sha: string) => void
  width: number
  collapsed: boolean
  onCollapsedChange: (collapsed: boolean) => void
}

// The rail jumps to these sections; they need the full width to show anything.
const SECTION_SHORTCUTS: { id: SectionId; label: string; icon: LucideIcon }[] = [
  { id: 'maintenance', label: 'Maintenance', icon: Wrench },
  { id: 'branches', label: 'Branches', icon: GitBranch },
  { id: 'stash', label: 'Stash', icon: Package },
  { id: 'tags', label: 'Tags', icon: Tag },
  { id: 'remotes', label: 'Remotes', icon: Cloud },
]

function NavPane({
  repoPath,
  repoVersion,
  workingTree,
  onBranchChanged,
  onFetched,
  pruneOnFetch,
  onOpenHeadCommit,
  width,
  collapsed,
  onCollapsedChange,
}: NavPaneProps) {
  const { confirm } = useDialogs()
  const [actionError, setActionError] = useState<string | null>(null)
  const [reflogOpen, setReflogOpen] = useState(false)
  const [bisectOpen, setBisectOpen] = useState(false)
  const [fsckBusy, setFsckBusy] = useState(false)
  const [fsckError, setFsckError] = useState<string | null>(null)
  const [fsckResult, setFsckResult] = useState<string[] | null>(null)
  const [gcBusy, setGcBusy] = useState(false)
  const [gcError, setGcError] = useState<string | null>(null)
  const [gcMessage, setGcMessage] = useState<string | null>(null)
  const [workspaceOpen, setWorkspaceOpen] = useState(true)
  const [maintenanceOpen, setMaintenanceOpen] = useState(false)
  const sectionRefs = useRef<Partial<Record<SectionId, HTMLElement | null>>>({})
  const pendingSection = useRef<SectionId | null>(null)

  // A section picked from the rail is scrolled to once the pane has expanded.
  useEffect(() => {
    if (collapsed || !pendingSection.current) return
    sectionRefs.current[pendingSection.current]?.scrollIntoView({ block: 'start' })
    pendingSection.current = null
  }, [collapsed])

  // A failure from the rail expands the pane, so its message is visible.
  const run = (action: Promise<unknown>, failure: string) => {
    setActionError(null)
    action.catch((err: unknown) => {
      setActionError(`${failure}: ${errorMessage(err)}`)
      setWorkspaceOpen(true)
      onCollapsedChange(false)
    })
  }

  const undoLast = async () => {
    const plan = await UndoService.Preview(repoPath)
    if (!(await confirm(undoConfirmOptions(plan)))) return
    await UndoService.Apply(repoPath, plan)
    onBranchChanged()
  }

  const workspaceActions: { label: string; icon: LucideIcon; onClick: () => void }[] = [
    {
      label: 'Open Terminal',
      icon: Terminal,
      onClick: () => run(PlatformService.OpenTerminal(repoPath), 'Could not open terminal'),
    },
    {
      label: 'Reveal in File Explorer',
      icon: FolderOpen,
      onClick: () => run(PlatformService.RevealInFileManager(repoPath), 'Could not open file manager'),
    },
    {
      label: 'Open in Editor',
      icon: Code2,
      onClick: () => run(PlatformService.OpenInEditor(repoPath), 'Could not open editor'),
    },
    {
      label: 'Undo last operation',
      icon: Undo2,
      onClick: () => run(undoLast(), 'Could not undo'),
    },
    { label: 'Reflog', icon: History, onClick: () => setReflogOpen(true) },
    { label: 'Bisect', icon: Split, onClick: () => setBisectOpen(true) },
    {
      label: 'Import Patch…',
      icon: Import,
      onClick: () => run(PatchService.ImportPatch(repoPath).finally(onBranchChanged), 'Could not import patch'),
    },
  ]

  const showSection = (id: SectionId) => {
    if (id === 'maintenance') setMaintenanceOpen(true)
    pendingSection.current = id
    onCollapsedChange(false)
  }

  const checkIntegrity = () => {
    setFsckError(null)
    setFsckResult(null)
    setFsckBusy(true)
    DashboardService.RunFsckCheck(repoPath)
      .then((issues) => setFsckResult(issues))
      .catch((err: unknown) => setFsckError(errorMessage(err)))
      .finally(() => setFsckBusy(false))
  }

  const runCleanup = () => {
    setGcError(null)
    setGcMessage(null)
    setGcBusy(true)
    DashboardService.RunGC(repoPath)
      .then(() => {
        setGcMessage('Cleanup complete.')
        onBranchChanged()
      })
      .catch((err: unknown) => setGcError(errorMessage(err)))
      .finally(() => setGcBusy(false))
  }

  const sectionRef = (id: SectionId) => (el: HTMLElement | null) => {
    sectionRefs.current[id] = el
  }

  const modals = (
    <>
      {reflogOpen && (
        <ReflogPanel repoPath={repoPath} onClose={() => setReflogOpen(false)} onBranchChanged={onBranchChanged} />
      )}
      {bisectOpen && (
        <BisectWizard
          repoPath={repoPath}
          onClose={() => setBisectOpen(false)}
          onViewCommit={(sha) => {
            setBisectOpen(false)
            onOpenHeadCommit(sha)
          }}
        />
      )}
    </>
  )

  if (collapsed) {
    return (
      <nav className="nav-pane nav-pane-collapsed" aria-label="Repository">
        <button
          type="button"
          className="nav-pane-rail-button"
          onClick={() => onCollapsedChange(false)}
          aria-label="Expand sidebar"
          title="Expand sidebar"
        >
          <PanelLeftOpen size={16} strokeWidth={1.75} />
        </button>
        <div className="nav-pane-rail-divider" />
        {workspaceActions.map(({ label, icon: ActionIcon, onClick }) => (
          <button
            key={label}
            type="button"
            className="nav-pane-rail-button"
            onClick={onClick}
            aria-label={label}
            title={label}
          >
            <ActionIcon size={16} strokeWidth={1.75} />
          </button>
        ))}
        <div className="nav-pane-rail-divider" />
        {SECTION_SHORTCUTS.map(({ id, label, icon: SectionIcon }) => (
          <button
            key={id}
            type="button"
            className="nav-pane-rail-button"
            onClick={() => showSection(id)}
            aria-label={`Show ${label}`}
            title={label}
          >
            <SectionIcon size={16} strokeWidth={1.75} />
          </button>
        ))}
        {modals}
      </nav>
    )
  }

  return (
    <nav className="nav-pane" style={{ width, minWidth: width }} aria-label="Repository">
      <div className="nav-pane-scroll">
        <section className="nav-pane-section">
          <div className="nav-pane-section-header">
            <button type="button" className="nav-pane-section-toggle" onClick={() => setWorkspaceOpen((open) => !open)}>
              {workspaceOpen ? (
                <ChevronDown size={14} strokeWidth={1.5} className="nav-pane-section-chevron" />
              ) : (
                <ChevronRight size={14} strokeWidth={1.5} className="nav-pane-section-chevron" />
              )}
              <span className="nav-pane-label">Workspace</span>
            </button>
            <button
              type="button"
              className="nav-pane-collapse"
              onClick={() => onCollapsedChange(true)}
              aria-label="Collapse sidebar"
              title="Collapse sidebar"
            >
              <PanelLeftClose size={16} strokeWidth={1.75} />
            </button>
          </div>
          {workspaceOpen &&
            workspaceActions.map(({ label, icon: ActionIcon, onClick }) => (
              <button key={label} type="button" className="nav-pane-row" onClick={onClick}>
                <ActionIcon size={16} strokeWidth={1.75} className="nav-pane-row-icon" />
                <span className="nav-pane-row-label">{label}</span>
              </button>
            ))}
          {actionError && <p className="nav-pane-error">{actionError}</p>}
        </section>

        <section className="nav-pane-section" ref={sectionRef('maintenance')}>
          <button type="button" className="nav-pane-section-toggle" onClick={() => setMaintenanceOpen((open) => !open)}>
            {maintenanceOpen ? (
              <ChevronDown size={14} strokeWidth={1.5} className="nav-pane-section-chevron" />
            ) : (
              <ChevronRight size={14} strokeWidth={1.5} className="nav-pane-section-chevron" />
            )}
            <span className="nav-pane-label">Maintenance</span>
          </button>
          {maintenanceOpen && (
            <>
              <button type="button" className="nav-pane-row" onClick={checkIntegrity} disabled={fsckBusy}>
                <ShieldCheck size={16} strokeWidth={1.75} className="nav-pane-row-icon" />
                <span className="nav-pane-row-label">Check integrity</span>
              </button>
              {fsckError && <p className="nav-pane-error">Could not check integrity: {fsckError}</p>}
              {fsckResult !== null &&
                (fsckResult.length === 0 ? (
                  <p className="nav-pane-success">Repository is healthy.</p>
                ) : (
                  <div className="nav-pane-fsck-issues">
                    {fsckResult.map((issue, i) => (
                      <p key={i}>{issue}</p>
                    ))}
                  </div>
                ))}
              <button type="button" className="nav-pane-row" onClick={runCleanup} disabled={gcBusy}>
                <Sparkles size={16} strokeWidth={1.75} className="nav-pane-row-icon" />
                <span className="nav-pane-row-label">Run cleanup</span>
              </button>
              {gcError && <p className="nav-pane-error">Could not run cleanup: {gcError}</p>}
              {gcMessage && <p className="nav-pane-success">{gcMessage}</p>}
            </>
          )}
        </section>

        <section className="nav-pane-section" ref={sectionRef('branches')}>
          <BranchSidebar
            repoPath={repoPath}
            dirty={(workingTree.files?.length ?? 0) > 0}
            refreshKey={repoVersion}
            onBranchChanged={onBranchChanged}
          />
        </section>

        <section className="nav-pane-section" ref={sectionRef('stash')}>
          <StashPanel
            repoPath={repoPath}
            dirty={(workingTree.files?.length ?? 0) > 0}
            onStashChanged={onBranchChanged}
          />
        </section>

        <section className="nav-pane-section" ref={sectionRef('tags')}>
          <TagsPanel repoPath={repoPath} onTagChanged={onBranchChanged} />
        </section>

        <section className="nav-pane-section" ref={sectionRef('remotes')}>
          <RemotesPanel repoPath={repoPath} onFetched={onFetched} pruneOnFetch={pruneOnFetch} />
        </section>
      </div>

      <HeadCommitPanel repoPath={repoPath} refreshKey={repoVersion} onOpen={onOpenHeadCommit} />
      {modals}
    </nav>
  )
}

export default NavPane
