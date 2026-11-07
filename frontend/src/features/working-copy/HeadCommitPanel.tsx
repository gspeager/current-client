import { GitCommitVertical } from 'lucide-react'
import { useHeadCommit } from './useHeadCommit'
import { shortSha } from '../../lib/text'
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
      aria-label={`HEAD ${shortSha(commit.sha)} ${commit.subject}`}
      title={commit.subject}
    >
      <GitCommitVertical size={16} strokeWidth={1.75} className="head-commit-panel-icon" />
      <span className="head-commit-panel-sha">{shortSha(commit.sha)}</span>
      <span className="head-commit-panel-subject">{commit.subject}</span>
    </button>
  )
}

export default HeadCommitPanel
