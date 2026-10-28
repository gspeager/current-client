import { useState } from 'react'
import { Cloud, Pencil, Plus, RefreshCw, X } from 'lucide-react'
import { RowActionButtons, useRowMenu, type RowAction } from '../../components/controls/RowActions'
import { RemoteService, type RemoteInfo } from '@current-client-bindings/app'
import { useCancellableOperation } from '../../lib/cancellableOperation'
import { errorMessage } from '../../lib/errors'
import { useAsyncData } from '../../lib/useAsyncData'
import { useDialogs } from '../../lib/useDialogs'
import PathText from '../../components/git/PathText'
import SidebarSection from '../../components/chrome/SidebarSection'
import type { SidebarSections } from '../../lib/useSidebarSections'
import './RemotesPanel.scss'

interface RemotesPanelProps {
  repoPath: string
  sections: SidebarSections
  onFetched?: () => void
  pruneOnFetch?: boolean
}

// The host for a URL; null for a filesystem remote, whose path is shown instead.
function hostLabel(url: string): string | null {
  const httpMatch = url.match(/^\w+:\/\/([^/]+)/)
  if (httpMatch) return httpMatch[1].replace(/^www\./, '')
  const sshMatch = url.match(/^[\w.-]+@([^:/]+)/)
  if (sshMatch) return sshMatch[1]
  return null
}

function RemotesPanel({ repoPath, sections, onFetched, pruneOnFetch = false }: RemotesPanelProps) {
  const {
    data: remotes,
    error: loadError,
    reload: loadRemotes,
  } = useAsyncData(() => RemoteService.List(repoPath), [repoPath])
  const { confirm, prompt } = useDialogs()
  const [actionError, setActionError] = useState<string | null>(null)
  const error = loadError ?? actionError
  const [newName, setNewName] = useState('')
  const [newUrl, setNewUrl] = useState('')
  const [showAddRemote, setShowAddRemote] = useState(false)
  const [fetchingRemote, setFetchingRemote] = useState<string | null>(null)
  const fetchOp = useCancellableOperation()
  const rowMenu = useRowMenu()

  const addRemote = () => {
    setActionError(null)
    RemoteService.AddRemote(repoPath, newName, newUrl)
      .then(() => {
        setNewName('')
        setNewUrl('')
        setShowAddRemote(false)
        loadRemotes()
      })
      .catch((err: unknown) => setActionError(errorMessage(err)))
  }

  const removeRemote = async (name: string) => {
    const confirmed = await confirm({
      title: 'Remove remote',
      message: `Remove remote "${name}"?`,
      confirmLabel: 'Remove',
      destructive: true,
    })
    if (!confirmed) return
    setActionError(null)
    RemoteService.RemoveRemote(repoPath, name)
      .then(loadRemotes)
      .catch((err: unknown) => setActionError(errorMessage(err)))
  }

  const editRemote = async (remote: RemoteInfo) => {
    const url = await prompt({
      title: 'Edit remote',
      label: `New URL for "${remote.name}"`,
      initialValue: remote.fetchUrl,
      confirmLabel: 'Save',
    })
    if (!url) return
    setActionError(null)
    RemoteService.EditRemote(repoPath, remote.name, url)
      .then(loadRemotes)
      .catch((err: unknown) => setActionError(errorMessage(err)))
  }

  const fetchRemote = (name: string) => {
    setFetchingRemote(name)
    fetchOp.run((auth) => RemoteService.Fetch(repoPath, name, auth), onFetched).finally(() => setFetchingRemote(null))
  }

  const fetchAll = () => {
    setFetchingRemote('*')
    const fetch = pruneOnFetch ? RemoteService.FetchAllPrune : RemoteService.FetchAll
    fetchOp.run((auth) => fetch(repoPath, auth), onFetched).finally(() => setFetchingRemote(null))
  }

  const hasRemotes = (remotes?.length ?? 0) > 0

  const remoteActions = (r: RemoteInfo): RowAction[] => [
    fetchingRemote === r.name
      ? { label: 'Cancel fetching', ariaLabel: `Cancel fetching ${r.name}`, icon: X, run: fetchOp.cancel }
      : {
          label: 'Fetch',
          ariaLabel: `Fetch ${r.name}`,
          icon: RefreshCw,
          run: () => fetchRemote(r.name),
          disabled: fetchingRemote !== null,
        },
    { label: 'Edit…', ariaLabel: `Edit ${r.name}`, icon: Pencil, run: () => void editRemote(r) },
    {
      label: 'Remove…',
      ariaLabel: `Remove ${r.name}`,
      icon: X,
      run: () => void removeRemote(r.name),
      destructive: true,
    },
  ]

  return (
    <div className="remotes-panel">
      <SidebarSection
        label="Remotes"
        count={remotes?.length}
        open={sections.isOpen('remotes', hasRemotes)}
        onToggle={() => sections.toggle('remotes', hasRemotes)}
        actions={
          <button
            type="button"
            className="remotes-panel-add"
            onClick={() => setShowAddRemote((v) => !v)}
            aria-label="Add remote"
            title="Add remote"
          >
            <Plus size={18} strokeWidth={1.75} />
          </button>
        }
        pinned={
          showAddRemote && (
            <div className="remotes-panel-new">
              <input
                aria-label="new remote name"
                placeholder="Remote name"
                value={newName}
                onChange={(e) => setNewName(e.target.value)}
                autoFocus
              />
              <input
                aria-label="new remote url"
                placeholder="Remote URL"
                value={newUrl}
                onChange={(e) => setNewUrl(e.target.value)}
              />
              <button type="button" onClick={addRemote} disabled={newName.trim() === '' || newUrl.trim() === ''}>
                Add remote
              </button>
            </div>
          )
        }
      >
        {error && <p className="remotes-panel-error">{error}</p>}
        {fetchOp.error && <p className="remotes-panel-error">Could not fetch: {fetchOp.error}</p>}
        {remotes === null ? (
          <p className="remotes-panel-hint">Loading remotes…</p>
        ) : remotes.length === 0 ? (
          <p className="remotes-panel-hint">No remotes.</p>
        ) : (
          <>
            {remotes.length > 1 && (
              <button
                type="button"
                className="remotes-panel-fetch-all"
                onClick={fetchingRemote === '*' ? fetchOp.cancel : fetchAll}
                disabled={fetchingRemote !== null && fetchingRemote !== '*'}
              >
                {fetchingRemote === '*' ? 'Fetching all… (cancel)' : 'Fetch all'}
              </button>
            )}
            <ul className="remotes-panel-list">
              {remotes.map((r) => (
                <li
                  key={r.name}
                  className="remote-row"
                  title={r.fetchUrl}
                  onContextMenu={rowMenu.open(remoteActions(r))}
                >
                  <Cloud size={14} strokeWidth={1.5} className="remote-row-icon" />
                  <span className="remote-row-name">{r.name}</span>
                  {hostLabel(r.fetchUrl) ? (
                    <span className="remote-row-host">{hostLabel(r.fetchUrl)}</span>
                  ) : (
                    <PathText path={r.fetchUrl} className="remote-row-path" />
                  )}
                  <RowActionButtons actions={remoteActions(r)} className="remote-row-actions" />
                </li>
              ))}
            </ul>
          </>
        )}
      </SidebarSection>
      {rowMenu.menu}
    </div>
  )
}

export default RemotesPanel
