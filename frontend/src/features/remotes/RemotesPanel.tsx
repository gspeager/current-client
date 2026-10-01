import { useState } from 'react'
import { Cloud, Pencil, Plus, RefreshCw, X } from 'lucide-react'
import { RemoteService, type RemoteInfo } from '@current-client-bindings/app'
import { useCancellableOperation } from '../../lib/cancellableOperation'
import { errorMessage } from '../../lib/errors'
import { useAsyncData } from '../../lib/useAsyncData'
import { useDialogs } from '../../lib/useDialogs'
import './RemotesPanel.scss'

interface RemotesPanelProps {
  repoPath: string
  onFetched?: () => void
}

function hostLabel(url: string): string {
  const httpMatch = url.match(/^\w+:\/\/([^/]+)/)
  if (httpMatch) return httpMatch[1].replace(/^www\./, '')
  const sshMatch = url.match(/^[\w.-]+@([^:/]+)/)
  if (sshMatch) return sshMatch[1]
  return url
}

function RemotesPanel({ repoPath, onFetched }: RemotesPanelProps) {
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
    fetchOp.run(RemoteService.Fetch(repoPath, name), onFetched).finally(() => setFetchingRemote(null))
  }

  const fetchAll = () => {
    setFetchingRemote('*')
    fetchOp.run(RemoteService.FetchAll(repoPath), onFetched).finally(() => setFetchingRemote(null))
  }

  return (
    <div className="remotes-panel">
      <div className="remotes-panel-header">
        <span className="remotes-panel-label">Remotes</span>
        <button
          type="button"
          className="remotes-panel-add"
          onClick={() => setShowAddRemote((v) => !v)}
          aria-label="Add remote"
          title="Add remote"
        >
          <Plus size={18} strokeWidth={1.75} />
        </button>
      </div>

      {showAddRemote && (
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
      )}

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
              <li key={r.name} className="remote-row" title={r.fetchUrl}>
                <Cloud size={14} strokeWidth={1.5} className="remote-row-icon" />
                <span className="remote-row-name">{r.name}</span>
                <span className="remote-row-host">{hostLabel(r.fetchUrl)}</span>
                <span className="remote-row-actions">
                  <button
                    type="button"
                    onClick={() => (fetchingRemote === r.name ? fetchOp.cancel() : fetchRemote(r.name))}
                    disabled={fetchingRemote !== null && fetchingRemote !== r.name}
                    aria-label={fetchingRemote === r.name ? `Cancel fetching ${r.name}` : `Fetch ${r.name}`}
                    title={fetchingRemote === r.name ? 'Cancel fetching' : 'Fetch'}
                  >
                    {fetchingRemote === r.name ? (
                      <X size={16} strokeWidth={1.75} />
                    ) : (
                      <RefreshCw size={16} strokeWidth={1.75} />
                    )}
                  </button>
                  <button type="button" onClick={() => editRemote(r)} aria-label={`Edit ${r.name}`} title="Edit">
                    <Pencil size={16} strokeWidth={1.75} />
                  </button>
                  <button
                    type="button"
                    onClick={() => removeRemote(r.name)}
                    aria-label={`Remove ${r.name}`}
                    title="Remove"
                  >
                    <X size={16} strokeWidth={1.75} />
                  </button>
                </span>
              </li>
            ))}
          </ul>
        </>
      )}
    </div>
  )
}

export default RemotesPanel
