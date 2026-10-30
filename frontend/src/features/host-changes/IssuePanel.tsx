import { X } from 'lucide-react'
import { useWindowKeydown } from '../../lib/useWindowKeydown'
import IssueDetail from './IssueDetail'

interface IssuePanelProps {
  repoPath: string
  issueKey: string
  onClose: () => void
  onBranchChanged?: () => void
}

// The issue a key opens, from History, a pull request or the sidebar: an
// inspector beside the view it was opened from rather than a dialog over it.
function IssuePanel({ repoPath, issueKey, onClose, onBranchChanged }: IssuePanelProps) {
  useWindowKeydown((e) => {
    if (e.key === 'Escape') onClose()
  })
  return (
    <aside className="issue-inspector">
      <IssueDetail
        repoPath={repoPath}
        issueKey={issueKey}
        onBranchChanged={onBranchChanged}
        headerAccessory={
          <button type="button" className="issue-inspector-close" onClick={onClose} aria-label="Close" title="Close">
            <X size={16} strokeWidth={1.75} />
          </button>
        }
      />
    </aside>
  )
}

export default IssuePanel
