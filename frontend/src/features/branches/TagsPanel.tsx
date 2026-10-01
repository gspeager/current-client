import { useState } from 'react'
import { ChevronDown, ChevronRight, Plus, Tag, Upload, X } from 'lucide-react'
import { RemoteService, TagService, type TagInfo, type TagMessageInfo } from '@current-client-bindings/app'
import { errorMessage } from '../../lib/errors'
import { relativeTime } from '../../lib/relativeTime'
import { withSignIn } from '../../lib/signIn'
import { useAsyncData } from '../../lib/useAsyncData'
import { useDialogs } from '../../lib/useDialogs'
import './TagsPanel.scss'

interface TagsPanelProps {
  repoPath: string
  onTagChanged?: () => void
}

function TagsPanel({ repoPath, onTagChanged }: TagsPanelProps) {
  const {
    data: tags,
    error: loadError,
    reload: loadTags,
  } = useAsyncData(() => TagService.ListTags(repoPath), [repoPath])
  const { confirm, signIn } = useDialogs()
  const [actionError, setActionError] = useState<string | null>(null)
  const error = loadError ?? actionError
  const [newName, setNewName] = useState('')
  const [newMessage, setNewMessage] = useState('')
  const [showNewTag, setShowNewTag] = useState(false)
  const [busyName, setBusyName] = useState<string | null>(null)
  const [expandedName, setExpandedName] = useState<string | null>(null)
  const [messages, setMessages] = useState<Map<string, TagMessageInfo>>(new Map())

  const createTag = () => {
    setActionError(null)
    TagService.CreateTag(repoPath, newName, newMessage, '')
      .then(() => {
        setNewName('')
        setNewMessage('')
        setShowNewTag(false)
        loadTags()
        onTagChanged?.()
      })
      .catch((err: unknown) => setActionError(errorMessage(err)))
  }

  const deleteTag = async (name: string) => {
    const confirmed = await confirm({
      title: 'Delete tag',
      message: `Delete tag "${name}"? This cannot be undone locally.`,
      confirmLabel: 'Delete',
      destructive: true,
    })
    if (!confirmed) return
    setActionError(null)
    setBusyName(name)
    TagService.DeleteTag(repoPath, name)
      .then(() => {
        loadTags()
        onTagChanged?.()
      })
      .catch((err: unknown) => setActionError(errorMessage(err)))
      .finally(() => setBusyName(null))
  }

  const pushTag = (name: string) => {
    setActionError(null)
    setBusyName(name)
    RemoteService.List(repoPath)
      .then((remotes) => {
        const remoteName = remotes.find((r) => r.name === 'origin')?.name ?? remotes[0]?.name
        if (!remoteName) {
          throw new Error('No remote configured to push to.')
        }
        return withSignIn((auth) => TagService.PushTag(repoPath, remoteName, name, auth), signIn)
      })
      .catch((err: unknown) => setActionError(errorMessage(err)))
      .finally(() => setBusyName(null))
  }

  const toggleExpand = (t: TagInfo) => {
    if (!t.annotated) return
    if (expandedName === t.name) {
      setExpandedName(null)
      return
    }
    setExpandedName(t.name)
    if (!messages.has(t.name)) {
      TagService.GetTagMessage(repoPath, t.name)
        .then((result) =>
          setMessages((prev) => new Map(prev).set(t.name, { subject: result.subject, body: result.body })),
        )
        .catch((err: unknown) => setActionError(errorMessage(err)))
    }
  }

  return (
    <div className="tags-panel">
      <div className="tags-panel-header">
        <span className="tags-panel-label">Tags</span>
        <button
          type="button"
          className="tags-panel-add"
          onClick={() => setShowNewTag((v) => !v)}
          aria-label="New tag"
          title="New tag"
        >
          <Plus size={18} strokeWidth={1.75} />
        </button>
      </div>

      {showNewTag && (
        <div className="tags-panel-new">
          <input
            aria-label="new tag name"
            placeholder="Tag name"
            value={newName}
            onChange={(e) => setNewName(e.target.value)}
            autoFocus
          />
          <input
            aria-label="new tag message"
            placeholder="Message (blank = lightweight tag)"
            value={newMessage}
            onChange={(e) => setNewMessage(e.target.value)}
          />
          <button type="button" onClick={createTag} disabled={newName.trim() === ''}>
            Create tag
          </button>
        </div>
      )}

      {error && <p className="tags-panel-error">{error}</p>}
      {tags === null ? (
        <p className="tags-panel-hint">Loading tags…</p>
      ) : tags.length === 0 ? (
        <p className="tags-panel-hint">No tags.</p>
      ) : (
        <ul className="tags-panel-list">
          {tags.map((t) => (
            <li key={t.name} className="tag-row-group">
              <div
                className={t.annotated ? 'tag-row tag-row-expandable' : 'tag-row'}
                onClick={() => toggleExpand(t)}
                title={t.name}
              >
                {t.annotated ? (
                  expandedName === t.name ? (
                    <ChevronDown size={14} strokeWidth={1.5} className="tag-row-chevron" />
                  ) : (
                    <ChevronRight size={14} strokeWidth={1.5} className="tag-row-chevron" />
                  )
                ) : (
                  <Tag size={14} strokeWidth={1.5} className="tag-row-icon" />
                )}
                <span className="tag-row-name">{t.name}</span>
                <span className="tag-row-time">{relativeTime(new Date(t.date))}</span>
                <span className="tag-row-actions">
                  <button
                    type="button"
                    onClick={(e) => {
                      e.stopPropagation()
                      pushTag(t.name)
                    }}
                    disabled={busyName !== null}
                    aria-label={`Push tag ${t.name}`}
                    title="Push"
                  >
                    <Upload size={16} strokeWidth={1.75} />
                  </button>
                  <button
                    type="button"
                    onClick={(e) => {
                      e.stopPropagation()
                      deleteTag(t.name)
                    }}
                    disabled={busyName !== null}
                    aria-label={`Delete tag ${t.name}`}
                    title="Delete"
                  >
                    <X size={16} strokeWidth={1.75} />
                  </button>
                </span>
              </div>
              {t.annotated && expandedName === t.name && (
                <div className="tag-row-message">
                  {messages.has(t.name) ? (
                    <>
                      <p className="tag-row-message-subject">{messages.get(t.name)!.subject}</p>
                      {messages.get(t.name)!.body && (
                        <p className="tag-row-message-body">{messages.get(t.name)!.body}</p>
                      )}
                    </>
                  ) : (
                    <p className="tag-row-message-subject">Loading…</p>
                  )}
                </div>
              )}
            </li>
          ))}
        </ul>
      )}
    </div>
  )
}

export default TagsPanel
