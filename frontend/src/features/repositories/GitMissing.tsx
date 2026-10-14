import { useState } from 'react'
import { GitService, PlatformService } from '@current-client-bindings/app'
import { errorMessage } from '../../lib/errors'
import { useAsyncData } from '../../lib/useAsyncData'

interface GitMissingProps {
  error: string
  onCheckAgain: () => void
  onSetPath: () => void
}

// On a Mac without Apple's Command Line Tools, running git would open Apple's
// install window unasked, so the app offers that installer itself.
function GitMissing({ error, onCheckAgain, onSetPath }: GitMissingProps) {
  const { data: toolsMissing } = useAsyncData(() => GitService.CommandLineToolsMissing(), [], {
    refreshKey: error,
  })
  const [installError, setInstallError] = useState<string | null>(null)

  const install = () => {
    setInstallError(null)
    GitService.InstallCommandLineTools().catch((err: unknown) => setInstallError(errorMessage(err)))
  }

  return (
    <>
      <p className="app-welcome-status app-welcome-status-error">{error}</p>
      {toolsMissing && (
        <p className="app-welcome-status">Apple's installer downloads them, which takes a few minutes.</p>
      )}
      <div className="app-welcome-git-actions">
        {toolsMissing ? (
          <button type="button" className="repo-switcher-primary" onClick={install}>
            Install Command Line Tools
          </button>
        ) : (
          <>
            <button
              type="button"
              className="repo-switcher-secondary"
              onClick={() => void PlatformService.OpenGitDownloadPage()}
            >
              Download Git
            </button>
            <button type="button" className="repo-switcher-secondary" onClick={onSetPath}>
              Set git executable path…
            </button>
          </>
        )}
        <button type="button" className="repo-switcher-secondary" onClick={onCheckAgain}>
          Check again
        </button>
      </div>
      {installError && <p className="app-welcome-status app-welcome-status-error">{installError}</p>}
    </>
  )
}

export default GitMissing
