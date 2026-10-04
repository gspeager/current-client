import { useState } from 'react'
import { ArrowDownToLine, Download, Eye, Package, PackagePlus, X } from 'lucide-react'
import { StashService, type StashInfo } from '@current-client-bindings/app'
import Checkbox from '../../components/forms/Checkbox'
import { errorMessage } from '../../lib/errors'
import { relativeTime } from '../../lib/relativeTime'
import { useAsyncData } from '../../lib/useAsyncData'
import { useDialogs } from '../../lib/useDialogs'
import StashModal from './StashModal'
import './StashPanel.scss'

interface StashPanelProps {
  repoPath: string
  dirty: boolean
  onStashChanged?: () => void
}

function StashPanel({ repoPath, dirty, onStashChanged }: StashPanelProps) {
  const {
    data: stashes,
    error: loadError,
    reload: loadStashes,
  } = useAsyncData(() => StashService.ListStash(repoPath), [repoPath])
  const { confirm } = useDialogs()
  const [actionError, setActionError] = useState<string | null>(null)
  const error = loadError ?? actionError
  const [newMessage, setNewMessage] = useState('')
  const [includeUntracked, setIncludeUntracked] = useState(false)
  const [showNewStash, setShowNewStash] = useState(false)
  const [busyIndex, setBusyIndex] = useState<number | null>(null)
  const [shownStash, setShownStash] = useState<StashInfo | null>(null)

  const saveStash = () => {
    setActionError(null)
    StashService.StashSave(repoPath, newMessage, includeUntracked)
      .then(() => {
        setNewMessage('')
        setIncludeUntracked(false)
        setShowNewStash(false)
        loadStashes()
        onStashChanged?.()
      })
      .catch((err: unknown) => setActionError(errorMessage(err)))
  }

  const applyStash = (index: number) => {
    setActionError(null)
    setBusyIndex(index)
    StashService.StashApply(repoPath, index)
      .then(() => onStashChanged?.())
      .catch((err: unknown) => setActionError(errorMessage(err)))
      .finally(() => setBusyIndex(null))
  }

  const popStash = (index: number) => {
    setActionError(null)
    setBusyIndex(index)
    StashService.StashPop(repoPath, index)
      .then(() => {
        loadStashes()
        onStashChanged?.()
      })
      .catch((err: unknown) => setActionError(errorMessage(err)))
      .finally(() => setBusyIndex(null))
  }

  const dropStash = async (index: number) => {
    const confirmed = await confirm({
      title: 'Drop stash',
      message: 'Drop this stash? This cannot be undone.',
      confirmLabel: 'Drop',
      destructive: true,
    })
    if (!confirmed) return
    setActionError(null)
    setBusyIndex(index)
    StashService.StashDrop(repoPath, index)
      .then(() => loadStashes())
      .catch((err: unknown) => setActionError(errorMessage(err)))
      .finally(() => setBusyIndex(null))
  }

  return (
    <div className="stash-panel">
      <div className="stash-panel-header">
        <span className="stash-panel-label">Stash</span>
        <button
          type="button"
          className="stash-panel-add"
          onClick={() => setShowNewStash((v) => !v)}
          aria-label="Stash changes"
          title="Stash changes"
          disabled={!dirty}
        >
          <PackagePlus size={18} strokeWidth={1.75} />
        </button>
      </div>

      {showNewStash && (
        <div className="stash-panel-new">
          <input
            aria-label="new stash message"
            placeholder="Stash message (optional)"
            value={newMessage}
            onChange={(e) => setNewMessage(e.target.value)}
            autoFocus
          />
          <Checkbox checked={includeUntracked} onChange={setIncludeUntracked} label="Include untracked files" />
          <button type="button" onClick={saveStash}>
            Stash
          </button>
        </div>
      )}

      {error && <p className="stash-panel-error">{error}</p>}
      {stashes === null ? (
        <p className="stash-panel-hint">Loading stashes…</p>
      ) : stashes.length === 0 ? (
        <p className="stash-panel-hint">No stashes.</p>
      ) : (
        <ul className="stash-panel-list">
          {stashes.map((s) => (
            <li key={s.index} className="stash-row" title={s.message}>
              <Package size={14} strokeWidth={1.5} className="stash-row-icon" />
              <span className="stash-row-message">{s.message}</span>
              <span className="stash-row-time">{relativeTime(new Date(s.date))}</span>
              <span className="stash-row-actions">
                <button
                  type="button"
                  onClick={() => setShownStash(s)}
                  aria-label={`Show changes in stash: ${s.message}`}
                  title="Show changes"
                >
                  <Eye size={16} strokeWidth={1.75} />
                </button>
                <button
                  type="button"
                  onClick={() => applyStash(s.index)}
                  disabled={busyIndex !== null}
                  aria-label={`Apply stash: ${s.message}`}
                  title="Apply (keep in list)"
                >
                  <Download size={16} strokeWidth={1.75} />
                </button>
                <button
                  type="button"
                  onClick={() => popStash(s.index)}
                  disabled={busyIndex !== null}
                  aria-label={`Pop stash: ${s.message}`}
                  title="Pop (apply and remove)"
                >
                  <ArrowDownToLine size={16} strokeWidth={1.75} />
                </button>
                <button
                  type="button"
                  onClick={() => dropStash(s.index)}
                  disabled={busyIndex !== null}
                  aria-label={`Drop stash: ${s.message}`}
                  title="Drop"
                >
                  <X size={16} strokeWidth={1.75} />
                </button>
              </span>
            </li>
          ))}
        </ul>
      )}
      {shownStash && <StashModal repoPath={repoPath} stash={shownStash} onClose={() => setShownStash(null)} />}
    </div>
  )
}

export default StashPanel
