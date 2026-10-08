import { useState } from 'react'
import { BlameService } from '@current-client-bindings/app'
import { ageIntensity } from './blameAge'
import { relativeTime } from '../../lib/relativeTime'
import { useClockTick } from '../../lib/useClockTick'
import { useAsyncData } from '../../lib/useAsyncData'
import { useCurrentUser } from '../../lib/useCurrentUser'
import Modal from '../../components/chrome/Modal'
import PathText from '../../components/git/PathText'
import './BlameView.scss'

interface BlameViewProps {
  repoPath: string
  path: string
  onClose: () => void
}

function BlameView({ repoPath, path, onClose }: BlameViewProps) {
  useClockTick()
  const { data: lines, error } = useAsyncData(() => BlameService.GetBlame(repoPath, path), [repoPath, path])
  const [focusedLine, setFocusedLine] = useState<number | null>(null)
  const currentUser = useCurrentUser(repoPath)

  return (
    <Modal
      title="Blame"
      onClose={onClose}
      className="blame-view"
      headerContent={<PathText path={path} className="modal-subtitle" />}
    >
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
                <span className="blame-row-content">
                  {l.content}
                  {focused && (
                    <span className="blame-row-ghost">
                      {' '}
                      — {relativeTime(new Date(l.date))}, {isYou ? 'you' : l.authorName}, "{l.summary}"
                    </span>
                  )}
                </span>
              </div>
            )
          })
        )}
      </div>
    </Modal>
  )
}

export default BlameView
