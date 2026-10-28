import { FolderOpen, Package2, RefreshCw } from 'lucide-react'
import { RowActionButtons, useRowMenu, type RowAction } from '../../components/controls/RowActions'
import { SubmoduleService, type SubmoduleInfo } from '@current-client-bindings/app'
import { useCancellableOperation } from '../../lib/cancellableOperation'
import { joinPath } from '../../lib/paths'
import { useAsyncData } from '../../lib/useAsyncData'
import SidebarSection from '../../components/chrome/SidebarSection'
import type { SidebarSections } from '../../lib/useSidebarSections'
import './SubmodulesPanel.scss'

interface SubmodulesPanelProps {
  repoPath: string
  refreshKey?: unknown
  sections: SidebarSections
  onOpenRepository: (path: string) => void
  onSubmodulesChanged?: () => void
}

function state(sm: SubmoduleInfo): string | null {
  if (sm.conflicted) return 'conflicted'
  if (!sm.initialized) return 'not initialised'
  if (sm.commitChanged) return 'different commit'
  return null
}

// Renders nothing, not even its section, in a repository without submodules.
function SubmodulesPanel({
  repoPath,
  refreshKey,
  sections,
  onOpenRepository,
  onSubmodulesChanged,
}: SubmodulesPanelProps) {
  const { data: submodules, reload } = useAsyncData(() => SubmoduleService.List(repoPath), [repoPath], { refreshKey })
  const updateOp = useCancellableOperation()
  const rowMenu = useRowMenu()

  if (!submodules || submodules.length === 0) return null

  const update = (paths: string[]) =>
    void updateOp.run(
      (auth) => SubmoduleService.Update(repoPath, paths, auth),
      () => {
        reload()
        onSubmodulesChanged?.()
      },
    )

  const submoduleActions = (sm: SubmoduleInfo): RowAction[] => [
    {
      label: 'Update',
      ariaLabel: `Update submodule ${sm.path}`,
      icon: RefreshCw,
      run: () => update([sm.path]),
      disabled: updateOp.running,
    },
    ...(sm.initialized
      ? [
          {
            label: 'Open',
            ariaLabel: `Open submodule ${sm.path}`,
            icon: FolderOpen,
            run: () => onOpenRepository(joinPath(repoPath, sm.path)),
          },
        ]
      : []),
  ]

  return (
    <section className="nav-pane-section submodules-panel">
      <SidebarSection
        label="Submodules"
        count={submodules.length}
        open={sections.isOpen('submodules', true)}
        onToggle={() => sections.toggle('submodules', true)}
        actions={
          <button
            type="button"
            className="submodules-panel-update-all"
            onClick={updateOp.running ? updateOp.cancel : () => update([])}
            title="Initialise and update every submodule to the commit this repository records"
          >
            {updateOp.running ? 'Updating… (cancel)' : 'Update all'}
          </button>
        }
      >
        {updateOp.error && <p className="submodules-panel-error">Could not update: {updateOp.error}</p>}
        <ul className="submodules-panel-list">
          {submodules.map((sm) => (
            <li
              key={sm.path}
              className="submodule-row"
              title={sm.path}
              onContextMenu={rowMenu.open(submoduleActions(sm))}
            >
              <Package2 size={14} strokeWidth={1.5} className="submodule-row-icon" />
              <span className="submodule-row-path">{sm.path}</span>
              <span className="submodule-row-commit">{sm.commit.slice(0, 7)}</span>
              {state(sm) && <span className="submodule-row-state">{state(sm)}</span>}
              <RowActionButtons actions={submoduleActions(sm)} className="submodule-row-actions" />
            </li>
          ))}
        </ul>
      </SidebarSection>
      {rowMenu.menu}
    </section>
  )
}

export default SubmodulesPanel
