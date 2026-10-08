import { useState } from 'react'
import { DiffService, HistoryService } from '@current-client-bindings/app'
import { diffBaseFor } from '../diff/diffMapping'
import { relativeTime } from '../../lib/relativeTime'
import { useClockTick } from '../../lib/useClockTick'
import { useAsyncData } from '../../lib/useAsyncData'
import Modal from '../../components/chrome/Modal'
import DiffViewer from '../diff/DiffViewer'
import PathText from '../../components/git/PathText'
import './FileHistoryPanel.scss'

interface FileHistoryPanelProps {
  repoPath: string
  path: string
  onClose: () => void
}

function FileHistoryPanel({ repoPath, path, onClose }: FileHistoryPanelProps) {
  useClockTick()
  const { data: commits, error } = useAsyncData(
    () => HistoryService.GetFileHistory(repoPath, path, 200, 0),
    [repoPath, path],
  )
  const [selectedSha, setSelectedSha] = useState<string | null>(null)
  const selectedCommit = commits?.find((c) => c.sha === selectedSha)
  const [forcedSha, setForcedSha] = useState<string | null>(null)
  const force = selectedSha !== null && forcedSha === selectedSha
  const { data: diff, error: diffError } = useAsyncData(
    () =>
      selectedCommit
        ? DiffService.GetRefDiff(
            repoPath,
            path,
            diffBaseFor(selectedCommit.parentShas),
            selectedCommit.sha,
            force,
            false,
          )
        : null,
    [repoPath, path, selectedCommit, force],
  )

  return (
    <Modal
      title="File history"
      onClose={onClose}
      className="file-history-panel"
      headerContent={<PathText path={path} className="modal-subtitle" />}
    >
      <div className="file-history-body">
        <div className="file-history-list">
          {error ? (
            <p className="file-history-hint">Could not load history: {error}</p>
          ) : commits === null ? (
            <p className="file-history-hint">Loading history…</p>
          ) : commits.length === 0 ? (
            <p className="file-history-hint">No history for this file.</p>
          ) : (
            commits.map((c) => (
              <button
                type="button"
                key={c.sha}
                className={c.sha === selectedSha ? 'file-history-row file-history-row-selected' : 'file-history-row'}
                onClick={() => setSelectedSha(c.sha)}
              >
                <span className="file-history-row-subject">{c.subject}</span>
                <span className="file-history-row-author">{c.authorName}</span>
                <span className="file-history-row-sha">{c.sha.slice(0, 7)}</span>
                <span className="file-history-row-time">{relativeTime(new Date(c.date))}</span>
              </button>
            ))
          )}
        </div>

        <div className="file-history-diff">
          {!selectedSha ? (
            <p className="file-history-hint">Select a commit to see its diff.</p>
          ) : diffError ? (
            <p className="file-history-hint">Could not load diff: {diffError}</p>
          ) : diff ? (
            <DiffViewer
              diff={diff}
              path={path}
              viewMode="unified"
              repoPath={repoPath}
              onForceLoad={() => setForcedSha(selectedSha)}
              images={
                selectedCommit && {
                  repoPath,
                  before: { kind: 'commit', rev: diffBaseFor(selectedCommit.parentShas) },
                  after: { kind: 'commit', rev: selectedCommit.sha },
                }
              }
            />
          ) : (
            <p className="file-history-hint">Loading diff…</p>
          )}
        </div>
      </div>
    </Modal>
  )
}

export default FileHistoryPanel
