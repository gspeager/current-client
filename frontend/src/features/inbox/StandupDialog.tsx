import { useMemo, useState } from 'react'
import { Copy } from 'lucide-react'
import { InboxService } from '@current-client-bindings/app'
import Modal from '../../components/chrome/Modal'
import SegmentedControl from '../../components/controls/SegmentedControl'
import { standupSince, standupText, type StandupFormat, type StandupPeriod } from '../../lib/standup'
import { useAsyncData } from '../../lib/useAsyncData'
import { useCopyToClipboard } from '../../lib/useCopyToClipboard'
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
  const [format, setFormat] = useState<StandupFormat>('plain')
  const since = useMemo(() => standupSince(period).toISOString(), [period])
  const key = paths.join('\n')
  const { data: repos, error } = useAsyncData(() => InboxService.Standup(key.split('\n'), since), [key, since], {
    keepData: true,
  })
  const text = repos ? standupText(repos, format) : ''
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
            value={format}
            onChange={setFormat}
            options={[
              { value: 'plain', label: 'Plain' },
              { value: 'markdown', label: 'Markdown' },
            ]}
          />
        </div>
        {error && (
          <p className="dialog-error" role="alert">
            {error}
          </p>
        )}
        {repos && (
          <pre className="standup-text" aria-label="Standup text">
            {text || 'Nothing since then.'}
          </pre>
        )}
        <p className="standup-hint">
          Commits are yours on any branch, by each repository&rsquo;s user.email. Merge requests and reviews come from
          the host&rsquo;s last refresh, for repositories with integration on; a review counts where it still stands on
          a change updated since.
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
            className="dialog-button dialog-button-primary"
            onClick={() => copy(text)}
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

export default StandupDialog
