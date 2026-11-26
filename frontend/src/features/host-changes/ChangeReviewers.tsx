import { hostUserName, useNameIdentities } from '../../lib/useNameIdentities'
import IdentityBadge from '../../components/git/IdentityBadge'
import { ReviewState, type Review } from '@current-client-bindings/core/hostapi'

const REVIEW_STATE_LABELS: Partial<Record<ReviewState, string>> = {
  [ReviewState.ReviewApproved]: 'Approved',
  [ReviewState.ReviewChangesRequested]: 'Changes requested',
  [ReviewState.ReviewRequested]: 'Waiting',
  [ReviewState.ReviewCommented]: 'Commented',
}

function ChangeReviewers({ repoPath, reviews }: { repoPath: string; reviews: Review[] | null }) {
  const nameOf = (review: Review) => hostUserName(review.reviewer)
  const identities = useNameIdentities((reviews ?? []).map(nameOf), repoPath)
  return (
    <section className="host-change-reviewers" aria-label="Reviewers">
      <h4 className="host-change-overview-heading">Reviewers</h4>
      {!reviews?.length && <p className="host-changes-empty">None requested</p>}
      <ul>
        {reviews?.map((review) => {
          const name = nameOf(review)
          const identity = identities.get(name)
          return (
            <li key={name}>
              {identity && <IdentityBadge identity={identity} name={name} email="" size={20} />}
              <span className="host-change-reviewer-name">{identity?.displayName ?? name}</span>
              <span
                className={`host-change-reviewer-state host-change-reviewer-state-${review.state.replace(/_/g, '-')}`}
              >
                {REVIEW_STATE_LABELS[review.state]}
              </span>
            </li>
          )
        })}
      </ul>
    </section>
  )
}

export default ChangeReviewers
