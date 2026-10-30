import { Fragment, useMemo, useRef, useState } from 'react'
import { Copy } from 'lucide-react'
import { InboxService, type StandupRepo } from '@current-client-bindings/app'
import Modal from '../../components/chrome/Modal'
import SegmentedControl from '../../components/controls/SegmentedControl'
import { standupGroups, standupSince, standupText, type StandupFormat, type StandupPeriod } from '../../lib/standup'
import { useAsyncData } from '../../lib/useAsyncData'
import { useCopyToClipboard } from '../../lib/useCopyToClipboard'
import '../../components/git/HostMarkdown.scss'
import './StandupDialog.scss'

interface StandupDialogProps {
  paths: string[]
  onClose: () => void
}

// What you did across the open repositories, as text to paste: commits from
// Git, and with a host on, changes and reviews from its last refresh. Opening
// it asks no host.
function StandupDialog({ paths, onClose }: StandupDialogProps) {
  const [period, setPeriod] = useState<StandupPeriod>('yesterday')
  const [view, setView] = useState<StandupFormat | 'rendered'>('plain')
  const rendered = useRef<HTMLDivElement>(null)
  const since = useMemo(() => standupSince(period).toISOString(), [period])
  const key = paths.join('\n')
  const { data: repos, error } = useAsyncData(() => InboxService.Standup(key.split('\n'), since), [key, since], {
    keepData: true,
  })
  const text = repos ? standupText(repos, view === 'plain' ? 'plain' : 'markdown') : ''
  const { copied, copy } = useCopyToClipboard()
  const notRefreshed = (repos ?? []).filter((r) => r.hostEnabled && !r.fetchedAt).map((r) => r.name)
  const failed = (repos ?? []).filter((r) => r.error)

  return (
    <Modal title="Standup" onClose={onClose} className="dialog-panel standup-panel">
      <div className="dialog-body">
        <div className="standup-controls">
          <SegmentedControl
            value={period}
            onChange={setPeriod}
            options={[
              { value: 'yesterday', label: 'Since yesterday' },
              { value: 'week', label: 'This week' },
            ]}
          />
          <SegmentedControl
            value={view}
            onChange={setView}
            options={[
              { value: 'plain', label: 'Plain' },
              { value: 'markdown', label: 'Markdown' },
              { value: 'rendered', label: 'Rendered' },
            ]}
          />
        </div>
        {error && (
          <p className="dialog-error" role="alert">
            {error}
          </p>
        )}
        {repos &&
          (view === 'rendered' && text ? (
            <div ref={rendered} className="host-markdown standup-text standup-rendered" aria-label="Standup text">
              <StandupRendered repos={repos} />
            </div>
          ) : (
            <pre className="standup-text" aria-label="Standup text">
              {text || 'Nothing since then.'}
            </pre>
          ))}
        <p className="standup-hint">
          Commits by each repository&rsquo;s user.email; changes and reviews from the last refresh.
        </p>
        {notRefreshed.length > 0 && (
          <p className="standup-hint">Not refreshed yet, so no host rows: {notRefreshed.join(', ')}.</p>
        )}
        {failed.map((r) => (
          <p key={r.path} className="dialog-error">
            {r.name}: {r.error}
          </p>
        ))}
        <div className="dialog-actions">
          <button type="button" className="dialog-button" onClick={onClose}>
            Close
          </button>
          <button
            type="button"
            className="dialog-button dialog-button-primary standup-copy"
            onClick={() => copy(text, rendered.current?.innerHTML)}
            disabled={!text}
          >
            <Copy size={14} strokeWidth={1.75} />
            {copied ? 'Copied' : 'Copy'}
          </button>
        </div>
      </div>
    </Modal>
  )
}

// The Markdown view's text as it reads once formatted.
function StandupRendered({ repos }: { repos: StandupRepo[] }) {
  return repos.map((repo) => {
    const parts = standupGroups(repo)
    if (parts.length === 0) return null
    return (
      <Fragment key={repo.path}>
        <h3>{repo.name}</h3>
        {parts.map(([label, items]) => (
          <Fragment key={label}>
            <p>
              <strong>{label}</strong>
            </p>
            <ul>
              {items.map((item, i) => (
                <li key={i}>{item}</li>
              ))}
            </ul>
          </Fragment>
        ))}
      </Fragment>
    )
  })
}

export default StandupDialog
