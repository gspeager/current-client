import { useState } from 'react'
import { BlameService, type BlameLineInfo } from '@current-client-bindings/app'
import { ageIntensity } from './blameAge'
import { relativeTime } from '../../lib/relativeTime'
import { useClockTick } from '../../lib/useClockTick'
import { useAsyncData } from '../../lib/useAsyncData'
import { useCurrentUser } from '../../lib/useCurrentUser'
import { useOpenCommit } from '../../lib/openCommit'
import { shortSha } from '../../lib/text'
import Modal from '../../components/chrome/Modal'
import PathText from '../../components/git/PathText'
import './BlameView.scss'

interface BlameViewProps {
  repoPath: string
  path: string
  onClose: () => void
}

// Where Blame before this commit has stepped back to.
interface Before {
  rev: string
  path: string
  line: number
}

const uncommitted = (sha: string) => /^0+$/.test(sha)

function BlameView({ repoPath, path, onClose }: BlameViewProps) {
  useClockTick()
  const [steps, setSteps] = useState<Before[]>([])
  const at = steps[steps.length - 1]
  const { data: lines, error } = useAsyncData(
    () => (at ? BlameService.GetBlameAt(repoPath, at.path, at.rev) : BlameService.GetBlame(repoPath, path)),
    [repoPath, path, at],
  )
  const [focusedLine, setFocusedLine] = useState<number | null>(null)
  const [copied, setCopied] = useState<string | null>(null)
  const currentUser = useCurrentUser(repoPath)
  const openCommit = useOpenCommit()

  const stepBack = (l: BlameLineInfo) => {
    setSteps((prev) => [...prev, { rev: l.previous, path: l.previousPath, line: l.lineNo }])
    setFocusedLine(null)
  }
  const stepForward = (to: number) => {
    setFocusedLine(steps[to]?.line ?? null)
    setSteps((prev) => prev.slice(0, to))
  }
  const copy = (sha: string) => void navigator.clipboard.writeText(sha).then(() => setCopied(sha))

  return (
    <Modal
      title="Blame"
      onClose={onClose}
      className="blame-view"
      headerContent={<PathText path={at?.path ?? path} className="modal-subtitle" />}
    >
      {at && (
        <p className="blame-at" role="status">
          As of {shortSha(at.rev)}, before the line’s commit{at.path !== path && `, when it was ${at.path}`}.
          <button type="button" onClick={() => stepForward(steps.length - 1)}>
            Back
          </button>
          {steps.length > 1 && (
            <button type="button" onClick={() => stepForward(0)}>
              Latest
            </button>
          )}
        </p>
      )}
      <div className="blame-body">
        {error ? (
          <p className="blame-hint">Could not load blame: {error}</p>
        ) : lines === null ? (
          <p className="blame-hint">Loading blame…</p>
        ) : lines.length === 0 ? (
          <p className="blame-hint">No lines.</p>
        ) : (
          lines.map((l) => {
            const focused = focusedLine === l.lineNo
            const isYou = currentUser !== null && l.authorEmail === currentUser.email
            const committed = !uncommitted(l.sha)
            return (
              <div
                key={l.lineNo}
                className={focused ? 'blame-row blame-row-focused' : 'blame-row'}
                onClick={() => setFocusedLine(focused ? null : l.lineNo)}
                title={`${l.authorName} — ${l.summary}`}
              >
                <span
                  className="blame-row-gutter"
                  style={{
                    background: `color-mix(in srgb, var(--warning) ${ageIntensity(new Date(l.date))}%, transparent)`,
                  }}
                />
                <span className="blame-row-lineno">{l.lineNo}</span>
                <span className="blame-row-content">{l.content}</span>
                {focused && (
                  <div className="blame-row-detail" onClick={(e) => e.stopPropagation()}>
                    {committed ? (
                      <>
                        <code className="blame-row-sha">{shortSha(l.sha)}</code>
                        <span>
                          {relativeTime(new Date(l.date))}, {isYou ? 'you' : l.authorName}, “{l.summary}”
                        </span>
                        {openCommit && (
                          <button
                            type="button"
                            onClick={() => {
                              onClose()
                              openCommit(l.sha)
                            }}
                          >
                            Open in History
                          </button>
                        )}
                        <button type="button" onClick={() => copy(l.sha)}>
                          {copied === l.sha ? 'Copied' : 'Copy hash'}
                        </button>
                        {l.previous ? (
                          <button type="button" onClick={() => stepBack(l)}>
                            Blame before this commit
                          </button>
                        ) : (
                          <span className="blame-row-first">The commit that added this file</span>
                        )}
                      </>
                    ) : (
                      <span>Not committed yet</span>
                    )}
                  </div>
                )}
              </div>
            )
          })
        )}
      </div>
    </Modal>
  )
}

export default BlameView
