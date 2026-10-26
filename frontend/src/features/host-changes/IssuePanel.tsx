import { ExternalLink } from 'lucide-react'
import { HostService, PlatformService } from '@current-client-bindings/app'
import Modal from '../../components/chrome/Modal'
import HostMarkdown from '../../components/git/HostMarkdown'
import { ignoreDecorativeFailure } from '../../lib/errors'
import { plural } from '../../lib/plural'
import { relativeTime } from '../../lib/relativeTime'
import { useAsyncData } from '../../lib/useAsyncData'
import './IssuePanel.scss'

interface IssuePanelProps {
  repoPath: string
  issueKey: string
  onClose: () => void
}

// An issue, read-only: fetched the first time it opens and kept until the
// next refresh. Replying and closing stay on the host.
function IssuePanel({ repoPath, issueKey, onClose }: IssuePanelProps) {
  const { data, error } = useAsyncData(() => HostService.IssueDetails(repoPath, issueKey), [repoPath, issueKey])
  const issue = data?.issue
  return (
    <Modal
      title={data?.reference ?? issueKey}
      onClose={onClose}
      className="issue-panel"
      headerContent={
        issue?.state && <span className={`host-change-state host-change-state-${issue.state}`}>{issue.state}</span>
      }
    >
      <div className="issue-panel-body">
        {error && (
          <p className="dialog-error" role="alert">
            Could not load the issue: {error}
          </p>
        )}
        {!data && !error && <p className="issue-panel-hint">Loading…</p>}
        {data && issue && (
          <>
            <h3 className="issue-panel-title">{issue.title}</h3>
            <dl className="issue-panel-facts">
              <dt>Author</dt>
              <dd>{issue.author.name || issue.author.username}</dd>
              <dt>Opened</dt>
              <dd>{relativeTime(new Date(issue.createdAt))}</dd>
              {issue.milestone && (
                <>
                  <dt>Milestone</dt>
                  <dd>{issue.milestone}</dd>
                </>
              )}
              {issue.labels.length > 0 && (
                <>
                  <dt>Labels</dt>
                  <dd className="issue-panel-labels">
                    {issue.labels.map((label) => (
                      <span key={label} className="issue-panel-label">
                        {label}
                      </span>
                    ))}
                  </dd>
                </>
              )}
            </dl>
            {data.descriptionHtml ? (
              <HostMarkdown html={data.descriptionHtml} className="issue-panel-description" />
            ) : (
              <p className="issue-panel-hint">No description.</p>
            )}
            <section className="issue-panel-comments" aria-label="Comments">
              <h4 className="issue-panel-comments-title">{plural(data.comments.length, 'comment')}</h4>
              {data.comments.map((comment, i) => (
                <article key={i} className="issue-panel-comment">
                  <p className="issue-panel-comment-head">
                    <span className="issue-panel-comment-author">{comment.author.name || comment.author.username}</span>
                    <span className="issue-panel-comment-time">{relativeTime(new Date(comment.createdAt))}</span>
                  </p>
                  <HostMarkdown html={comment.bodyHtml} className="issue-panel-comment-body" />
                </article>
              ))}
            </section>
            <div className="issue-panel-actions">
              <button
                type="button"
                className="dialog-button"
                onClick={() => PlatformService.OpenWebPage(issue.webUrl).catch(ignoreDecorativeFailure)}
                disabled={!issue.webUrl}
              >
                <ExternalLink size={14} strokeWidth={1.75} />
                Open on host
              </button>
            </div>
          </>
        )}
      </div>
    </Modal>
  )
}

export default IssuePanel
