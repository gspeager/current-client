import { GitCommitVertical } from 'lucide-react'
import { useHeadCommit } from './useHeadCommit'
import './HeadCommitPanel.scss'

interface HeadCommitPanelProps {
  repoPath: string
  refreshKey: number
  onOpen: (sha: string) => void
}

function HeadCommitPanel({ repoPath, refreshKey, onOpen }: HeadCommitPanelProps) {
  const commit = useHeadCommit(repoPath, refreshKey)

  if (!commit) {
    return null
  }

  return (
    <button type="button" className="head-commit-panel" onClick={() => onOpen(commit.sha)}>
      <div className="head-commit-panel-header">
        <span className="head-commit-panel-label-group">
          <GitCommitVertical size={12} strokeWidth={1.75} className="head-commit-panel-icon" />
          <span className="head-commit-panel-label">Head commit</span>
        </span>
        <span className="head-commit-panel-sha">{commit.sha.slice(0, 7)}</span>
      </div>
      <span className="head-commit-panel-subject">{commit.subject}</span>
    </button>
  )
}

export default HeadCommitPanel
