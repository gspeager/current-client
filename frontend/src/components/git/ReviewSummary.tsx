import { Check, CircleDot, CircleX } from 'lucide-react'
import { type Review, ReviewState } from '@current-client-bindings/core/hostapi'
import './ReviewSummary.scss'

const GROUP_LABELS: [ReviewState, string][] = [
  [ReviewState.ReviewChangesRequested, 'Changes requested'],
  [ReviewState.ReviewApproved, 'Approved'],
  [ReviewState.ReviewRequested, 'Requested'],
  [ReviewState.ReviewCommented, 'Commented'],
]

// Reviewers by state, verdicts first. A closed change drops the ones still
// requested, which no longer wait on anything.
export function reviewGroups(reviews: Review[] | null, open: boolean): { label: string; names: string[] }[] {
  return GROUP_LABELS.filter(([state]) => open || state !== ReviewState.ReviewRequested)
    .map(([state, label]) => ({
      label,
      names: (reviews ?? []).filter((r) => r.state === state).map((r) => r.reviewer.name || r.reviewer.username),
    }))
    .filter((group) => group.names.length > 0)
}

export function reviewRequestedOf(reviews: Review[] | null, username: string): boolean {
  return (
    !!username &&
    (reviews ?? []).some((r) => r.state === ReviewState.ReviewRequested && r.reviewer.username === username)
  )
}

// One line for a change card: a request for changes outweighs approvals,
// which outweigh reviews still waiting. A review waiting on you reads as a
// request; one waiting on others names them.
function ReviewSummary({ reviews, open, me = '' }: { reviews: Review[] | null; open: boolean; me?: string }) {
  const groups = reviewGroups(reviews, open)
  const count = (state: ReviewState) => (reviews ?? []).filter((r) => r.state === state).length
  const title = groups.map((g) => `${g.label}: ${g.names.join(', ')}`).join('\n')
  if (count(ReviewState.ReviewChangesRequested) > 0) {
    return (
      <span className="review-summary review-summary-changes" title={title}>
        <CircleX size={12} strokeWidth={1.75} />
        Changes requested
      </span>
    )
  }
  const approved = count(ReviewState.ReviewApproved)
  if (approved > 0) {
    return (
      <span className="review-summary review-summary-approved" title={title}>
        <Check size={12} strokeWidth={1.75} />
        {approved} approved
      </span>
    )
  }
  if (open && count(ReviewState.ReviewRequested) > 0) {
    const waitingOn = (reviews ?? [])
      .filter((r) => r.state === ReviewState.ReviewRequested)
      .map((r) => r.reviewer.name || r.reviewer.username)
    return (
      <span className="review-summary" title={title}>
        <CircleDot size={12} strokeWidth={1.75} />
        {reviewRequestedOf(reviews, me) || !me ? 'Review requested' : `Waiting on ${waitingOn.join(', ')}`}
      </span>
    )
  }
  return null
}

export default ReviewSummary
