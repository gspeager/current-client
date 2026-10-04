import { useState } from 'react'
import { Cloud, GitBranchPlus, RefreshCw, Search, X } from 'lucide-react'
import { BranchService, StatusService } from '@current-client-bindings/app'
import { localNameFor } from './branches'
import { toBranchName } from '../../lib/branchName'
import { errorMessage } from '../../lib/errors'
import { useLaneColors } from '../../lib/laneColor'
import { relativeTime } from '../../lib/relativeTime'
import { useAsyncData } from '../../lib/useAsyncData'
import { useDialogs } from '../../lib/useDialogs'
import { useDeleteRemoteBranch } from '../remotes/useDeleteRemoteBranch'
import { useFetchAll } from '../remotes/useFetchAll'
import BranchPill from '../../components/git/BranchPill'
import ContextMenu, { type ContextMenuState } from '../../components/controls/ContextMenu'
import './BranchSwitcher.scss'

interface BranchSwitcherProps {
  repoPath: string
  onBranchChanged?: () => void
  onFetched?: () => void
}

function loadSwitcher(repoPath: string) {
  return Promise.all([
    BranchService.ListLocal(repoPath),
    BranchService.ListRemote(repoPath),
    BranchService.CurrentBranchStatus(repoPath)
      .then((status) => status.ahead)
      .catch(() => 0),
    StatusService.GetStatus(repoPath)
      .then((files) => files.length)
      .catch(() => 0),
  ]).then(([local, remote, ahead, dirtyCount]) => ({ local, remote, ahead, dirtyCount }))
}

function BranchSwitcher({ repoPath, onBranchChanged, onFetched }: BranchSwitcherProps) {
  const { branchColor } = useLaneColors()
  const [filter, setFilter] = useState('')
  const { data, error: loadError, reload: load } = useAsyncData(() => loadSwitcher(repoPath), [repoPath])
  const { prompt } = useDialogs()
  const [actionError, setActionError] = useState<string | null>(null)
  const error = loadError ?? actionError
  const localBranches = data?.local ?? []
  const remoteBranches = data?.remote ?? []
  const ahead = data?.ahead ?? 0
  const dirtyCount = data?.dirtyCount ?? 0

  const checkoutBranch = (name: string) => {
    setActionError(null)
    BranchService.CheckoutBranch(repoPath, name)
      .then(() => {
        load()
        onBranchChanged?.()
      })
      .catch((err: unknown) => setActionError(errorMessage(err)))
  }

  const newBranchFromHead = async () => {
    const name = await prompt({
      title: 'New branch',
      label: 'Branch name (from HEAD)',
      confirmLabel: 'Create',
      transform: toBranchName,
    })
    if (!name) return
    setActionError(null)
    BranchService.CreateBranch(repoPath, name)
      .then(() => BranchService.CheckoutBranch(repoPath, name))
      .then(() => {
        load()
        onBranchChanged?.()
      })
      .catch((err: unknown) => setActionError(errorMessage(err)))
  }

  const fetchAllOp = useFetchAll(repoPath, () => {
    onFetched?.()
    load()
  })

  const [contextMenu, setContextMenu] = useState<ContextMenuState | null>(null)
  const deleteRemote = useDeleteRemoteBranch(repoPath, () => {
    onBranchChanged?.()
    load()
  })

  const current = localBranches.find((b) => b.current) ?? null
  const query = filter.trim().toLowerCase()
  const filteredLocal = localBranches.filter((b) => !b.current && b.name.toLowerCase().includes(query))
  const localNames = new Set(localBranches.map((b) => b.name))
  const filteredRemote = remoteBranches.filter((name) => name.toLowerCase().includes(query))

  return (
    <div className="branch-switcher">
      <div className="branch-switcher-filter">
        <Search size={14} strokeWidth={1.5} />
        <input
          aria-label="Filter branches"
          placeholder="Filter branches…"
          value={filter}
          onChange={(e) => setFilter(e.target.value)}
          autoFocus
        />
      </div>

      {error && <p className="branch-switcher-error">{error}</p>}
      {fetchAllOp.error && <p className="branch-switcher-error">Could not fetch: {fetchAllOp.error}</p>}
      {deleteRemote.error && <p className="branch-switcher-error">Could not delete on remote: {deleteRemote.error}</p>}

      {current && (
        <div className="branch-switcher-section">
          <span className="branch-switcher-label">Checked out</span>
          <div className="branch-switcher-row branch-switcher-row-current">
            <BranchPill
              name={current.name}
              color={branchColor(current.name, true)}
              current
              bare
              dirty={dirtyCount > 0}
            />
            <span className="branch-switcher-meta">
              {ahead > 0 && <span className="branch-switcher-ahead">↑{ahead}</span>}
              {dirtyCount > 0 && <span className="branch-switcher-dirty">{dirtyCount} dirty</span>}
            </span>
          </div>
        </div>
      )}

      <div className="branch-switcher-section">
        <div className="branch-switcher-section-header">
          <span className="branch-switcher-label">Local</span>
          <span className="branch-switcher-count">{filteredLocal.length}</span>
        </div>
        {filteredLocal.map((b) => (
          <button type="button" key={b.name} className="branch-switcher-row" onClick={() => checkoutBranch(b.name)}>
            <BranchPill name={b.name} color={branchColor(b.name, false)} bare />
            <span className="branch-switcher-meta">
              {b.ahead > 0 && <span className="branch-switcher-ahead">↑{b.ahead}</span>}
              {b.behind > 0 && <span className="branch-switcher-behind">↓{b.behind}</span>}
              {b.lastCommitDate && (
                <span className="branch-switcher-time">{relativeTime(new Date(b.lastCommitDate))}</span>
              )}
            </span>
          </button>
        ))}
      </div>

      <div className="branch-switcher-section">
        <div className="branch-switcher-section-header">
          <span className="branch-switcher-label">Remote-tracking</span>
          <span className="branch-switcher-count">{filteredRemote.length}</span>
        </div>
        {filteredRemote.map((name) => {
          const notLocal = !localNames.has(localNameFor(name))
          return (
            <button
              type="button"
              key={name}
              className="branch-switcher-row branch-switcher-row-remote"
              onClick={() => checkoutBranch(localNameFor(name))}
              onContextMenu={(e) => {
                e.preventDefault()
                setContextMenu({
                  x: e.clientX,
                  y: e.clientY,
                  items: [
                    { label: 'Checkout', onClick: () => checkoutBranch(localNameFor(name)) },
                    {
                      label: 'Delete on remote…',
                      onClick: () => void deleteRemote.deleteRemoteBranch(name),
                      destructive: true,
                      disabled: deleteRemote.running,
                    },
                  ],
                })
              }}
            >
              <Cloud size={14} strokeWidth={1.5} className="branch-switcher-remote-icon" />
              <span className="branch-switcher-remote-name">{name}</span>
              {notLocal && <span className="branch-switcher-meta">not local</span>}
            </button>
          )
        })}
      </div>

      <div className="branch-switcher-footer">
        <button type="button" onClick={newBranchFromHead}>
          <GitBranchPlus size={14} strokeWidth={1.75} />
          New branch from HEAD
        </button>
        <button type="button" onClick={fetchAllOp.running ? fetchAllOp.cancel : fetchAllOp.fetchAll}>
          {fetchAllOp.running ? <X size={14} strokeWidth={1.75} /> : <RefreshCw size={14} strokeWidth={1.75} />}
          {fetchAllOp.running ? 'Fetching… (cancel)' : 'Fetch all'}
        </button>
      </div>
      {contextMenu && <ContextMenu state={contextMenu} onClose={() => setContextMenu(null)} />}
    </div>
  )
}

export default BranchSwitcher
