import type { ReactNode } from 'react'
import { Plus, X } from 'lucide-react'
import ResizeHandle from '../../components/chrome/ResizeHandle'
import SegmentedControl from '../../components/controls/SegmentedControl'
import { useResizableWidth } from '../../lib/useResizableWidth'
import TerminalPane from './TerminalPane'
import type { useTerminals } from './useTerminals'
import './TerminalPanel.scss'

interface TerminalPanelProps {
  repoPath: string
  terminals: ReturnType<typeof useTerminals>
  focusRequest: number
  theme?: string
  // The task runner, shown beside the shells.
  tasks?: ReactNode
}

// Every repository's shells stay mounted, so they keep their output, and only
// the open repository's are shown.
function TerminalPanel({ repoPath, terminals, focusRequest, theme, tasks }: TerminalPanelProps) {
  const height = useResizableWidth('pane-height-terminal', 260, 120, 720, 'grow-left', 'y')
  const { current } = terminals
  const view = current.view

  return (
    <>
      {current.open && <ResizeHandle onDragStart={height.onDragStart} ariaLabel="Resize terminal" horizontal />}
      <section className="terminal-panel" aria-label="Terminal" style={{ height: height.width }} hidden={!current.open}>
        <header className="terminal-panel-header">
          {tasks && (
            <SegmentedControl
              value={view}
              onChange={(next) => {
                terminals.setView(repoPath, next)
                if (next === 'shells' && current.shells.length === 0) terminals.newShell(repoPath)
              }}
              options={[
                { value: 'shells', label: 'Terminal' },
                { value: 'tasks', label: 'Tasks' },
              ]}
            />
          )}
          <div className="terminal-panel-tabs" role="tablist" aria-label="Shells" hidden={view !== 'shells'}>
            {current.shells.map((shell) => (
              <span
                key={shell.id}
                className={shell.id === current.active ? 'terminal-tab terminal-tab-active' : 'terminal-tab'}
              >
                <button
                  type="button"
                  role="tab"
                  aria-selected={shell.id === current.active}
                  onClick={() => terminals.select(repoPath, shell.id)}
                >
                  Shell {shell.number}
                </button>
                <button
                  type="button"
                  aria-label={`Close Shell ${shell.number}`}
                  title="Close"
                  onClick={() => void terminals.closeShell(repoPath, shell.id)}
                >
                  <X size={12} strokeWidth={1.75} />
                </button>
              </span>
            ))}
            <button
              type="button"
              className="terminal-panel-action"
              aria-label="New shell"
              title="New shell"
              onClick={() => terminals.newShell(repoPath)}
            >
              <Plus size={14} strokeWidth={1.75} />
            </button>
          </div>
          <button type="button" className="terminal-panel-hide" onClick={() => terminals.hide(repoPath)}>
            Hide
          </button>
        </header>
        {view === 'tasks' && <div className="terminal-panel-tasks">{tasks}</div>}
        <div className="terminal-panel-body" hidden={view !== 'shells'}>
          {Object.entries(terminals.byRepo).flatMap(([repo, state]) =>
            state.shells.map((shell) => {
              const shown = view === 'shells' && repo === repoPath && state.open && shell.id === state.active
              return (
                <TerminalPane
                  key={shell.id}
                  id={shell.id}
                  visible={shown}
                  focusRequest={focusRequest}
                  theme={theme}
                  onExit={() => terminals.removeShell(repo, shell.id)}
                />
              )
            }),
          )}
        </div>
      </section>
    </>
  )
}

export default TerminalPanel
