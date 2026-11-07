import {
  CircleCheck,
  CircleDot,
  GitMerge,
  GitPullRequest,
  GitPullRequestClosed,
  GitPullRequestDraft,
  type LucideIcon,
} from 'lucide-react'
import { capitalize } from '../../lib/text'

const CHANGE_ICONS: Record<string, LucideIcon> = {
  open: GitPullRequest,
  draft: GitPullRequestDraft,
  merged: GitMerge,
  closed: GitPullRequestClosed,
}
const ISSUE_ICONS: Record<string, LucideIcon> = { open: CircleDot, closed: CircleCheck }

// A state the host reports, set apart from labels people add by its icon and square outline.
function ChangeStateLabel({ state, issue = false }: { state: string; issue?: boolean }) {
  const Icon = (issue ? ISSUE_ICONS : CHANGE_ICONS)[state]
  return (
    <span className={`host-change-state host-change-state-${state}`}>
      {Icon && <Icon size={12} strokeWidth={1.75} aria-hidden="true" />}
      {capitalize(state)}
    </span>
  )
}

export default ChangeStateLabel
