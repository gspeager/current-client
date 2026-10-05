import { useState, type MouseEvent } from 'react'
import {
  Check,
  Cherry,
  Copy,
  GitBranchPlus,
  GitCommitVertical,
  RotateCcw,
  ShieldAlert,
  ShieldCheck,
} from 'lucide-react'
import {
  DiffService,
  HistoryService,
  type ChangedFile,
  type IdentityInfo,
  type CommitInfo,
} from '@current-client-bindings/app'
import { diffBaseFor } from '../diff/diffMapping'
import { relativeTime } from '../../lib/relativeTime'
import { useAsyncData } from '../../lib/useAsyncData'
import { useCommitActions } from './useCommitActions'
import { useCopyToClipboard } from '../../lib/useCopyToClipboard'
import { useFileTools } from './useFileTools'
import ChangedFileRow from './ChangedFileRow'
import DiffViewer from '../diff/DiffViewer'
import DiffViewModeToggle, { type DiffViewMode } from '../diff/DiffViewModeToggle'
import IdentityBadge from '../../components/git/IdentityBadge'
import './CommitDetailPanel.scss'

interface CommitDetailPanelProps {
  repoPath: string
  commit: CommitInfo
  identity?: IdentityInfo
  colorError: string | null
  onColorChange: (color: string) => void
  onCreateBranchHere: () => void
  onBranchChanged?: () => void
}

function CommitDetailPanel({
  repoPath,
  commit,
  identity,
  colorError,
  onColorChange,
  onCreateBranchHere,
  onBranchChanged,
}: CommitDetailPanelProps) {
  const { cherryPick, revert, checkoutCommit, actionError } = useCommitActions(repoPath, onBranchChanged)
  // A selection only counts for the commit it was made in.
  const [selection, setSelection] = useState<{ sha: string; path: string } | null>(null)
  const selectedFilePath = selection?.sha === commit.sha ? selection.path : null
  const [viewMode, setViewMode] = useState<DiffViewMode>('split')
  const { copied: shaCopied, copy } = useCopyToClipboard()
  const fileTools = useFileTools(repoPath)

  const { data: changedFiles, error: changedFilesError } = useAsyncData(
    () => HistoryService.GetChangedFiles(repoPath, commit.sha),
    [repoPath, commit.sha],
  )
  const { data: fileDiff, error: fileDiffError } = useAsyncData(
    () =>
      selectedFilePath
        ? DiffService.GetRefDiff(repoPath, selectedFilePath, diffBaseFor(commit.parentShas), commit.sha, false)
        : null,
    [repoPath, selectedFilePath, commit.sha, commit.parentShas],
  )

  const fileContextMenu = (f: ChangedFile) => (e: MouseEvent) => fileTools.openMenu(e, fileTools.pathItems(f.path))

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
            <input
              type="color"
              className="commit-detail-color"
              aria-label="Badge color"
              value={identity.color}
              onChange={(e) => onColorChange(e.target.value)}
            />
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

        <p className="commit-detail-subject">{commit.subject}</p>
        {commit.body && <p className="commit-detail-body">{commit.body}</p>}

        <div className="commit-detail-section-header">
          <span className="commit-detail-label">Changed files</span>
          {changedFiles && <span className="commit-detail-section-count">{changedFiles.length} files</span>}
        </div>
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
                selected={selectedFilePath === f.path}
                onSelect={() => setSelection({ sha: commit.sha, path: f.path })}
                onContextMenu={fileContextMenu(f)}
              />
            ))}
          </ul>
        )}

        {selectedFilePath && (
          <div className="commit-detail-diff">
            {fileDiffError ? (
              <p className="commit-detail-error">Could not load diff: {fileDiffError}</p>
            ) : fileDiff ? (
              <>
                <div className="commit-detail-diff-toolbar">
                  <DiffViewModeToggle value={viewMode} onChange={setViewMode} />
                </div>
                <DiffViewer
                  diff={fileDiff}
                  path={selectedFilePath}
                  viewMode={viewMode}
                  repoPath={repoPath}
                  images={{
                    repoPath,
                    before: { kind: 'commit', rev: diffBaseFor(commit.parentShas) },
                    after: { kind: 'commit', rev: commit.sha },
                  }}
                />
              </>
            ) : (
              <p className="commit-detail-hint">Loading diff…</p>
            )}
          </div>
        )}
      </div>

      <div className="commit-detail-actions-footer">
        <div className="commit-detail-section-header">
          <span className="commit-detail-label">Actions</span>
        </div>
        <div className="commit-detail-actions">
          <button type="button" onClick={onCreateBranchHere}>
            <GitBranchPlus size={14} strokeWidth={1.75} />
            Branch here
          </button>
          <button type="button" onClick={() => void checkoutCommit(commit.sha)}>
            <GitCommitVertical size={14} strokeWidth={1.75} />
            Check out
          </button>
          <button type="button" onClick={() => cherryPick(commit.sha)}>
            <Cherry size={14} strokeWidth={1.75} />
            Cherry-pick
          </button>
          <button type="button" onClick={() => revert(commit.sha)}>
            <RotateCcw size={14} strokeWidth={1.75} />
            Revert
          </button>
        </div>
        {actionError && <p className="commit-detail-error">Could not apply: {actionError}</p>}
      </div>

      {fileTools.overlays}
    </section>
  )
}

export default CommitDetailPanel
