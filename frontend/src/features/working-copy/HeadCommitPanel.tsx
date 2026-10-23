import { useHeadCommit } from './useHeadCommit'
import './HeadCommitPanel.scss'

interface HeadCommitPanelProps {
  repoPath: string
  refreshKey: number
  onOpen: (sha: string) => void
}

// One fixed line at the top of the sidebar: where HEAD is.
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
      <span className="head-commit-panel-label">HEAD</span>
      <span className="head-commit-panel-sha">{commit.sha.slice(0, 7)}</span>
      <span className="head-commit-panel-subject">{commit.subject}</span>
    </button>
  )
}

export default HeadCommitPanel
