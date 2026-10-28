import { useEffect, useState } from 'react'
import {
  ChevronDown,
  ChevronRight,
  ClipboardCheck,
  CloudOff,
  Copy,
  ExternalLink,
  PackagePlus,
  Plus,
  Tag,
  Upload,
  X,
} from 'lucide-react'
import {
  HostService,
  PlatformService,
  type TagInfo,
  type TagMessageInfo,
  TagService,
} from '@current-client-bindings/app'
import { errorMessage } from '../../lib/errors'
import { primaryRemoteName } from '../../lib/remotes'
import { relativeTime } from '../../lib/relativeTime'
import { useClockTick } from '../../lib/useClockTick'
import { withSignIn } from '../../lib/signIn'
import { useAsyncData } from '../../lib/useAsyncData'
import { useDialogs } from '../../lib/useDialogs'
import { useCopyToClipboard } from '../../lib/useCopyToClipboard'
import { RowActionButtons, useRowMenu, type RowAction } from '../../components/controls/RowActions'
import './TagsPanel.scss'
import { useHostLinks } from '../../lib/useHostLinks'
import type { SidebarRequest } from '../../components/chrome/NavPane'
import { tagPage } from '../../lib/hostPages'
import SidebarSection from '../../components/chrome/SidebarSection'
import PublishReleaseDialog from '../host-changes/PublishReleaseDialog'
import ReleaseReadinessDialog from './ReleaseReadinessDialog'
import type { SidebarSections } from '../../lib/useSidebarSections'

interface TagsPanelProps {
  repoPath: string
  sections: SidebarSections
  onTagChanged?: () => void
  request?: SidebarRequest | null
}

function TagsPanel({ repoPath, sections, onTagChanged, request = null }: TagsPanelProps) {
  const hostLinks = useHostLinks(repoPath)
  useClockTick()
  const {
    data: tags,
    error: loadError,
    reload: loadTags,
  } = useAsyncData(() => TagService.ListTags(repoPath), [repoPath])
  const { confirm, signIn } = useDialogs()
  const { copy } = useCopyToClipboard()
  const rowMenu = useRowMenu()
  const [actionError, setActionError] = useState<string | null>(null)
  const error = loadError ?? actionError
  const [newName, setNewName] = useState('')
  const [newMessage, setNewMessage] = useState('')
  const [showNewTag, setShowNewTag] = useState(false)
  const [busyName, setBusyName] = useState<string | null>(null)
  const [expandedName, setExpandedName] = useState<string | null>(null)
  const [messages, setMessages] = useState<Map<string, TagMessageInfo>>(new Map())
  const [releaseTag, setReleaseTag] = useState<string | null>(null)
  const [readiness, setReadiness] = useState(false)
  const { data: canPublish } = useAsyncData(
    () => HostService.CanPublishReleases(repoPath).catch(() => false),
    [repoPath],
  )

  // Publish release from the palette takes the newest tag, which is listed first.
  useEffect(() => {
    if (!request) return
    if (request.action === 'new-tag') setShowNewTag(true)
    if (request.action === 'release-readiness') setReadiness(true)
    if (request.action === 'publish-release') {
      const newest = tags?.[0]
      if (!newest) setActionError('No tag to publish a release from')
      else if (!canPublish) setActionError('Publishing a release needs the host integration on with write scope')
      else setReleaseTag(newest.name)
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [request])

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
    primaryRemoteName(repoPath)
      .then((remote) => withSignIn((auth) => TagService.PushTag(repoPath, remote, name, auth), signIn))
      .catch((err: unknown) => setActionError(errorMessage(err)))
      .finally(() => setBusyName(null))
  }

  const deleteRemoteTag = async (name: string) => {
    setActionError(null)
    let remote: string
    try {
      remote = await primaryRemoteName(repoPath)
    } catch (err) {
      setActionError(errorMessage(err))
      return
    }
    const confirmed = await confirm({
      title: 'Delete tag on remote',
      message: `Delete tag "${name}" on ${remote}? It's removed for everyone who uses ${remote}. The local tag is kept.`,
      confirmLabel: 'Delete',
      destructive: true,
    })
    if (!confirmed) return
    setBusyName(name)
    withSignIn((auth) => TagService.DeleteRemoteTag(repoPath, remote, name, auth), signIn)
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

  const hasTags = (tags?.length ?? 0) > 0

  const busy = busyName !== null
  const tagActions = (name: string): RowAction[] => [
    ...(hostLinks?.tag
      ? [
          {
            label: 'Open on host',
            ariaLabel: `Open tag ${name} on host`,
            icon: ExternalLink,
            run: () => void PlatformService.OpenWebPage(tagPage(hostLinks, name)),
          },
        ]
      : []),
    ...(canPublish
      ? [
          {
            label: 'Publish release…',
            ariaLabel: `Publish release for ${name} on host`,
            icon: PackagePlus,
            run: () => setReleaseTag(name),
            disabled: busy,
          },
        ]
      : []),
    { label: 'Push tag', ariaLabel: `Push tag ${name}`, icon: Upload, run: () => pushTag(name), disabled: busy },
    { label: 'Copy name', ariaLabel: `Copy ${name}`, icon: Copy, run: () => copy(name), menuOnly: true },
    {
      label: 'Delete on remote…',
      ariaLabel: `Delete tag ${name} on remote`,
      icon: CloudOff,
      run: () => void deleteRemoteTag(name),
      disabled: busy,
      destructive: true,
    },
    {
      label: 'Delete…',
      ariaLabel: `Delete tag ${name}`,
      icon: X,
      run: () => void deleteTag(name),
      disabled: busy,
      destructive: true,
    },
  ]

  const toggleNewTag = () => {
    setShowNewTag((v) => !v)
    sections.open('tags')
  }

  return (
    <div className="tags-panel">
      <SidebarSection
        label="Tags"
        count={tags?.length}
        open={sections.isOpen('tags', hasTags)}
        onToggle={() => sections.toggle('tags', hasTags)}
        actions={
          <>
            <button
              type="button"
              className="tags-panel-add"
              onClick={() => setReadiness(true)}
              aria-label="Release readiness"
              title="Release readiness"
            >
              <ClipboardCheck size={18} strokeWidth={1.75} />
            </button>
            <button
              type="button"
              className="tags-panel-add"
              onClick={toggleNewTag}
              aria-label="New tag"
              title="New tag"
            >
              <Plus size={18} strokeWidth={1.75} />
            </button>
          </>
        }
        pinned={
          showNewTag && (
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
          )
        }
      >
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
                  onContextMenu={rowMenu.open(tagActions(t.name))}
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
                  <RowActionButtons actions={tagActions(t.name)} className="tag-row-actions" />
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
      </SidebarSection>
      {rowMenu.menu}
      {readiness && <ReleaseReadinessDialog repoPath={repoPath} onClose={() => setReadiness(false)} />}
      {releaseTag && <PublishReleaseDialog repoPath={repoPath} tag={releaseTag} onClose={() => setReleaseTag(null)} />}
    </div>
  )
}

export default TagsPanel
