import { useState } from 'react'
import { HookService, type HookInfo, type HooksInfo } from '@current-client-bindings/app'
import Modal from '../../components/chrome/Modal'
import Switch from '../../components/forms/Switch'
import PathText from '../../components/git/PathText'
import { errorMessage } from '../../lib/errors'
import { baseName } from '../../lib/paths'
import { useAsyncData } from '../../lib/useAsyncData'
import { useDialogs } from '../../lib/useDialogs'
import './HooksPanel.scss'

interface HooksPanelProps {
  repoPath: string
  onClose: () => void
}

// Lists what Git runs and turns hooks off and on. It never runs one.
function HooksPanel({ repoPath, onClose }: HooksPanelProps) {
  const [version, setVersion] = useState(0)
  const { data: hooks, error: loadError } = useAsyncData(() => HookService.Hooks(repoPath), [repoPath], {
    refreshKey: version,
    keepData: true,
  })
  const { confirm } = useDialogs()
  const [error, setError] = useState<string | null>(null)

  const setOn = async (hook: HookInfo, on: boolean) => {
    if (hooks?.tracked && hook.path) {
      const confirmed = await confirm({
        title: on ? `Turn on ${hook.event}` : `Turn off ${hook.event}`,
        message: `${baseName(hooks.dir)} is tracked in the repository, so this is a change to commit.`,
        confirmLabel: on ? 'Turn on' : 'Turn off',
      })
      if (!confirmed) return
    }
    setError(null)
    HookService.SetHookOn(repoPath, hook, on)
      .catch((err: unknown) => setError(errorMessage(err)))
      .finally(() => setVersion((n) => n + 1))
  }

  return (
    <Modal title="Git hooks" onClose={onClose} className="hooks-panel">
      {(error ?? loadError) && <p className="hooks-error">{error ?? loadError}</p>}
      {hooks === null ? (
        <p className="hooks-hint">Loading hooks…</p>
      ) : (
        <div className="hooks-body">
          <div className="hooks-about">
            <PathText path={hooks.dir} className="hooks-dir" />
            {notesOf(hooks).map((note) => (
              <p key={note} className="hooks-hint">
                {note}
              </p>
            ))}
          </div>
          {hooks.hooks.length === 0 ? (
            <p className="hooks-hint">No hooks. Git runs none in this repository.</p>
          ) : (
            <ul className="hooks-list" aria-label="Hooks">
              {hooks.hooks.map((hook) => (
                <li key={`${hook.event}\0${hook.name}\0${hook.path}`} className="hooks-row">
                  <span className="hooks-row-event">{hook.event}</span>
                  <span className="hooks-row-source">{sourceOf(hook)}</span>
                  {hook.manager ? (
                    <span className="hooks-row-manager">Written by {hook.manager}</span>
                  ) : (
                    <Switch
                      checked={hook.on}
                      onChange={(on) => void setOn(hook, on)}
                      ariaLabel={`${hook.event}${hook.path ? '' : ` ${hook.name}`} on`}
                    />
                  )}
                </li>
              ))}
            </ul>
          )}
        </div>
      )}
    </Modal>
  )
}

function notesOf(hooks: HooksInfo): string[] {
  const notes: string[] = []
  if (hooks.scope === 'global' || hooks.scope === 'system') {
    notes.push(`Set by core.hooksPath in the ${hooks.scope} config, so every repository uses it.`)
  } else if (hooks.scope) {
    notes.push('Set by core.hooksPath in this repository’s config.')
  }
  if (hooks.tracked) notes.push('Tracked in the repository: turning a hook off or on is a change to commit.')
  if (hooks.worktrees > 1) notes.push(`Shared by ${hooks.worktrees} worktrees.`)
  if (hooks.configUnsupported) notes.push('Hooks set in config need Git 2.54 or newer, so they don’t run here.')
  return notes
}

function sourceOf(hook: HookInfo): string {
  if (hook.path) return baseName(hook.path) === hook.event ? '' : baseName(hook.path)
  return `${hook.name} · ${hook.scope} config · ${hook.command}`
}

export default HooksPanel
