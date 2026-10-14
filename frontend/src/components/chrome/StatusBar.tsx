import { System } from '@wailsio/runtime'
import { PlatformService } from '@current-client-bindings/app'
import PathText from '../git/PathText'
import { tildePath } from '../../lib/paths'
import { useCopyToClipboard } from '../../lib/useCopyToClipboard'
import { useAsyncData } from '../../lib/useAsyncData'
import { useBranchStatus } from '../../lib/useBranchStatus'
import './StatusBar.scss'

interface StatusBarProps {
  repoPath: string
  repoVersion: number
  gitVersion: string | null
}

function StatusBar({ repoPath, repoVersion, gitVersion }: StatusBarProps) {
  const status = useBranchStatus(repoPath, repoVersion)
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

export default StatusBar
