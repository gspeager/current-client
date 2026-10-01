import { useState } from 'react'
import { RotateCcw } from 'lucide-react'
import { SettingsService } from '@current-client-bindings/app'
import { errorMessage } from '../../lib/errors'
import { useAsyncData } from '../../lib/useAsyncData'
import SettingsRow from './SettingsRow'

function IdentityColorSettings() {
  const {
    data: overrides,
    error: loadError,
    reload,
  } = useAsyncData(() => SettingsService.GetIdentityColorOverrides(), [])
  const [removeError, setRemoveError] = useState<string | null>(null)

  const remove = (key: string) => {
    setRemoveError(null)
    SettingsService.RemoveIdentityColorOverride(key)
      .then(reload)
      .catch((err: unknown) => setRemoveError(errorMessage(err)))
  }

  const entries = overrides ? Object.entries(overrides) : []

  return (
    <SettingsRow
      label="Custom badge colors"
      hint={
        overrides === null && !loadError
          ? 'Loading…'
          : entries.length === 0
            ? "None set. An author's color can be changed from a commit's details in History."
            : 'Reset returns an author to their default color.'
      }
      error={loadError ?? removeError}
      stacked
    >
      {entries.length > 0 && (
        <ul className="settings-identity-list">
          {entries.map(([key, color]) => (
            <li key={key}>
              <span className="settings-identity-swatch" style={{ background: color }} />
              <span className="settings-identity-key">{key}</span>
              <button type="button" className="settings-button settings-button-small" onClick={() => remove(key)}>
                <RotateCcw size={12} strokeWidth={1.75} />
                Reset
              </button>
            </li>
          ))}
        </ul>
      )}
    </SettingsRow>
  )
}

export default IdentityColorSettings
