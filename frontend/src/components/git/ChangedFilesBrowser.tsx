import { useEffect, useState } from 'react'
import type { ChangedFile, FileDiff } from '@current-client-bindings/app'
import ChangedFileRow from '../../features/history/ChangedFileRow'
import { useReviewThreads } from '../../lib/useReviewThreads'
import type { ImageSources } from '../../features/diff/ImageDiff'
import ChangedFileDiff from './ChangedFileDiff'
import './ChangedFilesBrowser.scss'

interface ChangedFilesBrowserProps {
  files: ChangedFile[] | null
  error: string | null
  emptyHint: string
  // force loads a file over the large-file limit.
  loadDiff: (file: ChangedFile, force: boolean, ignoreWhitespace: boolean) => Promise<FileDiff>
  // Where to read a file's two versions for the image view.
  images?: (file: ChangedFile) => ImageSources
  // The file to show first, when the list is already loaded.
  initialPath?: string
  // Lets a submodule's change list the commits between its versions.
  repoPath?: string
}

// Remount it (with a key) when what it shows changes, so the selection resets.
function ChangedFilesBrowser({
  files,
  error,
  emptyHint,
  loadDiff,
  images,
  initialPath,
  repoPath,
}: ChangedFilesBrowserProps) {
  const reviewThreads = useReviewThreads(repoPath ?? '', files)
  const [selected, setSelected] = useState<ChangedFile | null>(() => files?.find((f) => f.path === initialPath) ?? null)
  // The first file (or the one asked for) opens with the list, so the sheet is useful at once.
  useEffect(() => {
    if (selected === null && files && files.length > 0) {
      setSelected(files.find((f) => f.path === initialPath) ?? files[0])
    }
  }, [files, initialPath, selected])

  return (
    <div className="changed-files-browser">
      <div className="changed-files-browser-files">
        <div className="changed-files-browser-header">
          <span className="changed-files-browser-label">Changed files</span>
          {files && <span className="changed-files-browser-count">{files.length}</span>}
        </div>
        {error ? (
          <p className="changed-files-browser-error">Could not load changed files: {error}</p>
        ) : files === null ? (
          <p className="changed-files-browser-hint">Loading changed files…</p>
        ) : files.length === 0 ? (
          <p className="changed-files-browser-hint">{emptyHint}</p>
        ) : (
          <ul className="changed-file-list">
            {files.map((file) => (
              <ChangedFileRow
                key={file.path}
                file={file}
                selected={selected?.path === file.path}
                onSelect={() => setSelected(file)}
                threads={reviewThreads.counts[file.path]}
              />
            ))}
          </ul>
        )}
      </div>

      <div className="changed-files-browser-diff">
        {!selected ? (
          <p className="changed-files-browser-hint">Select a file to view its diff.</p>
        ) : (
          <ChangedFileDiff
            file={selected}
            loadDiff={loadDiff}
            images={images?.(selected)}
            repoPath={repoPath}
            reviewThreads={reviewThreads}
          />
        )}
      </div>
    </div>
  )
}

export default ChangedFilesBrowser
