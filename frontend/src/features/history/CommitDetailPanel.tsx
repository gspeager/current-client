import type { KeyboardEvent, MouseEvent } from 'react'
import { Check, ChevronDown, Copy, GitCommitVertical, Palette, ShieldAlert, ShieldCheck } from 'lucide-react'
import {
  type ChangedFile,
  type CommitInfo,
  HistoryService,
  type IdentityInfo,
  PlatformService,
} from '@current-client-bindings/app'
import type { Links } from '@current-client-bindings/core/hostlink'
import { relativeTime } from '../../lib/relativeTime'
import { useClockTick } from '../../lib/useClockTick'
import { useAsyncData } from '../../lib/useAsyncData'
import { PRIMARY_COMMIT_ACTIONS, type CommitAction } from './useCommitActions'
import { useCopyToClipboard } from '../../lib/useCopyToClipboard'
import { plural } from '../../lib/plural'
import { useDismissibleDetails } from '../../lib/useDismissibleDetails'
import { useFileTools } from './useFileTools'
import ChangedFileRow from './ChangedFileRow'
import IdentityBadge from '../../components/git/IdentityBadge'
import ReferenceText from '../../components/git/ReferenceText'
import type { ReferenceCards } from '../../lib/useReferenceCards'
import { filePage } from '../../lib/hostPages'
import './CommitDetailPanel.scss'

interface CommitDetailPanelProps {
  repoPath: string
  commit: CommitInfo
  identity?: IdentityInfo
  hostLinks?: Links | null
  referenceCards?: ReferenceCards
  colorError: string | null
  onColorChange: (color: string) => void
  // The commit's actions, in order; the first few are buttons, the rest under More.
  actions: CommitAction[]
  actionError?: string | null
  // The file whose diff History shows in place of the graph.
  openedPath?: string | null
  onOpenFile?: (path: string) => void
  // Unresolved review threads per file.
  threadCounts?: Record<string, number | undefined>
}

function CommitDetailPanel({
  repoPath,
  commit,
  identity,
  hostLinks = null,
  referenceCards,
  colorError,
  onColorChange,
  actions,
  actionError = null,
  openedPath = null,
  onOpenFile,
  threadCounts,
}: CommitDetailPanelProps) {
  useClockTick()
  const moreRef = useDismissibleDetails()
  const { copied: shaCopied, copy } = useCopyToClipboard()
  const fileTools = useFileTools(repoPath)

  const { data: changedFiles, error: changedFilesError } = useAsyncData(
    () => HistoryService.GetChangedFiles(repoPath, commit.sha),
    [repoPath, commit.sha],
  )
  const { data: branches } = useAsyncData(
    () => HistoryService.BranchesContaining(repoPath, commit.sha),
    [repoPath, commit.sha],
  )

  const runMore = (run: () => void) => {
    if (moreRef.current) moreRef.current.open = false
    run()
  }

  const fileContextMenu = (f: ChangedFile) => (e: MouseEvent | KeyboardEvent) =>
    fileTools.openMenu(e, [
      ...fileTools.pathItems(f.path),
      ...(hostLinks?.blob
        ? [
            {
              label: 'Open on host',
              onClick: () => void PlatformService.OpenWebPage(filePage(hostLinks, commit.sha, f.path)),
            },
          ]
        : []),
    ])

  return (
    <section className="commit-detail">
      <div className="commit-detail-scroll">
        <div className="commit-detail-header">
          <GitCommitVertical size={16} strokeWidth={1.75} className="commit-detail-title-icon" />
          <span className="commit-detail-title">Commit detail</span>
          <span className="commit-detail-sha">{commit.sha.slice(0, 7)}</span>
          <button
            type="button"
            className="commit-detail-copy-sha"
            onClick={() => copy(commit.sha)}
            aria-label="Copy full commit hash"
            title="Copy full commit hash"
          >
            {shaCopied ? <Check size={14} strokeWidth={1.75} /> : <Copy size={14} strokeWidth={1.75} />}
          </button>
        </div>

        <div className="commit-detail-author">
          {identity && (
            <IdentityBadge identity={identity} name={commit.authorName} email={commit.authorEmail} size={32} />
          )}
          <div className="commit-detail-author-info">
            <span className="commit-detail-author-name">
              {commit.authorName}
              {commit.signature === 'verified' && (
                <span className="commit-detail-signature commit-detail-signature-verified">
                  <ShieldCheck size={12} strokeWidth={1.75} /> Verified
                </span>
              )}
              {commit.signature === 'unverified' && (
                <span className="commit-detail-signature commit-detail-signature-unverified">
                  <ShieldAlert size={12} strokeWidth={1.75} /> Unverified
                </span>
              )}
            </span>
            <span className="commit-detail-author-email">{commit.authorEmail}</span>
          </div>
          {identity && (
            <label className="commit-detail-color" title="Change the author's badge color">
              <Palette size={14} strokeWidth={1.75} />
              <span className="commit-detail-color-swatch" style={{ background: identity.color }} />
              <input
                type="color"
                aria-label="Badge color"
                value={identity.color}
                onChange={(e) => onColorChange(e.target.value)}
              />
            </label>
          )}
        </div>
        {colorError && <p className="commit-detail-error">Could not set color: {colorError}</p>}

        <div className="commit-detail-stats">
          <div>
            <span className="commit-detail-stat-label">Committed</span>
            <span>{relativeTime(new Date(commit.date))}</span>
          </div>
          <div>
            <span className="commit-detail-stat-label">Parent</span>
            <span className="commit-detail-parent-sha">
              {commit.parentShas.length > 0
                ? commit.parentShas.map((p) => p.slice(0, 7)).join(', ')
                : 'None (root commit)'}
            </span>
          </div>
          <div>
            <span className="commit-detail-stat-label">Changes</span>
            <span>{changedFiles ? plural(changedFiles.length, 'file') : '…'}</span>
          </div>
          <div>
            <span className="commit-detail-stat-label">On branches</span>
            <span className="commit-detail-branches" title={branches?.join(', ')}>
              {branches === null ? '…' : branches.length === 0 ? 'None' : branches.join(', ')}
            </span>
          </div>
          {commit.coAuthors.length > 0 && (
            <div className="commit-detail-coauthors">
              <span className="commit-detail-stat-label">Co-authors</span>
              <span>
                {commit.coAuthors.map((a, i) => (
                  <span key={`${a.name}${a.email}`} title={a.email || undefined}>
                    {i > 0 && ', '}
                    {a.name}
                  </span>
                ))}
              </span>
            </div>
          )}
        </div>

        <p className="commit-detail-subject">
          <ReferenceText text={commit.subject} links={hostLinks} cards={referenceCards} />
        </p>
        {commit.body && (
          <p className="commit-detail-body">
            <ReferenceText text={commit.body} links={hostLinks} cards={referenceCards} />
          </p>
        )}

        <div className="commit-detail-section-header">
          <span className="commit-detail-label">Changed files</span>
          {changedFiles && <span className="commit-detail-section-count">{plural(changedFiles.length, 'file')}</span>}
        </div>
        {commit.parentShas.length > 1 && (
          <p className="commit-detail-hint">Compared with the first parent, {commit.parentShas[0].slice(0, 7)}.</p>
        )}
        {changedFilesError ? (
          <p className="commit-detail-error">Could not load changed files: {changedFilesError}</p>
        ) : changedFiles === null ? (
          <p className="commit-detail-hint">Loading changed files…</p>
        ) : changedFiles.length === 0 ? (
          <p className="commit-detail-hint">No file changes.</p>
        ) : (
          <ul className="changed-file-list">
            {changedFiles.map((f) => (
              <ChangedFileRow
                key={f.path}
                file={f}
                selected={openedPath === f.path}
                onSelect={() => onOpenFile?.(f.path)}
                onContextMenu={fileContextMenu(f)}
                threads={threadCounts?.[f.path]}
              />
            ))}
          </ul>
        )}
      </div>

      <div className="commit-detail-actions-footer">
        <div className="commit-detail-section-header">
          <span className="commit-detail-label">Actions</span>
        </div>
        <div className="commit-detail-actions">
          {actions.slice(0, PRIMARY_COMMIT_ACTIONS).map(({ id, label, icon: Icon, run }) => (
            <button key={id} type="button" onClick={run}>
              <Icon size={14} strokeWidth={1.75} />
              {label}
            </button>
          ))}
          {actions.length > PRIMARY_COMMIT_ACTIONS && (
            <details ref={moreRef} className="commit-detail-more">
              <summary className="commit-detail-more-toggle" aria-label="More actions" title="More actions">
                More
                <ChevronDown size={14} strokeWidth={1.75} />
              </summary>
              <div className="commit-detail-more-menu">
                {actions.slice(PRIMARY_COMMIT_ACTIONS).map(({ id, label, icon: Icon, run }) => (
                  <button key={id} type="button" onClick={() => runMore(run)}>
                    <Icon size={14} strokeWidth={1.75} />
                    {label}
                  </button>
                ))}
              </div>
            </details>
          )}
        </div>
        {actionError && <p className="commit-detail-error">{actionError}</p>}
      </div>

      {fileTools.overlays}
    </section>
  )
}

export default CommitDetailPanel
