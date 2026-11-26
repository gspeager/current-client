import type { ReactNode } from 'react'
import { Settings } from 'lucide-react'
import WaveMark from '../../components/chrome/WaveMark'
import GitMissing from './GitMissing'
import RepoSwitcher from './RepoSwitcher'
import type { useRepositoryLifecycle } from './useRepositoryLifecycle'
import './WelcomeScreen.scss'

interface WelcomeScreenProps {
  repo: ReturnType<typeof useRepositoryLifecycle>
  gitVersion: string | null
  gitError: string | null
  onCheckGitAgain: () => void
  defaultLocation: string
  onOpenSettings: () => void
  accessory?: ReactNode
  notice?: ReactNode
  children?: ReactNode
}

function WelcomeScreen({
  repo,
  gitVersion,
  gitError,
  onCheckGitAgain,
  defaultLocation,
  onOpenSettings,
  accessory,
  notice,
  children,
}: WelcomeScreenProps) {
  return (
    <div className="app-welcome">
      <div className="app-welcome-corner">
        {accessory}
        <button
          type="button"
          className="app-welcome-settings"
          onClick={onOpenSettings}
          aria-label="Settings"
          title="Settings"
        >
          <Settings size={18} strokeWidth={1.75} />
        </button>
      </div>
      <span className="app-welcome-mark">
        <WaveMark width={72} />
      </span>
      <h1>Current Client</h1>
      {notice}
      {gitError ? (
        <GitMissing error={gitError} onCheckAgain={onCheckGitAgain} onSetPath={onOpenSettings} />
      ) : gitVersion ? (
        <p className="app-welcome-status">git {gitVersion}</p>
      ) : (
        <p className="app-welcome-status">Detecting Git installation…</p>
      )}
      {!gitError && <RepoSwitcher repo={repo} defaultLocation={defaultLocation} onOpenSettings={onOpenSettings} />}
      {children}
    </div>
  )
}

export default WelcomeScreen
