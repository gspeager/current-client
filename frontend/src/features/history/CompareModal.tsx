import { useState } from 'react'
import { CompareService, DiffService } from '@current-client-bindings/app'
import ChangedFileRow from './ChangedFileRow'
import DiffViewer from '../diff/DiffViewer'
import DiffViewModeToggle, { type DiffViewMode } from '../diff/DiffViewModeToggle'
import RefSelect, { useRefGroups } from '../../components/git/RefSelect'
import { useAsyncData } from '../../lib/useAsyncData'
import Modal from '../../components/chrome/Modal'
import './CompareModal.scss'

interface CompareModalProps {
  repoPath: string
  initialFromRef: string
  initialToRef: string
  onClose: () => void
}

function CompareModal({ repoPath, initialFromRef, initialToRef, onClose }: CompareModalProps) {
  const [fromRef, setFromRef] = useState(initialFromRef)
  const [toRef, setToRef] = useState(initialToRef)
  const [selectedPath, setSelectedPath] = useState<string | null>(null)
  const [viewMode, setViewMode] = useState<DiffViewMode>('split')

  const refGroups = useRefGroups(repoPath)
  const { data: files, error: filesError } = useAsyncData(
    () => CompareService.GetChangedFiles(repoPath, fromRef, toRef),
    [repoPath, fromRef, toRef],
  )
  const { data: diff, error: diffError } = useAsyncData(
    () => (selectedPath ? DiffService.GetRefDiff(repoPath, selectedPath, fromRef, toRef, false) : null),
    [repoPath, selectedPath, fromRef, toRef],
  )

  return (
    <Modal
      title="Compare"
      onClose={onClose}
      className="compare-panel"
      headerContent={
        <div className="compare-refs">
          <RefSelect
            groups={refGroups}
            value={fromRef}
            onChange={(ref) => {
              setFromRef(ref)
              setSelectedPath(null)
            }}
            label="Compare from"
          />
          <span className="compare-refs-sep">…</span>
          <RefSelect
            groups={refGroups}
            value={toRef}
            onChange={(ref) => {
              setToRef(ref)
              setSelectedPath(null)
            }}
            label="Compare to"
          />
        </div>
      }
    >
      <div className="compare-body">
        <div className="compare-files">
          <div className="compare-files-header">
            <span className="compare-label">Changed files</span>
            {files && <span className="compare-files-count">{files.length}</span>}
          </div>
          {filesError ? (
            <p className="compare-error">Could not load changed files: {filesError}</p>
          ) : files === null ? (
            <p className="compare-hint">Loading changed files…</p>
          ) : files.length === 0 ? (
            <p className="compare-hint">No differences between these refs.</p>
          ) : (
            <ul className="changed-file-list">
              {files.map((file) => (
                <ChangedFileRow
                  key={file.path}
                  file={file}
                  selected={selectedPath === file.path}
                  onSelect={() => setSelectedPath(file.path)}
                />
              ))}
            </ul>
          )}
        </div>

        <div className="compare-diff">
          {!selectedPath ? (
            <p className="compare-hint">Select a file to view its diff.</p>
          ) : diffError ? (
            <p className="compare-error">Could not load diff: {diffError}</p>
          ) : diff ? (
            <>
              <div className="compare-diff-toolbar">
                <DiffViewModeToggle value={viewMode} onChange={setViewMode} />
              </div>
              <DiffViewer
                diff={diff}
                path={selectedPath}
                viewMode={viewMode}
                images={{
                  repoPath,
                  before: { kind: 'commit', rev: fromRef },
                  after: { kind: 'commit', rev: toRef },
                }}
              />
            </>
          ) : (
            <p className="compare-hint">Loading diff…</p>
          )}
        </div>
      </div>
    </Modal>
  )
}

export default CompareModal
