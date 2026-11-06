import { useEffect, useRef, useState, type MouseEvent, type ReactNode } from 'react'
import {
  ChevronDown,
  ChevronRight,
  Cloud,
  FolderGit2,
  GitBranch,
  Package,
  PanelLeftClose,
  PanelLeftOpen,
  Tag,
  Wrench,
  X,
  type LucideIcon,
} from 'lucide-react'
import type { ConflictStateInfo } from '@current-client-bindings/app'
import ContextMenu, { menuAnchor, type ContextMenuState } from '../controls/ContextMenu'
import BranchSidebar from '../../features/branches/BranchSidebar'
import CurrentChangeSection from '../../features/branches/CurrentChangeSection'
import CurrentIssueSection from '../../features/branches/CurrentIssueSection'
import HeadCommitPanel from '../../features/working-copy/HeadCommitPanel'
import RemotesPanel from '../../features/remotes/RemotesPanel'
import StashPanel from '../../features/branches/StashPanel'
import TagsPanel from '../../features/branches/TagsPanel'
import WorktreesPanel from '../../features/worktrees/WorktreesPanel'
import SubmodulesPanel from '../../features/submodules/SubmodulesPanel'
import SidebarSection from './SidebarSection'
import { useSidebarSections } from '../../lib/useSidebarSections'
import { useRovingRows } from '../../lib/useRovingRows'
import type { useWorkingTree } from '../../features/working-copy/useWorkingTree'
import type { WorkspaceAction, WorkspaceFeedback, WorkspaceGroup } from './useWorkspaceActions'
import './NavPane.scss'

type SectionId = 'branches' | 'stash' | 'tags' | 'remotes' | 'worktrees'

// Something elsewhere in the app, such as the palette, asking a sidebar
// section to open one of its forms or dialogs; n makes each ask distinct.
export interface SidebarRequest {
  action:
    | 'new-branch'
    | 'clean-up'
    | 'branch-review'
    | 'tidy-up'
    | 'restack'
    | 'new-tag'
    | 'publish-release'
    | 'new-stash'
    | 'show-stash'
    | 'new-worktree'
    | 'new-remote'
  n: number
}

interface NavPaneProps {
  repoPath: string
  repoVersion: number
  workingTree: ReturnType<typeof useWorkingTree>
  workspaceActions: WorkspaceAction[]
  workspaceFeedback: WorkspaceFeedback | null
  onDismissFeedback: () => void
  // Bumped when a host list is refreshed, so the change row re-reads itself.
  hostVersion?: number
  onPushToCreate?: () => void
  conflictState?: ConflictStateInfo | null
  onBranchChanged: () => void
  onFetched: () => void
  pruneOnFetch?: boolean
  onOpenHeadCommit: (sha: string) => void
  // Opens a merge request / pull request in its tab.
  onOpenChange?: (number: number) => void
  onOpenFile?: (path: string) => void
  onCreateChange?: (branch: string) => void
  request?: SidebarRequest | null
  // Opens a repository (such as a worktree or submodule) as a tab.
  onOpenRepository: (path: string) => void
  width: number
  collapsed: boolean
  onCollapsedChange: (collapsed: boolean) => void
}

// The rail jumps to these sections; they need the full width to show anything.
const SECTION_SHORTCUTS: { id: SectionId; label: string; icon: LucideIcon }[] = [
  { id: 'branches', label: 'Branches', icon: GitBranch },
  { id: 'stash', label: 'Stash', icon: Package },
  { id: 'tags', label: 'Tags', icon: Tag },
  { id: 'remotes', label: 'Remotes', icon: Cloud },
  { id: 'worktrees', label: 'Worktrees', icon: FolderGit2 },
]

function RailButton({
  label,
  icon: Icon,
  onClick,
  disabled = false,
  reason,
}: {
  label: string
  icon: LucideIcon
  onClick: (e: MouseEvent) => void
  disabled?: boolean
  reason?: string
}) {
  return (
    <button
      type="button"
      className="nav-pane-rail-button"
      onClick={onClick}
      disabled={disabled}
      aria-label={label}
      title={reason ?? label}
    >
      <Icon size={16} strokeWidth={1.75} />
    </button>
  )
}

function ToolGroup({ label, actions, children }: { label: string; actions: WorkspaceAction[]; children?: ReactNode }) {
  return (
    <div className="nav-pane-tool-group">
      <span className="nav-pane-tool-caption">{label}</span>
      <div className="nav-pane-tool-buttons">
        {actions.map(({ id, short, label: full, icon: Icon, run, disabled, reason }) => (
          <button
            key={id}
            type="button"
            className="nav-pane-tool"
            onClick={run}
            disabled={disabled}
            title={reason ?? full}
          >
            <Icon size={14} strokeWidth={1.75} />
            {short}
          </button>
        ))}
        {children}
      </div>
    </div>
  )
}

const SIDEBAR_ROWS =
  '.sidebar-section-header, .branch-row, .tag-row, .stash-row, .remote-row, .worktree-row, .submodule-row'

function NavPane({
  repoPath,
  repoVersion,
  workingTree,
  workspaceActions,
  workspaceFeedback,
  onDismissFeedback,
  hostVersion = 0,
  onPushToCreate,
  conflictState = null,
  onBranchChanged,
  onFetched,
  pruneOnFetch,
  onOpenHeadCommit,
  onOpenRepository,
  width,
  collapsed,
  onCollapsedChange,
  onOpenChange,
  onOpenFile,
  onCreateChange,
  request = null,
}: NavPaneProps) {
  const sections = useSidebarSections(repoPath)
  const [maintenanceOpen, setMaintenanceOpen] = useState(false)
  const rovingRows = useRovingRows(SIDEBAR_ROWS)
  const [toolboxMenu, setToolboxMenu] = useState<ContextMenuState | null>(null)
  const sectionRefs = useRef<Partial<Record<SectionId, HTMLElement | null>>>({})
  const pendingSection = useRef<SectionId | null>(null)
  const dirty = (workingTree.files?.length ?? 0) > 0
  const group = (name: WorkspaceGroup) => workspaceActions.filter((a) => a.group === name)

  // A section picked from the rail is scrolled to once the pane has expanded.
  useEffect(() => {
    if (collapsed || !pendingSection.current) return
    sectionRefs.current[pendingSection.current]?.scrollIntoView({ block: 'start' })
    pendingSection.current = null
  }, [collapsed])

  const showSection = (id: SectionId) => {
    sections.open(id)
    pendingSection.current = id
    onCollapsedChange(false)
  }

  useEffect(() => {
    if (!request) return
    const owner: Partial<Record<SidebarRequest['action'], SectionId>> = {
      'new-branch': 'branches',
      'clean-up': 'branches',
      'new-stash': 'stash',
      'show-stash': 'stash',
      'new-worktree': 'worktrees',
      'new-remote': 'remotes',
    }
    showSection(owner[request.action] ?? 'tags')
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [request])

  const openToolboxMenu = (e: MouseEvent) => {
    const groups = (['open', 'tools', 'maintenance'] as const).map(group)
    setToolboxMenu({
      ...menuAnchor(e),
      x: e.currentTarget.getBoundingClientRect().right + 4,
      items: groups.flatMap((actions, g) =>
        actions.map((a, i) => ({
          label: a.label,
          onClick: a.run,
          disabled: a.disabled,
          separatorBefore: g > 0 && i === 0,
        })),
      ),
    })
  }

  const sectionRef = (id: SectionId) => (el: HTMLElement | null) => {
    sectionRefs.current[id] = el
  }

  if (collapsed) {
    return (
      <nav className="nav-pane nav-pane-collapsed" aria-label="Repository">
        <RailButton label="Expand sidebar" icon={PanelLeftOpen} onClick={() => onCollapsedChange(false)} />
        <div className="nav-pane-rail-divider" />
        {SECTION_SHORTCUTS.map(({ id, label, icon }) => (
          <RailButton key={id} label={`Show ${label}`} icon={icon} onClick={() => showSection(id)} />
        ))}
        <div className="nav-pane-rail-toolbox">
          <div className="nav-pane-rail-divider" />
          <RailButton label="Toolbox" icon={Wrench} onClick={openToolboxMenu} />
        </div>
        {toolboxMenu && <ContextMenu state={toolboxMenu} onClose={() => setToolboxMenu(null)} />}
      </nav>
    )
  }

  return (
    <nav className="nav-pane" data-pane="sidebar" style={{ width, minWidth: width }} aria-label="Repository">
      <div className="nav-pane-head">
        <HeadCommitPanel repoPath={repoPath} refreshKey={repoVersion} onOpen={onOpenHeadCommit} />
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

      <div className="nav-pane-scroll" ref={rovingRows}>
        <CurrentChangeSection
          repoPath={repoPath}
          refreshKey={`${repoVersion}:${hostVersion}`}
          onOpenChange={onOpenChange}
          onOpenFile={onOpenFile}
          onCreateChange={onCreateChange}
          onPushToCreate={onPushToCreate}
        />
        <CurrentIssueSection repoPath={repoPath} refreshKey={`${repoVersion}:${hostVersion}`} />

        <section className="nav-pane-section" ref={sectionRef('branches')}>
          <BranchSidebar
            repoPath={repoPath}
            dirty={dirty}
            refreshKey={repoVersion}
            sections={sections}
            conflict={conflictState}
            onBranchChanged={onBranchChanged}
            onCreateChange={onCreateChange}
            request={request}
          />
        </section>

        <section className="nav-pane-section" ref={sectionRef('stash')}>
          <StashPanel
            repoPath={repoPath}
            dirty={dirty}
            refreshKey={repoVersion}
            sections={sections}
            onStashChanged={onBranchChanged}
            request={request}
          />
        </section>

        <section className="nav-pane-section" ref={sectionRef('tags')}>
          <TagsPanel repoPath={repoPath} sections={sections} onTagChanged={onBranchChanged} request={request} />
        </section>

        <section className="nav-pane-section" ref={sectionRef('remotes')}>
          <RemotesPanel
            repoPath={repoPath}
            sections={sections}
            onFetched={onFetched}
            pruneOnFetch={pruneOnFetch}
            request={request}
          />
        </section>

        <section className="nav-pane-section" ref={sectionRef('worktrees')}>
          <WorktreesPanel
            repoPath={repoPath}
            refreshKey={repoVersion}
            sections={sections}
            onOpenRepository={onOpenRepository}
            onWorktreesChanged={onBranchChanged}
            request={request}
          />
        </section>
        <SubmodulesPanel
          repoPath={repoPath}
          refreshKey={repoVersion}
          sections={sections}
          onOpenRepository={onOpenRepository}
          onSubmodulesChanged={onBranchChanged}
        />
      </div>

      <div className="nav-pane-footer">
        {workspaceFeedback && (
          <p className={`nav-pane-feedback nav-pane-feedback-${workspaceFeedback.kind}`} role="status">
            <span>{workspaceFeedback.text}</span>
            <button type="button" onClick={onDismissFeedback} aria-label="Dismiss" title="Dismiss">
              <X size={14} strokeWidth={1.75} />
            </button>
          </p>
        )}
        <SidebarSection
          label="Toolbox"
          open={sections.isOpen('toolbox', false)}
          onToggle={() => sections.toggle('toolbox', false)}
        >
          <ToolGroup label="Open in" actions={group('open')} />
          <ToolGroup label="Tools" actions={group('tools')}>
            <button
              type="button"
              className="nav-pane-tool"
              onClick={() => setMaintenanceOpen((open) => !open)}
              aria-expanded={maintenanceOpen}
              title="Check integrity, run cleanup"
            >
              <Wrench size={14} strokeWidth={1.75} />
              Maintenance
              {maintenanceOpen ? (
                <ChevronDown size={12} strokeWidth={1.5} />
              ) : (
                <ChevronRight size={12} strokeWidth={1.5} />
              )}
            </button>
          </ToolGroup>
          {maintenanceOpen && <ToolGroup label="Maintenance" actions={group('maintenance')} />}
        </SidebarSection>
      </div>
    </nav>
  )
}

export default NavPane
