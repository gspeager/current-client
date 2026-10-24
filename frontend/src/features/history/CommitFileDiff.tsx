import { useState } from 'react'
import { ArrowLeft } from 'lucide-react'
import { type CommitInfo, DiffService } from '@current-client-bindings/app'
import PathText from '../../components/git/PathText'
import DiffViewer from '../diff/DiffViewer'
import DiffViewMenu from '../diff/DiffViewMenu'
import DiffViewModeToggle, { type DiffViewMode } from '../diff/DiffViewModeToggle'
import { diffBaseFor } from '../diff/diffMapping'
import { useAsyncData } from '../../lib/useAsyncData'
import type { ReviewThreadsView } from '../../lib/useReviewThreads'
import './CommitFileDiff.scss'

interface CommitFileDiffProps {
  repoPath: string
  commit: CommitInfo
  path: string
  reviewThreads: ReviewThreadsView
  onBack: () => void
}

// A commit's file, in the main column in place of the graph; the inspector stays.
function CommitFileDiff({ repoPath, commit, path, reviewThreads, onBack }: CommitFileDiffProps) {
  const base = diffBaseFor(commit.parentShas)
  const [viewMode, setViewMode] = useState<DiffViewMode>('split')
  const [ignoreWhitespace, setIgnoreWhitespace] = useState(false)
  const [forced, setForced] = useState(false)
  const { data: diff, error } = useAsyncData(
    () => DiffService.GetRefDiff(repoPath, path, base, commit.sha, forced, ignoreWhitespace),
    [repoPath, path, base, commit.sha, forced, ignoreWhitespace],
  )

  return (
    <div className="commit-file-diff">
      <div className="commit-file-diff-header">
        <button type="button" className="commit-file-diff-back" onClick={onBack}>
          <ArrowLeft size={14} strokeWidth={1.75} />
          History
        </button>
        <PathText path={path} className="commit-file-diff-path" />
        <span className="commit-file-diff-sha">{commit.sha.slice(0, 7)}</span>
        <span className="commit-file-diff-spacer" />
        <DiffViewModeToggle value={viewMode} onChange={setViewMode} />
        <DiffViewMenu ignoreWhitespace={ignoreWhitespace} onIgnoreWhitespaceChange={setIgnoreWhitespace} />
      </div>
      <div className="commit-file-diff-body">
        {error ? (
          <p className="commit-file-diff-hint">Could not load diff: {error}</p>
        ) : diff ? (
          <DiffViewer
            diff={diff}
            path={path}
            viewMode={viewMode}
            baseSide={{ label: 'Parent', sha: base.slice(0, 7) }}
            targetSide={{ label: 'Commit', sha: commit.sha.slice(0, 7) }}
            threads={reviewThreads.byPath[path]}
            outdatedThreads={reviewThreads.outdatedByPath[path]}
            onForceLoad={() => setForced(true)}
            repoPath={repoPath}
            images={{
              repoPath,
              before: { kind: 'commit', rev: base },
              after: { kind: 'commit', rev: commit.sha },
            }}
          />
        ) : (
          <p className="commit-file-diff-hint">Loading diff…</p>
        )}
      </div>
    </div>
  )
}

export default CommitFileDiff
