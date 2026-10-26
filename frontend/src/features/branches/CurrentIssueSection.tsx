import { useMemo } from 'react'
import { ExternalLink, PanelRight } from 'lucide-react'
import { BranchService, PlatformService } from '@current-client-bindings/app'
import { ignoreDecorativeFailure } from '../../lib/errors'
import { branchIssue } from '../../lib/hostReferences'
import { useOpenIssue } from '../../lib/issuePanel'
import { useHostLinks } from '../../lib/useHostLinks'
import { useAsyncData } from '../../lib/useAsyncData'
import { useReferenceCards } from '../../lib/useReferenceCards'
import './CurrentChangeSection.scss'

interface CurrentIssueSectionProps {
  repoPath: string
  refreshKey: unknown
}

const NO_CHANGES = {}

// The issue the checked-out branch is named after, as its merge request
// shows above it: title and state from the same titles the hover cards use.
function CurrentIssueSection({ repoPath, refreshKey }: CurrentIssueSectionProps) {
  const { data: status } = useAsyncData(() => BranchService.CurrentBranchStatus(repoPath), [repoPath], {
    refreshKey,
    keepData: true,
  })
  const hostLinks = useHostLinks(repoPath)
  const issue = status?.current ? branchIssue(status.current, hostLinks) : null
  const text = issue ? `#${issue.text}` : ''
  const texts = useMemo(() => (text ? [text] : []), [text])
  const cards = useReferenceCards(repoPath, texts, NO_CHANGES)
  const openIssue = useOpenIssue()
  const card = text ? cards[text] : undefined
  if (!issue || !card?.known || !card.title) return null
  return (
    <section className="current-change" aria-label={`Issue for ${status?.current}`}>
      <div className="current-change-head">
        <span className="current-change-label">Issue</span>
        {card.state && <span className={`current-change-state current-change-state-${card.state}`}>{card.state}</span>}
      </div>
      <div className="current-change-title">
        <span className="current-change-target">{text}</span>
        <span className="current-change-title-text">{card.title}</span>
      </div>
      <div className="current-change-actions">
        {openIssue && (
          <button type="button" className="current-change-button" onClick={() => openIssue(issue.text)}>
            <PanelRight size={14} strokeWidth={1.75} />
            Details
          </button>
        )}
        <button
          type="button"
          className="current-change-button"
          onClick={() => PlatformService.OpenWebPage(issue.url).catch(ignoreDecorativeFailure)}
        >
          <ExternalLink size={14} strokeWidth={1.75} />
          Open on host
        </button>
      </div>
    </section>
  )
}

export default CurrentIssueSection
