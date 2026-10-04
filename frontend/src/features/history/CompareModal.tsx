import { useState } from 'react'
import { CompareService, DiffService } from '@current-client-bindings/app'
import ChangedFilesBrowser from '../../components/git/ChangedFilesBrowser'
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

  const refGroups = useRefGroups(repoPath)
  const { data: files, error } = useAsyncData(
    () => CompareService.GetChangedFiles(repoPath, fromRef, toRef),
    [repoPath, fromRef, toRef],
  )

  return (
    <Modal
      title="Compare"
      onClose={onClose}
      className="compare-panel"
      headerContent={
        <div className="compare-refs">
          <RefSelect groups={refGroups} value={fromRef} onChange={setFromRef} label="Compare from" />
          <span className="compare-refs-sep">…</span>
          <RefSelect groups={refGroups} value={toRef} onChange={setToRef} label="Compare to" />
        </div>
      }
    >
      <ChangedFilesBrowser
        key={`${fromRef}…${toRef}`}
        files={files}
        error={error}
        emptyHint="No differences between these refs."
        images={{
          repoPath,
          before: { kind: 'commit', rev: fromRef },
          after: { kind: 'commit', rev: toRef },
        }}
        loadDiff={(file) => DiffService.GetRefDiff(repoPath, file.path, fromRef, toRef, false)}
      />
    </Modal>
  )
}

export default CompareModal
