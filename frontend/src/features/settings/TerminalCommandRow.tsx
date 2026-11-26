import { useState } from 'react'
import { PlatformService } from '@current-client-bindings/app'
import type { Launcher } from '@current-client-bindings/internal/platform'
import { System } from '@wailsio/runtime'
import PathText from '../../components/git/PathText'
import { useAsyncData } from '../../lib/useAsyncData'
import SettingsRow from './SettingsRow'
import { useAction } from '../../lib/useAction'
import './settings.scss'

// Writes nothing until Install is clicked, and says where first.
function TerminalCommandRow() {
  const { data: loaded } = useAsyncData(() => PlatformService.TerminalCommand(), [])
  const [changed, setChanged] = useState<Launcher | null>(null)
  const data = changed ?? loaded
  const { run: runAction, running: busy, error } = useAction()
  if (!data?.path) return null

  const run = (action: () => Promise<Launcher>) => {
    void runAction(() => action().then(setChanged))
  }

  return (
    <SettingsRow label="Terminal command" hint={hintOf(data)} error={error}>
      <PathText path={data.path} className="settings-path" />
      {!data.byPackage && (
        <div className="settings-actions">
          {data.installed ? (
            <button
              type="button"
              className="settings-button"
              disabled={busy}
              onClick={() => run(PlatformService.RemoveTerminalCommand)}
            >
              Remove
            </button>
          ) : (
            <button
              type="button"
              className="settings-button"
              disabled={busy}
              onClick={() => run(PlatformService.InstallTerminalCommand)}
            >
              Install
            </button>
          )}
        </div>
      )}
      {data.installed && data.pathLine && (
        <p className="settings-hint">
          Its folder isn’t on the PATH. Add this line to {data.startupFile}, then open a new terminal:{' '}
          <code className="settings-code">{data.pathLine}</code>
        </p>
      )}
    </SettingsRow>
  )
}

function hintOf(launcher: Launcher): string {
  const opens = 'current-client . opens the folder a terminal is in.'
  if (launcher.byPackage) return `Installed by the app’s package. ${opens}`
  if (launcher.installed) return System.IsWindows() ? `${opens} New terminals have it.` : opens
  return System.IsWindows()
    ? `Install writes it here and adds its folder to the PATH. Then ${opens}`
    : `Install writes it here. Then ${opens}`
}

export default TerminalCommandRow
