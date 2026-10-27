import { useRef, useState } from 'react'
import { PlatformService, UpdateService } from '@current-client-bindings/app'
import type { Result } from '@current-client-bindings/internal/update'
import { errorMessage, ignoreDecorativeFailure } from '../../lib/errors'
import { useAsyncData } from '../../lib/useAsyncData'
import SettingsRow from './SettingsRow'

// The app's version and the one request the app makes to a vendor: a check
// of github.com for a newer release, on this button alone. Nothing runs at
// start-up or in the background, nothing is downloaded, and the answer
// isn't kept between launches.
function UpdateSection() {
  const { data: version } = useAsyncData(() => PlatformService.AppVersion(), [])
  const { data: releasesUrl } = useAsyncData(() => UpdateService.ReleasesURL(), [])
  const [running, setRunning] = useState(false)
  const [result, setResult] = useState<Result | null>(null)
  const [error, setError] = useState<string | null>(null)
  const pending = useRef<{ cancel: () => void } | null>(null)

  const check = () => {
    setRunning(true)
    setResult(null)
    setError(null)
    const promise = UpdateService.Check()
    pending.current = promise
    promise
      .then((r) => setResult(r))
      .catch((err: unknown) => setError(errorMessage(err)))
      .finally(() => {
        pending.current = null
        setRunning(false)
      })
  }
  const cancel = () => pending.current?.cancel()
  const open = (url: string) => PlatformService.OpenWebPage(url).catch(ignoreDecorativeFailure)

  const outcome = result && describe(result)

  return (
    <SettingsRow
      label="Version"
      hint="Checks github.com for a newer release, only when clicked. Nothing runs on its own, and nothing is downloaded."
      error={error}
    >
      <div className="settings-actions settings-update">
        <span className="settings-value">{version ?? '…'}</span>
        {running ? (
          <>
            <span className="settings-hint">Checking github.com…</span>
            <button type="button" className="settings-button" onClick={cancel}>
              Cancel
            </button>
          </>
        ) : (
          <button type="button" className="settings-button" onClick={check}>
            Check for updates
          </button>
        )}
        {outcome && (
          <span className="settings-hint" role="status">
            {outcome}
          </span>
        )}
        {result?.state === 'newer' && (
          <button type="button" className="settings-button" onClick={() => open(result.latestUrl)}>
            Open releases page
          </button>
        )}
        {releasesUrl && (
          <span className="settings-hint settings-update-link">
            <code className="settings-code">{releasesUrl}</code>
            <button type="button" className="settings-button" onClick={() => open(releasesUrl)}>
              Open
            </button>
          </span>
        )}
      </div>
    </SettingsRow>
  )
}

function describe(result: Result): string {
  const released = result.releasedAt ? `, released ${result.releasedAt}` : ''
  switch (result.state) {
    case 'up-to-date':
      return `You’re up to date (${result.current}).`
    case 'newer':
      return `Version ${result.latest} is available${released}.`
    case 'ahead':
      return `You’re ahead of the latest release (${result.latest}).`
    case 'dev':
      return `A development build can’t be compared with the releases; the latest is ${result.latest}${released}.`
    default:
      return ''
  }
}

export default UpdateSection
