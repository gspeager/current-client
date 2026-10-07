import { useState } from 'react'
import type { ChangedFile, FileDiff } from '@current-client-bindings/app'
import ChangedFileRow from '../../features/history/ChangedFileRow'
import DiffViewer from '../../features/diff/DiffViewer'
import { DiffWrapToggle } from '../../features/diff/diffWrap'
import type { ImageSources } from '../../features/diff/ImageDiff'
import DiffViewModeToggle, { type DiffViewMode } from '../../features/diff/DiffViewModeToggle'
import { useAsyncData } from '../../lib/useAsyncData'
import './ChangedFilesBrowser.scss'

interface ChangedFilesBrowserProps {
  files: ChangedFile[] | null
  error: string | null
  emptyHint: string
  // force loads a file over the large-file limit.
  loadDiff: (file: ChangedFile, force: boolean) => Promise<FileDiff>
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
  const [selected, setSelected] = useState<ChangedFile | null>(() => files?.find((f) => f.path === initialPath) ?? null)
  const [viewMode, setViewMode] = useState<DiffViewMode>('split')
  const [forcedPath, setForcedPath] = useState<string | null>(null)
  const force = selected !== null && forcedPath === selected.path
  const { data: diff, error: diffError } = useAsyncData(
    () => (selected ? loadDiff(selected, force) : null),
    // loadDiff is a new function every render; the key above covers what it depends on.
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [selected, force],
  )

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
              />
            ))}
          </ul>
        )}
      </div>

      <div className="changed-files-browser-diff">
        {!selected ? (
          <p className="changed-files-browser-hint">Select a file to view its diff.</p>
        ) : diffError ? (
          <p className="changed-files-browser-error">Could not load diff: {diffError}</p>
        ) : diff ? (
          <>
            <div className="changed-files-browser-toolbar">
              <DiffWrapToggle />
              <DiffViewModeToggle value={viewMode} onChange={setViewMode} />
            </div>
            <DiffViewer
              diff={diff}
              path={selected.path}
              viewMode={viewMode}
              images={images?.(selected)}
              repoPath={repoPath}
              onForceLoad={() => setForcedPath(selected.path)}
            />
          </>
        ) : (
          <p className="changed-files-browser-hint">Loading diff…</p>
        )}
      </div>
    </div>
  )
}

export default ChangedFilesBrowser
