import { System } from '@wailsio/runtime'
import { PlatformService, type HostStatusInfo } from '@current-client-bindings/app'
import { baseName, tildePath } from '../../lib/paths'
import { relativeTime } from '../../lib/relativeTime'
import { useCopyToClipboard } from '../../lib/useCopyToClipboard'
import { useAsyncData } from '../../lib/useAsyncData'
import { useBranchStatus } from '../../lib/useBranchStatus'
import { useHostStatus, type HostIndicator } from '../../lib/useHostIndicator'
import { useTracker } from '../../lib/issuePanel'
import './StatusBar.scss'

interface StatusBarProps {
  repoPath: string
  repoVersion: number
  gitVersion: string | null
  integration?: HostIndicator | null
  lastFetchedAt?: Date | null
  onOpenSettings?: () => void
}

function StatusBar({
  repoPath,
  repoVersion,
  gitVersion,
  integration = null,
  lastFetchedAt = null,
  onOpenSettings,
}: StatusBarProps) {
  const status = useBranchStatus(repoPath, repoVersion)
  const { data: appVersion } = useAsyncData(() => PlatformService.AppVersion(), [])
  const { data: home } = useAsyncData(() => (System.IsWindows() ? null : PlatformService.HomeDir()), [])
  const { copied, copy } = useCopyToClipboard()
  const tracker = useTracker()
  const trackerHost = tracker?.reading ? tracker.hostName : null
  const trackerStatus = useHostStatus(trackerHost)
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
      <span className="status-bar-fetched">
        {lastFetchedAt ? `fetched ${relativeTime(lastFetchedAt)}` : 'not fetched yet'}
      </span>
      {integration && (
        <ServiceItem
          name={integration.apiHost}
          status={integration.status}
          about={`Integration is on for ${integration.host.hostname}. Its API is contacted on Refresh, when a list first opens, and when details need more.`}
          onOpenSettings={onOpenSettings}
        />
      )}
      {trackerHost && (
        <ServiceItem
          name={trackerHost}
          status={trackerStatus}
          about={`Integration is on for the issue tracker ${trackerHost}. Its API is contacted on Refresh and when an issue opens.`}
          onOpenSettings={onOpenSettings}
        />
      )}
      <button
        type="button"
        className="status-bar-repo-path"
        onClick={() => copy(repoPath)}
        aria-label={`Copy ${repoPath}`}
        title={tildePath(repoPath, home ?? '')}
      >
        {copied ? 'Copied' : baseName(repoPath)}
      </button>
      {gitVersion && <span>git {gitVersion}</span>}
      {appVersion && <span>Current Client {appVersion}</span>}
    </footer>
  )
}

// One service that is on: its name and a dot, filled while a request is in
// flight. The budget and when it was last asked are in the tooltip, so two
// services fit a narrow window.
function ServiceItem({
  name,
  status,
  about,
  onOpenSettings,
}: {
  name: string
  status: HostStatusInfo | null
  about: string
  onOpenSettings?: () => void
}) {
  return (
    <button
      type="button"
      className={status?.inFlight ? 'status-bar-host status-bar-host-busy' : 'status-bar-host'}
      onClick={onOpenSettings}
      aria-label={`${name}, integration on`}
      title={`${about}\n${hostRequestLabel(status)}\nOpens Settings.`}
    >
      <span className="status-bar-host-dot" />
      <span className="status-bar-host-name">{name}</span>
    </button>
  )
}

// The budget the host reported with its last answer, then when it was
// last asked.
function hostRequestLabel(status: HostStatusInfo | null): string {
  const parts: string[] = []
  if (status?.limit) {
    parts.push(`${status.remaining} of ${status.limit} requests left`)
    if (status.remaining === 0 && status.resetAt) {
      parts.push(`resets at ${new Date(status.resetAt).toLocaleTimeString([], { timeStyle: 'short' })}`)
    }
  }
  if (status?.inFlight) parts.push('request in flight')
  else if (status?.lastRequest) parts.push(`last request ${relativeTime(new Date(status.lastRequest))}`)
  else parts.push('no requests yet')
  return parts.join(' · ')
}

export default StatusBar
