import { FolderOpen, Package2, RefreshCw } from 'lucide-react'
import { SubmoduleService, type SubmoduleInfo } from '@current-client-bindings/app'
import { useCancellableOperation } from '../../lib/cancellableOperation'
import { joinPath } from '../../lib/paths'
import { useAsyncData } from '../../lib/useAsyncData'
import './SubmodulesPanel.scss'

interface SubmodulesPanelProps {
  repoPath: string
  refreshKey?: unknown
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
function SubmodulesPanel({ repoPath, refreshKey, onOpenRepository, onSubmodulesChanged }: SubmodulesPanelProps) {
  const { data: submodules, reload } = useAsyncData(() => SubmoduleService.List(repoPath), [repoPath], { refreshKey })
  const updateOp = useCancellableOperation()

  if (!submodules || submodules.length === 0) return null

  const update = (paths: string[]) =>
    void updateOp.run(
      (auth) => SubmoduleService.Update(repoPath, paths, auth),
      () => {
        reload()
        onSubmodulesChanged?.()
      },
    )

  return (
    <section className="nav-pane-section submodules-panel">
      <div className="submodules-panel-header">
        <span className="submodules-panel-label">Submodules</span>
        <button
          type="button"
          className="submodules-panel-update-all"
          onClick={updateOp.running ? updateOp.cancel : () => update([])}
          title="Initialise and update every submodule to the commit this repository records"
        >
          {updateOp.running ? 'Updating… (cancel)' : 'Update all'}
        </button>
      </div>
      {updateOp.error && <p className="submodules-panel-error">Could not update: {updateOp.error}</p>}
      <ul className="submodules-panel-list">
        {submodules.map((sm) => (
          <li key={sm.path} className="submodule-row" title={sm.path}>
            <Package2 size={14} strokeWidth={1.5} className="submodule-row-icon" />
            <span className="submodule-row-path">{sm.path}</span>
            <span className="submodule-row-commit">{sm.commit.slice(0, 7)}</span>
            {state(sm) && <span className="submodule-row-state">{state(sm)}</span>}
            <span className="submodule-row-actions">
              <button
                type="button"
                onClick={() => update([sm.path])}
                disabled={updateOp.running}
                aria-label={`Update submodule ${sm.path}`}
                title="Update"
              >
                <RefreshCw size={16} strokeWidth={1.75} />
              </button>
              {sm.initialized && (
                <button
                  type="button"
                  onClick={() => onOpenRepository(joinPath(repoPath, sm.path))}
                  aria-label={`Open submodule ${sm.path}`}
                  title="Open"
                >
                  <FolderOpen size={16} strokeWidth={1.75} />
                </button>
              )}
            </span>
          </li>
        ))}
      </ul>
    </section>
  )
}

export default SubmodulesPanel
