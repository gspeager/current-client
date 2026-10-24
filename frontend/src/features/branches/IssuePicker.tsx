import { useState } from 'react'
import { ChevronDown, CircleDot } from 'lucide-react'
import { HostService } from '@current-client-bindings/app'
import { issueBranchName } from '../../lib/branchName'
import { useAsyncData } from '../../lib/useAsyncData'
import './IssuePicker.scss'

interface IssuePickerProps {
  repoPath: string
  onPick: (branchName: string) => void
}

// The open issues assigned to the token's user, for naming a new branch
// after one. Absent while the host's integration is off; the host is asked
// the first time the list opens, then the cache answers until a refresh.
function IssuePicker({ repoPath, onPick }: IssuePickerProps) {
  const [open, setOpen] = useState(false)
  const { data: host } = useAsyncData(() => HostService.CurrentChange(repoPath), [repoPath])
  const { data: issues, error } = useAsyncData(
    () => (open ? HostService.AssignedIssues(repoPath) : null),
    [open, repoPath],
  )
  if (!host?.enabled) return null

  return (
    <div className="issue-picker">
      <button type="button" className="issue-picker-toggle" aria-expanded={open} onClick={() => setOpen((v) => !v)}>
        <CircleDot size={14} strokeWidth={1.75} />
        From an issue
        <ChevronDown size={14} strokeWidth={1.75} className="issue-picker-chevron" />
      </button>
      {open &&
        (error ? (
          <p className="issue-picker-note issue-picker-error">Could not load issues: {error}</p>
        ) : !issues ? (
          <p className="issue-picker-note">Loading…</p>
        ) : issues.length === 0 ? (
          <p className="issue-picker-note">No open issues assigned to you.</p>
        ) : (
          <ul className="issue-picker-list" aria-label="Assigned issues">
            {issues.map((issue) => (
              <li key={issue.number}>
                <button
                  type="button"
                  className="issue-picker-issue"
                  title={issue.title}
                  onClick={() => {
                    onPick(issueBranchName(issue.number, issue.title))
                    setOpen(false)
                  }}
                >
                  <span className="issue-picker-number">#{issue.number}</span>
                  <span className="issue-picker-title">{issue.title}</span>
                </button>
              </li>
            ))}
          </ul>
        ))}
    </div>
  )
}

export default IssuePicker
