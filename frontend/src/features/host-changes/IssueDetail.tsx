import { useState, type FormEvent, type ReactNode } from 'react'
import { CircleCheck, CircleDot, GitBranch, GitBranchPlus } from 'lucide-react'
import { BranchService, HostService, PlatformService } from '@current-client-bindings/app'
import HostMarkdown from '../../components/git/HostMarkdown'
import { issueBranchName } from '../../lib/branchName'
import { errorMessage, ignoreDecorativeFailure } from '../../lib/errors'
import { useNewIssue } from '../../lib/issuePanel'
import { plural } from '../../lib/plural'
import { relativeTime } from '../../lib/relativeTime'
import { useAsyncData } from '../../lib/useAsyncData'
import './IssuePanel.scss'
import { hostLogo, issueProvider } from '../../components/git/hostLogos'
import { useHostLinks } from '../../lib/useHostLinks'

interface IssueDetailProps {
  repoPath: string
  issueKey: string
  // A local branch named after the issue, when the caller knows one.
  branch?: string
  onBranchChanged?: () => void
  // Beside the state in the header, such as the inspector's close button.
  headerAccessory?: ReactNode
}

// An issue, fetched the first time it opens and kept until the next refresh:
// the Issues tab's pane and the inspector a key opens. Where the tracker
// takes writes, it takes a comment and closes or reopens; each is one
// request on a click, and the pane then reads the cache again.
function IssueDetail({ repoPath, issueKey, branch = '', onBranchChanged, headerAccessory }: IssueDetailProps) {
  const HostIcon = hostLogo(issueProvider(issueKey, useHostLinks(repoPath)))
  const { data, error, reload } = useAsyncData(
    () => HostService.IssueDetails(repoPath, issueKey),
    [repoPath, issueKey],
    { keepData: true },
  )
  // Present only with write scope, so it also says whether writes are offered.
  const canWrite = !!useNewIssue()
  const [comment, setComment] = useState('')
  const [running, setRunning] = useState<'comment' | 'state' | 'branch' | null>(null)
  const [actionError, setActionError] = useState<string | null>(null)
  const issue = data?.issue
  const writable = canWrite && !!data?.writable
  const closed = issue?.state === 'closed'

  const run = (kind: 'comment' | 'state' | 'branch', request: () => Promise<unknown>, after: () => void) => {
    setRunning(kind)
    setActionError(null)
    request()
      .then(after)
      .catch((err: unknown) => setActionError(errorMessage(err)))
      .finally(() => setRunning(null))
  }
  const submitComment = (e: FormEvent) => {
    e.preventDefault()
    if (comment.trim())
      run(
        'comment',
        () => HostService.CommentOnIssue(repoPath, issueKey, comment),
        () => {
          setComment('')
          reload()
        },
      )
  }
  const toggleState = () => run('state', () => HostService.SetIssueState(repoPath, issueKey, closed), reload)
  const switchToBranch = () => {
    if (!issue) return
    const name = branch || issueBranchName(issue.number || issue.key, issue.title)
    run(
      'branch',
      () =>
        branch
          ? BranchService.CheckoutBranch(repoPath, name)
          : BranchService.CreateBranch(repoPath, name).then(() => BranchService.CheckoutBranch(repoPath, name)),
      () => onBranchChanged?.(),
    )
  }

  return (
    <article className="issue-detail" aria-label={data?.reference ?? issueKey}>
      <header className="issue-detail-header">
        <span className="issue-detail-reference">{data?.reference ?? issueKey}</span>
        {issue?.state && <span className={`host-change-state host-change-state-${issue.state}`}>{issue.state}</span>}
        <span className="issue-detail-spacer" />
        {headerAccessory}
      </header>
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
              {branch && (
                <>
                  <dt>Branch</dt>
                  <dd className="issue-panel-branch">{branch}</dd>
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
              {data.comments.map((c, i) => (
                <article key={i} className="issue-panel-comment">
                  <p className="issue-panel-comment-head">
                    <span className="issue-panel-comment-author">{c.author.name || c.author.username}</span>
                    <span className="issue-panel-comment-time">{relativeTime(new Date(c.createdAt))}</span>
                  </p>
                  <HostMarkdown html={c.bodyHtml} className="issue-panel-comment-body" />
                </article>
              ))}
            </section>
          </>
        )}
      </div>
      {data && issue && (
        <form className="issue-detail-footer" onSubmit={submitComment}>
          {writable && (
            <textarea
              className="dialog-input issue-panel-reply-input"
              aria-label="Comment"
              placeholder="Comment"
              value={comment}
              onChange={(e) => setComment(e.target.value)}
              rows={3}
            />
          )}
          <div className="issue-detail-actions">
            {writable && (
              <button type="button" className="dialog-button" disabled={running !== null} onClick={toggleState}>
                {closed ? <CircleDot size={14} strokeWidth={1.75} /> : <CircleCheck size={14} strokeWidth={1.75} />}
                {running === 'state' ? (closed ? 'Reopening…' : 'Closing…') : closed ? 'Reopen issue' : 'Close issue'}
              </button>
            )}
            <button type="button" className="dialog-button" disabled={running !== null} onClick={switchToBranch}>
              {branch ? <GitBranch size={14} strokeWidth={1.75} /> : <GitBranchPlus size={14} strokeWidth={1.75} />}
              {branch ? `Check out ${branch}` : 'New branch from issue'}
            </button>
            <button
              type="button"
              className="dialog-button"
              onClick={() => PlatformService.OpenWebPage(issue.webUrl).catch(ignoreDecorativeFailure)}
              disabled={!issue.webUrl}
            >
              <HostIcon size={14} strokeWidth={1.75} />
              Open on host
            </button>
            {writable && (
              <button
                type="submit"
                className="dialog-button dialog-button-primary issue-detail-comment"
                disabled={!comment.trim() || running !== null}
              >
                {running === 'comment' ? 'Commenting…' : 'Comment'}
              </button>
            )}
          </div>
          {actionError && (
            <p className="dialog-error" role="alert">
              {actionError}
            </p>
          )}
        </form>
      )}
    </article>
  )
}

export default IssueDetail
