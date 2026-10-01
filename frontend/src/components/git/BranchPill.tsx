import './BranchPill.scss'

interface BranchPillProps {
  name: string
  color: string
  current?: boolean
  bare?: boolean
  dirty?: boolean
}

// `bare` drops the pill shape when already inside a chip.
function BranchPill({ name, color, current, bare, dirty }: BranchPillProps) {
  const classes = ['branch-pill']
  if (current) classes.push('branch-pill-current')
  if (bare) classes.push('branch-pill-bare')

  return (
    <span className={classes.join(' ')}>
      <span className="branch-pill-dot" style={{ background: color }} />
      {name}
      {current && dirty && <span className="branch-pill-dirty" title="Uncommitted changes" />}
    </span>
  )
}

export default BranchPill
