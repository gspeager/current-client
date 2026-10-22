import { System } from '@wailsio/runtime'
import { PlatformService } from '@current-client-bindings/app'
import PathText from '../git/PathText'
import { tildePath } from '../../lib/paths'
import { relativeTime } from '../../lib/relativeTime'
import { useCopyToClipboard } from '../../lib/useCopyToClipboard'
import { useAsyncData } from '../../lib/useAsyncData'
import { useBranchStatus } from '../../lib/useBranchStatus'
import { useHostIndicator } from '../../lib/useHostIndicator'
import './StatusBar.scss'

interface StatusBarProps {
  repoPath: string
  repoVersion: number
  gitVersion: string | null
  hostsVersion?: number
  hostIntegration?: boolean
  onOpenSettings?: () => void
}

function StatusBar({
  repoPath,
  repoVersion,
  gitVersion,
  hostsVersion = 0,
  hostIntegration = true,
  onOpenSettings,
}: StatusBarProps) {
  const status = useBranchStatus(repoPath, repoVersion)
  const integration = useHostIndicator(repoPath, hostsVersion, hostIntegration)
  const { data: appVersion } = useAsyncData(() => PlatformService.AppVersion(), [])
  const { data: home } = useAsyncData(() => (System.IsWindows() ? null : PlatformService.HomeDir()), [])
  const { copied, copy } = useCopyToClipboard()
  const branchLabel =
    status &&
    (status.detachedAt
      ? `HEAD detached at ${status.detachedAt}`
      : status.upstream
        ? `${status.current} → ${status.upstream}`
        : status.current)

  return (
    <footer className="status-bar">
      <span className="status-bar-ready">
        <span className="status-bar-dot" />
        Ready
      </span>
      {branchLabel && <span className="status-bar-branch">{branchLabel}</span>}
      <div className="status-bar-spacer" />
      {integration && (
        <button
          type="button"
          className={integration.status?.inFlight ? 'status-bar-host status-bar-host-busy' : 'status-bar-host'}
          onClick={onOpenSettings}
          title={`Integration is on for ${integration.host.hostname}. Opens Settings.`}
        >
          <span className="status-bar-host-dot" />
          <span className="status-bar-host-name">{integration.apiHost}</span>
          <span>{hostRequestLabel(integration.status)}</span>
        </button>
      )}
      <button
        type="button"
        className="status-bar-repo-path"
        onClick={() => copy(repoPath)}
        aria-label={`Copy ${repoPath}`}
        title={repoPath}
      >
        {copied ? 'Copied' : <PathText path={tildePath(repoPath, home ?? '')} />}
      </button>
      {gitVersion && <span>git {gitVersion}</span>}
      {appVersion && <span>Current Client {appVersion}</span>}
    </footer>
  )
}

function hostRequestLabel(status: { inFlight: boolean; lastRequest: string | null } | null): string {
  if (status?.inFlight) return 'request in flight'
  if (status?.lastRequest) return `last request ${relativeTime(new Date(status.lastRequest))}`
  return 'no requests yet'
}

export default StatusBar
