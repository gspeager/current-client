import { GitCommitVertical } from 'lucide-react'
import { useHeadCommit } from './useHeadCommit'
import './HeadCommitPanel.scss'

interface HeadCommitPanelProps {
  repoPath: string
  refreshKey: number
  onOpen: (sha: string) => void
}

// The fixed card at the top of the sidebar: where HEAD is, and a way to the commit.
function HeadCommitPanel({ repoPath, refreshKey, onOpen }: HeadCommitPanelProps) {
  const commit = useHeadCommit(repoPath, refreshKey)

  if (!commit) {
    return null
  }

  return (
    <button
      type="button"
      className="head-commit-panel"
      onClick={() => onOpen(commit.sha)}
      aria-label={`HEAD ${commit.sha.slice(0, 7)} ${commit.subject}`}
      title={commit.subject}
    >
      <GitCommitVertical size={16} strokeWidth={1.75} className="head-commit-panel-icon" />
      <span className="head-commit-panel-lines">
        <span className="head-commit-panel-line">
          <span className="head-commit-panel-label">HEAD</span>
          <span className="head-commit-panel-sha">{commit.sha.slice(0, 7)}</span>
        </span>
        <span className="head-commit-panel-subject">{commit.subject}</span>
      </span>
    </button>
  )
}

export default HeadCommitPanel
