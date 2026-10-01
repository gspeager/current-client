import { useState } from 'react'
import { PlatformService } from '@current-client-bindings/app'
import { errorMessage, recentErrorsNewestFirst } from '../../lib/errors'
import SettingsRow from './SettingsRow'

function DiagnosticsSection() {
  const [copied, setCopied] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const copy = () => {
    setError(null)
    setCopied(false)
    PlatformService.GetDiagnostics(recentErrorsNewestFirst())
      .then((report) => navigator.clipboard.writeText(report))
      .then(() => setCopied(true))
      .catch((err: unknown) => setError(errorMessage(err)))
  }

  return (
    <SettingsRow
      label="Diagnostics"
      hint="App and Git versions, OS, and the last 20 errors shown, for a bug report. Repository paths are left out. Nothing is sent anywhere."
      error={error && `Could not copy: ${error}`}
    >
      <div className="settings-actions">
        {copied && <span className="settings-hint">Copied.</span>}
        <button type="button" className="settings-button" onClick={copy}>
          Copy diagnostics
        </button>
      </div>
    </SettingsRow>
  )
}

export default DiagnosticsSection
