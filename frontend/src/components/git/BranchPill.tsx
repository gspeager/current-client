import type { Links } from '@current-client-bindings/core/hostlink'
import { branchIssue, branchTrackerKey } from '../../lib/hostReferences'
import { useTracker } from '../../lib/issuePanel'
import { ReferenceLink } from './ReferenceText'
import type { ReferenceCards } from '../../lib/useReferenceCards'
import './BranchPill.scss'

interface BranchPillProps {
  name: string
  color: string
  current?: boolean
  bare?: boolean
  dirty?: boolean
  hostLinks?: Links | null
  referenceCards?: ReferenceCards
}

// `bare` drops the pill shape when already inside a chip.
function BranchPill({ name, color, current, bare, dirty, hostLinks = null, referenceCards }: BranchPillProps) {
  const issue = branchIssue(name, hostLinks)
  const tracker = useTracker()
  const key = issue ? null : branchTrackerKey(name, tracker)
  const classes = ['branch-pill']
  if (current) classes.push('branch-pill-current')
  if (bare) classes.push('branch-pill-bare')

  return (
    <span className={classes.join(' ')}>
      <span className="branch-pill-dot" style={{ background: color }} />
      {issue ? (
        <>
          <ReferenceLink reference={issue} card={referenceCards?.[`#${issue.text}`]} />
          {name.slice(issue.text.length)}
        </>
      ) : key ? (
        <>
          {name.slice(0, key.start)}
          <ReferenceLink reference={key} />
          {name.slice(key.start + key.text.length)}
        </>
      ) : (
        name
      )}
      {current && dirty && <span className="branch-pill-dirty" title="Uncommitted changes" />}
    </span>
  )
}

export default BranchPill
