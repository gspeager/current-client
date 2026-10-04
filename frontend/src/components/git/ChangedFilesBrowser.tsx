import { useState } from 'react'
import type { ChangedFile, FileDiff } from '@current-client-bindings/app'
import ChangedFileRow from '../../features/history/ChangedFileRow'
import DiffViewer from '../../features/diff/DiffViewer'
import type { ImageSources } from '../../features/diff/ImageDiff'
import DiffViewModeToggle, { type DiffViewMode } from '../../features/diff/DiffViewModeToggle'
import { useAsyncData } from '../../lib/useAsyncData'
import './ChangedFilesBrowser.scss'

interface ChangedFilesBrowserProps {
  files: ChangedFile[] | null
  error: string | null
  emptyHint: string
  loadDiff: (file: ChangedFile) => Promise<FileDiff>
  images?: ImageSources
}

// Remount it (with a key) when what it shows changes, so the selection resets.
function ChangedFilesBrowser({ files, error, emptyHint, loadDiff, images }: ChangedFilesBrowserProps) {
  const [selected, setSelected] = useState<ChangedFile | null>(null)
  const [viewMode, setViewMode] = useState<DiffViewMode>('split')
  const { data: diff, error: diffError } = useAsyncData(
    () => (selected ? loadDiff(selected) : null),
    // loadDiff is a new function every render; the key above covers what it depends on.
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [selected],
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
              <DiffViewModeToggle value={viewMode} onChange={setViewMode} />
            </div>
            <DiffViewer diff={diff} path={selected.path} viewMode={viewMode} images={images} />
          </>
        ) : (
          <p className="changed-files-browser-hint">Loading diff…</p>
        )}
      </div>
    </div>
  )
}

export default ChangedFilesBrowser
