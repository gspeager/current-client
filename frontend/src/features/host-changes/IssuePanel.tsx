import { useState, type FormEvent } from 'react'
import { ExternalLink, Plus } from 'lucide-react'
import { HostService, PlatformService } from '@current-client-bindings/app'
import Modal from '../../components/chrome/Modal'
import HostMarkdown from '../../components/git/HostMarkdown'
import { errorMessage, ignoreDecorativeFailure } from '../../lib/errors'
import { useNewIssue } from '../../lib/issuePanel'
import { plural } from '../../lib/plural'
import { relativeTime } from '../../lib/relativeTime'
import { useAsyncData } from '../../lib/useAsyncData'
import './IssuePanel.scss'

interface IssuePanelProps {
  repoPath: string
  issueKey: string
  onClose: () => void
}

// An issue, fetched the first time it opens and kept until the next refresh.
// Where the tracker takes writes, it takes a comment and closes or reopens;
// each is one request on a click, and the panel then reads the cache again.
function IssuePanel({ repoPath, issueKey, onClose }: IssuePanelProps) {
  const { data, error, reload } = useAsyncData(
    () => HostService.IssueDetails(repoPath, issueKey),
    [repoPath, issueKey],
    { keepData: true },
  )
  const newIssue = useNewIssue()
  const [comment, setComment] = useState('')
  const [running, setRunning] = useState<'comment' | 'state' | null>(null)
  const [writeError, setWriteError] = useState<string | null>(null)
  const issue = data?.issue
  const write = (kind: 'comment' | 'state', request: () => Promise<unknown>) => {
    setRunning(kind)
    setWriteError(null)
    request()
      .then(() => {
        if (kind === 'comment') setComment('')
        reload()
      })
      .catch((err: unknown) => setWriteError(errorMessage(err)))
      .finally(() => setRunning(null))
  }
  const submitComment = (e: FormEvent) => {
    e.preventDefault()
    if (comment.trim()) write('comment', () => HostService.CommentOnIssue(repoPath, issueKey, comment))
  }
  const closed = issue?.state === 'closed'
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
            {newIssue && data.writable && (
              <form className="issue-panel-reply" onSubmit={submitComment}>
                <textarea
                  className="dialog-input issue-panel-reply-input"
                  aria-label="Comment"
                  value={comment}
                  onChange={(e) => setComment(e.target.value)}
                  rows={3}
                />
                <div className="issue-panel-reply-actions">
                  <button
                    type="button"
                    className="dialog-button"
                    disabled={running !== null}
                    onClick={() => write('state', () => HostService.SetIssueState(repoPath, issueKey, closed))}
                  >
                    {running === 'state'
                      ? closed
                        ? 'Reopening…'
                        : 'Closing…'
                      : closed
                        ? 'Reopen issue'
                        : 'Close issue'}
                  </button>
                  <button type="submit" className="dialog-button" disabled={!comment.trim() || running !== null}>
                    {running === 'comment' ? 'Commenting…' : 'Comment'}
                  </button>
                </div>
                {writeError && (
                  <p className="dialog-error" role="alert">
                    {writeError}
                  </p>
                )}
              </form>
            )}
            <div className="issue-panel-actions">
              {newIssue && (
                <button type="button" className="dialog-button" onClick={() => newIssue()}>
                  <Plus size={14} strokeWidth={1.75} />
                  New issue
                </button>
              )}
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
