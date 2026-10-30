import { useState } from 'react'
import { ArrowDownToLine, Download, Eye, Package, PackagePlus, X } from 'lucide-react'
import { RowActionButtons, useRowMenu, type RowAction } from '../../components/controls/RowActions'
import { StashService, type StashInfo } from '@current-client-bindings/app'
import Checkbox from '../../components/forms/Checkbox'
import { errorMessage } from '../../lib/errors'
import { relativeTime } from '../../lib/relativeTime'
import { useClockTick } from '../../lib/useClockTick'
import { useAsyncData } from '../../lib/useAsyncData'
import { useDialogs } from '../../lib/useDialogs'
import StashModal from './StashModal'
import SidebarSection from '../../components/chrome/SidebarSection'
import type { SidebarSections } from '../../lib/useSidebarSections'
import './StashPanel.scss'

interface StashPanelProps {
  repoPath: string
  dirty: boolean
  // Reloads the list when it changes, such as after a stash made from Changes.
  refreshKey?: unknown
  sections: SidebarSections
  onStashChanged?: () => void
}

function StashPanel({ repoPath, dirty, refreshKey, sections, onStashChanged }: StashPanelProps) {
  useClockTick()
  const {
    data: stashes,
    error: loadError,
    reload: loadStashes,
  } = useAsyncData(() => StashService.ListStash(repoPath), [repoPath], { refreshKey })
  const { confirm } = useDialogs()
  const [actionError, setActionError] = useState<string | null>(null)
  const error = loadError ?? actionError
  const [newMessage, setNewMessage] = useState('')
  const [includeUntracked, setIncludeUntracked] = useState(false)
  const [keepIndex, setKeepIndex] = useState(false)
  const [showNewStash, setShowNewStash] = useState(false)
  const [busyIndex, setBusyIndex] = useState<number | null>(null)
  const [shownStash, setShownStash] = useState<StashInfo | null>(null)
  const rowMenu = useRowMenu()

  const saveStash = () => {
    setActionError(null)
    StashService.StashSave(repoPath, { message: newMessage, includeUntracked, keepIndex, paths: [] })
      .then(() => {
        setNewMessage('')
        setIncludeUntracked(false)
        setKeepIndex(false)
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

  const open = sections.isOpen('stash', (stashes?.length ?? 0) > 0)

  const toggleNewStash = () => {
    setShowNewStash((v) => !v)
    sections.open('stash')
  }

  const stashActions = (s: StashInfo): RowAction[] => {
    const busy = busyIndex !== null
    return [
      {
        label: 'Show changes',
        ariaLabel: `Show changes in stash: ${s.message}`,
        icon: Eye,
        run: () => setShownStash(s),
      },
      {
        label: 'Apply (keep in list)',
        ariaLabel: `Apply stash: ${s.message}`,
        icon: Download,
        run: () => applyStash(s.index),
        disabled: busy,
      },
      {
        label: 'Pop (apply and remove)',
        ariaLabel: `Pop stash: ${s.message}`,
        icon: ArrowDownToLine,
        run: () => popStash(s.index),
        disabled: busy,
      },
      {
        label: 'Drop…',
        ariaLabel: `Drop stash: ${s.message}`,
        icon: X,
        run: () => void dropStash(s.index),
        disabled: busy,
        destructive: true,
      },
    ]
  }

  return (
    <div className="stash-panel">
      <SidebarSection
        label="Stash"
        count={stashes?.length}
        open={open}
        onToggle={() => sections.toggle('stash', (stashes?.length ?? 0) > 0)}
        actions={
          <button
            type="button"
            className="stash-panel-add"
            onClick={toggleNewStash}
            aria-label="Stash changes"
            title="Stash changes"
            disabled={!dirty}
          >
            <PackagePlus size={18} strokeWidth={1.75} />
          </button>
        }
        pinned={
          showNewStash && (
            <div className="stash-panel-new">
              <input
                aria-label="new stash message"
                placeholder="Stash message (optional)"
                value={newMessage}
                onChange={(e) => setNewMessage(e.target.value)}
                autoFocus
              />
              <Checkbox checked={includeUntracked} onChange={setIncludeUntracked} label="Include untracked files" />
              <Checkbox checked={keepIndex} onChange={setKeepIndex} label="Keep staged changes" />
              <button type="button" onClick={saveStash}>
                Stash
              </button>
            </div>
          )
        }
      >
        {error && <p className="stash-panel-error">{error}</p>}
        {stashes === null ? (
          <p className="stash-panel-hint">Loading stashes…</p>
        ) : stashes.length === 0 ? (
          <p className="stash-panel-hint">No stashes.</p>
        ) : (
          <ul className="stash-panel-list">
            {stashes.map((s) => (
              <li key={s.index} className="stash-row" title={s.message} onContextMenu={rowMenu.open(stashActions(s))}>
                <Package size={14} strokeWidth={1.5} className="stash-row-icon" />
                <span className="stash-row-message">{s.message}</span>
                <span className="stash-row-time">{relativeTime(new Date(s.date))}</span>
                <RowActionButtons actions={stashActions(s)} className="stash-row-actions" />
              </li>
            ))}
          </ul>
        )}
      </SidebarSection>
      {rowMenu.menu}
      {shownStash && <StashModal repoPath={repoPath} stash={shownStash} onClose={() => setShownStash(null)} />}
    </div>
  )
}

export default StashPanel
