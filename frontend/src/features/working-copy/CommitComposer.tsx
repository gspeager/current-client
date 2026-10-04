import { useEffect, useState, type KeyboardEvent } from 'react'
import { GitCommitVertical, UserPlus, X } from 'lucide-react'
import { CommitService, IdentityService, type AuthorInfo } from '@current-client-bindings/app'
import { COMMIT_TYPES, parsePrefix, withPrefix } from '../../lib/conventionalCommit'
import { errorMessage, ignoreDecorativeFailure } from '../../lib/errors'
import { useAsyncData } from '../../lib/useAsyncData'
import { useDialogs } from '../../lib/useDialogs'
import { useDismissibleDetails } from '../../lib/useDismissibleDetails'
import SplitButton from '../../components/controls/SplitButton'
import IdentityPrompt from './IdentityPrompt'
import './CommitComposer.scss'

interface CommitComposerProps {
  repoPath: string
  stagedCount: number
  onCommitted: () => void
  onPushRequested?: () => void
  initialMessage?: string
  onMessageChange?: (message: string) => void
  showTypePicker?: boolean
}

const SUBJECT_TARGET = 72

function splitMessage(raw: string): { subject: string; body: string } {
  const newlineIndex = raw.indexOf('\n')
  if (newlineIndex === -1) {
    return { subject: raw, body: '' }
  }
  return { subject: raw.slice(0, newlineIndex), body: raw.slice(newlineIndex + 1).replace(/^\n+/, '') }
}

function joinMessage(subject: string, body: string): string {
  return body.trim() === '' ? subject : `${subject}\n\n${body}`
}

function withCoAuthorTrailers(body: string, coAuthors: AuthorInfo[]): string {
  if (coAuthors.length === 0) return body
  const trailers = coAuthors.map((a) => `Co-authored-by: ${a.name} <${a.email}>`).join('\n')
  return body.trim() === '' ? trailers : `${body}\n\n${trailers}`
}

function CommitComposer({
  repoPath,
  stagedCount,
  onCommitted,
  onPushRequested,
  initialMessage = '',
  onMessageChange,
  showTypePicker = false,
}: CommitComposerProps) {
  const initialSplit = splitMessage(initialMessage)
  const [subject, setSubject] = useState(initialSplit.subject)
  const [body, setBody] = useState(initialSplit.body)
  const [amend, setAmend] = useState(false)
  const [allowEmpty, setAllowEmpty] = useState(false)
  const { confirm } = useDialogs()
  const [menuOpen, setMenuOpen] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [coAuthors, setCoAuthors] = useState<AuthorInfo[]>([])
  const [coAuthorMenuOpen, setCoAuthorMenuOpen] = useState(false)
  const coAuthorMenuRef = useDismissibleDetails()
  const [sign, setSign] = useState(false)

  const { data: currentUser, reload: reloadCurrentUser } = useAsyncData(
    () => IdentityService.GetCurrentUser(repoPath),
    [repoPath],
  )
  const prefix = parsePrefix(subject)
  const setPrefix = (type: string, scope: string) => setMessage(withPrefix(subject, type, scope), body)

  const needsIdentity = currentUser !== null && (currentUser.name === '' || currentUser.email === '')

  const recentAuthors =
    useAsyncData(() => CommitService.ListRecentAuthors(repoPath).catch(() => []), [repoPath]).data ?? []

  useEffect(() => {
    // Seeded once per repo so a re-render never silently un-signs.
    CommitService.GetGPGSignDefault(repoPath).then(setSign).catch(ignoreDecorativeFailure)
  }, [repoPath])

  // After a squash merge, start the message with the squashed commits, unless a draft is already there.
  useEffect(() => {
    if (initialMessage !== '') return
    CommitService.GetSquashedSubjects(repoPath)
      .then((subjects) => {
        if (subjects?.length) setMessage('', subjects.map((s) => `- ${s}`).join('\n'))
      })
      .catch(ignoreDecorativeFailure)
    // Checked once when the composer opens; the draft takes over from there.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [repoPath])

  const setMessage = (nextSubject: string, nextBody: string) => {
    setSubject(nextSubject)
    setBody(nextBody)
    onMessageChange?.(joinMessage(nextSubject, nextBody))
  }

  const toggleCoAuthor = (a: AuthorInfo) => {
    setCoAuthors((prev) =>
      prev.some((c) => c.email === a.email) ? prev.filter((c) => c.email !== a.email) : [...prev, a],
    )
  }

  const toggleAmend = (checked: boolean) => {
    setError(null)
    if (!checked) {
      setAmend(false)
      setMessage('', '')
      return
    }
    CommitService.GetLastCommitMessage(repoPath)
      .then((lastMessage) => {
        setAmend(true)
        const split = splitMessage(lastMessage)
        setMessage(split.subject, split.body)
      })
      .catch((err: unknown) => setError(`Could not amend: ${errorMessage(err)}`))
  }

  const commit = (andPush: boolean) => {
    setError(null)
    setMenuOpen(false)
    CommitService.Commit(repoPath, joinMessage(subject, withCoAuthorTrailers(body, coAuthors)), amend, allowEmpty, sign)
      .then(() => {
        setMessage('', '')
        setAmend(false)
        setAllowEmpty(false)
        setCoAuthors([])
        onCommitted()
        if (andPush) {
          onPushRequested?.()
        }
      })
      .catch((err: unknown) => setError(`Could not commit: ${errorMessage(err)}`))
  }

  const undoLastCommit = () => {
    setError(null)
    setMenuOpen(false)
    CommitService.IsLastCommitPushed(repoPath)
      .then(async (pushed) => {
        const confirmed =
          !pushed ||
          (await confirm({
            title: 'Undo pushed commit',
            message: 'The last commit has already been pushed. Undoing it rewrites published history.',
            confirmLabel: 'Undo commit',
            destructive: true,
          }))
        if (confirmed) return CommitService.UndoLastCommit(repoPath).then(onCommitted)
      })
      .catch((err: unknown) => setError(`Could not undo last commit: ${errorMessage(err)}`))
  }

  const blockReason = needsIdentity
    ? 'Set a name and email to commit.'
    : subject.trim() === ''
      ? 'Enter a commit message.'
      : !amend && !allowEmpty && stagedCount === 0
        ? 'Nothing is staged.'
        : null

  const onMessageKeyDown = (e: KeyboardEvent) => {
    if ((e.metaKey || e.ctrlKey) && e.key === 'Enter' && blockReason === null) {
      e.preventDefault()
      commit(false)
    }
  }

  return (
    <section className="commit-composer">
      <div className="commit-composer-header">
        <span className="commit-composer-label">Commit message</span>
        {showTypePicker && (
          <div className="commit-composer-type">
            <select
              className="commit-composer-type-select"
              aria-label="Commit type"
              value={prefix?.type ?? ''}
              onChange={(e) => setPrefix(e.target.value, prefix?.scope ?? '')}
            >
              <option value="">No type</option>
              {prefix && !COMMIT_TYPES.includes(prefix.type) && <option value={prefix.type}>{prefix.type}</option>}
              {COMMIT_TYPES.map((type) => (
                <option key={type} value={type}>
                  {type}
                </option>
              ))}
            </select>
            <input
              className="commit-composer-type-scope"
              aria-label="Scope"
              placeholder="scope"
              value={prefix?.scope ?? ''}
              disabled={!prefix}
              onChange={(e) => prefix && setPrefix(prefix.type, e.target.value.replace(/[()\s]/g, ''))}
            />
          </div>
        )}
      </div>
      <div className="commit-composer-field">
        <input
          className="commit-composer-subject"
          value={subject}
          onChange={(e) => setMessage(e.target.value, body)}
          onKeyDown={onMessageKeyDown}
          placeholder="Summary"
        />
        <div className="commit-composer-body-wrap">
          <textarea
            className="commit-composer-body"
            value={body}
            onChange={(e) => setMessage(subject, e.target.value)}
            onKeyDown={onMessageKeyDown}
            placeholder="Description"
            rows={2}
          />
          <details
            ref={coAuthorMenuRef}
            className="commit-composer-coauthors"
            open={coAuthorMenuOpen}
            onToggle={(e) => setCoAuthorMenuOpen(e.currentTarget.open)}
          >
            <summary className="commit-composer-coauthors-toggle" title="Add co-authors" aria-label="Add co-authors">
              <UserPlus size={14} strokeWidth={1.75} />
              {coAuthors.length > 0 && <span className="commit-composer-coauthors-badge">{coAuthors.length}</span>}
            </summary>
            <div className="commit-composer-coauthors-menu">
              {recentAuthors.length === 0 ? (
                <p className="commit-composer-coauthors-empty">No recent authors.</p>
              ) : (
                recentAuthors.map((a) => (
                  <label key={a.email} className="commit-composer-coauthors-option">
                    <input
                      type="checkbox"
                      checked={coAuthors.some((c) => c.email === a.email)}
                      onChange={() => toggleCoAuthor(a)}
                    />
                    {a.name}
                  </label>
                ))
              )}
            </div>
          </details>
        </div>
      </div>
      <div className="commit-composer-meta">
        <span className="commit-composer-counter">
          Subject <span className="commit-composer-counter-value">{subject.length}</span> /{SUBJECT_TARGET}
          <span className="commit-composer-counter-bar">
            <span
              className="commit-composer-counter-fill"
              style={{ width: `${Math.min(100, (subject.length / SUBJECT_TARGET) * 100)}%` }}
            />
          </span>
        </span>
      </div>
      {coAuthors.length > 0 && (
        <div className="commit-composer-coauthor-chips">
          {coAuthors.map((a) => (
            <span key={a.email} className="commit-composer-coauthor-chip">
              {a.name}
              <button type="button" onClick={() => toggleCoAuthor(a)} aria-label={`Remove co-author ${a.name}`}>
                <X size={12} strokeWidth={2} />
              </button>
            </span>
          ))}
        </div>
      )}
      {needsIdentity && (
        <IdentityPrompt
          repoPath={repoPath}
          initialName={currentUser.name}
          initialEmail={currentUser.email}
          onSaved={reloadCurrentUser}
        />
      )}
      <SplitButton
        onClick={() => commit(false)}
        disabled={blockReason !== null}
        menuLabel="More commit actions"
        placement="above"
        open={menuOpen}
        onOpenChange={setMenuOpen}
        menu={
          <>
            <label className="commit-composer-menu-checkbox">
              <input type="checkbox" checked={amend} onChange={(e) => toggleAmend(e.target.checked)} />
              Amend last commit
            </label>
            <label className="commit-composer-menu-checkbox">
              <input type="checkbox" checked={allowEmpty} onChange={(e) => setAllowEmpty(e.target.checked)} />
              Allow empty commit
            </label>
            <label className="commit-composer-menu-checkbox">
              <input type="checkbox" checked={sign} onChange={(e) => setSign(e.target.checked)} />
              Sign this commit
            </label>
            <div className="split-button-menu-divider" />
            <button type="button" onClick={() => commit(true)} disabled={blockReason !== null}>
              Commit and push
            </button>
            <div className="split-button-menu-divider" />
            <button type="button" className="split-button-menu-destructive" onClick={undoLastCommit}>
              Undo last commit
            </button>
          </>
        }
      >
        <GitCommitVertical size={14} strokeWidth={1.75} />
        {amend
          ? 'Amend'
          : stagedCount === 0 && allowEmpty
            ? 'Commit (empty)'
            : `Commit ${stagedCount} file${stagedCount === 1 ? '' : 's'}`}
      </SplitButton>
      {blockReason && <p className="commit-composer-block-hint">{blockReason}</p>}
      {error && <p className="commit-composer-error">{error}</p>}
    </section>
  )
}

export default CommitComposer
