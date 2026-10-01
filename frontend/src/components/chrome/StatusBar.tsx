import { PlatformService } from '@current-client-bindings/app'
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
  const branchLabel = status && (status.upstream ? `${status.current} → ${status.upstream}` : status.current)

  return (
    <footer className="status-bar">
      <span className="status-bar-ready">
        <span className="status-bar-dot" />
        Ready
      </span>
      {branchLabel && <span className="status-bar-branch">{branchLabel}</span>}
      <div className="status-bar-spacer" />
      <span className="status-bar-repo-path">{repoPath}</span>
      {gitVersion && <span>git {gitVersion}</span>}
      {appVersion && <span>Current Client {appVersion}</span>}
    </footer>
  )
}

export default StatusBar
