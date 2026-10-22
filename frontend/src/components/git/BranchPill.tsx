import type { Links } from '@current-client-bindings/core/hostlink'
import { branchIssue } from '../../lib/hostReferences'
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
      ) : (
        name
      )}
      {current && dirty && <span className="branch-pill-dirty" title="Uncommitted changes" />}
    </span>
  )
}

export default BranchPill
