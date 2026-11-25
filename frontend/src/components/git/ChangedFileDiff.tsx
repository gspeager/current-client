import { useState, type ReactNode } from 'react'
import type { ChangedFile, FileDiff } from '@current-client-bindings/app'
import DiffViewer, { type DiffFocus } from '../../features/diff/DiffViewer'
import DiffViewMenu from '../../features/diff/DiffViewMenu'
import DiffViewModeToggle, { type DiffViewMode } from '../../features/diff/DiffViewModeToggle'
import type { ImageSources } from '../../features/diff/ImageDiff'
import type { LineNotes } from '../../features/review/ReviewNoteBlock'
import { useAsyncData } from '../../lib/useAsyncData'
import type { ReviewThreadsView } from '../../lib/useReviewThreads'

interface ChangedFileDiffProps {
  file: ChangedFile
  // force loads a file over the large-file limit.
  loadDiff: (file: ChangedFile, force: boolean, ignoreWhitespace: boolean) => Promise<FileDiff>
  // Changes when the file's diff would.
  version?: unknown
  images?: ImageSources
  // Lets a submodule's change list the commits between its versions.
  repoPath?: string
  reviewThreads?: ReviewThreadsView
  notes?: LineNotes
  // A line to open at; a new object asks again.
  focusLine?: DiffFocus | null
  toolbar?: ReactNode
}

// The half of a changed-files sheet that shows the selected file.
function ChangedFileDiff({
  file,
  loadDiff,
  version,
  images,
  repoPath,
  reviewThreads,
  notes,
  focusLine,
  toolbar,
}: ChangedFileDiffProps) {
  const [viewMode, setViewMode] = useState<DiffViewMode>('split')
  const [ignoreWhitespace, setIgnoreWhitespace] = useState(false)
  const [forcedPath, setForcedPath] = useState<string | null>(null)
  const force = forcedPath === file.path
  const { data: diff, error } = useAsyncData(
    () => loadDiff(file, force, ignoreWhitespace),
    // loadDiff is a new function every render; the path covers what it depends on.
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [file.path, force, ignoreWhitespace],
    { refreshKey: version },
  )
  if (error) return <p className="changed-files-browser-error">Could not load diff: {error}</p>
  if (!diff) return <p className="changed-files-browser-hint">Loading diff…</p>
  return (
    <>
      <div className="changed-files-browser-toolbar">
        {toolbar}
        <DiffViewModeToggle value={viewMode} onChange={setViewMode} />
        <DiffViewMenu ignoreWhitespace={ignoreWhitespace} onIgnoreWhitespaceChange={setIgnoreWhitespace} />
      </div>
      <DiffViewer
        diff={diff}
        path={file.path}
        viewMode={viewMode}
        threads={reviewThreads?.byPath[file.path]}
        outdatedThreads={reviewThreads?.outdatedByPath[file.path]}
        review={
          repoPath && reviewThreads?.number
            ? { repoPath, number: reviewThreads.number, prefix: reviewThreads.prefix }
            : undefined
        }
        images={images}
        repoPath={repoPath}
        notes={notes}
        focusLine={focusLine}
        onForceLoad={() => setForcedPath(file.path)}
      />
    </>
  )
}

export default ChangedFileDiff
