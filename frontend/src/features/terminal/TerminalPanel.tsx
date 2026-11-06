import { Plus, X } from 'lucide-react'
import ResizeHandle from '../../components/chrome/ResizeHandle'
import { useResizableWidth } from '../../lib/useResizableWidth'
import TerminalPane from './TerminalPane'
import type { useTerminals } from './useTerminals'
import './TerminalPanel.scss'

interface TerminalPanelProps {
  repoPath: string
  terminals: ReturnType<typeof useTerminals>
  focusRequest: number
  theme?: string
}

// Every repository's shells stay mounted, so they keep their output, and only
// the open repository's are shown.
function TerminalPanel({ repoPath, terminals, focusRequest, theme }: TerminalPanelProps) {
  const height = useResizableWidth('pane-height-terminal', 260, 120, 720, 'grow-left', 'y')
  const { current } = terminals

  return (
    <>
      {current.open && <ResizeHandle onDragStart={height.onDragStart} ariaLabel="Resize terminal" horizontal />}
      <section className="terminal-panel" aria-label="Terminal" style={{ height: height.width }} hidden={!current.open}>
        <header className="terminal-panel-header">
          <div className="terminal-panel-tabs" role="tablist" aria-label="Shells">
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
        <div className="terminal-panel-body">
          {Object.entries(terminals.byRepo).flatMap(([repo, state]) =>
            state.shells.map((shell) => {
              const shown = repo === repoPath && state.open && shell.id === state.active
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
