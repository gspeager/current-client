import { GitService } from '@current-client-bindings/app'
import { useAsyncData } from '../../lib/useAsyncData'
import type { LfsChange } from './lfsPointer'

function formatSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
}

function LfsDiff({ change }: { change: LfsChange }) {
  const { oldSize, newSize } = change
  const { data: installed } = useAsyncData(() => GitService.LFSInstalled(), [])

  return (
    <div className="lfs-diff">
      <p className="lfs-diff-summary">
        Stored in Git LFS{' · '}
        <span className="lfs-diff-sizes">
          {oldSize !== null && newSize !== null
            ? oldSize === newSize
              ? `${formatSize(newSize)}, contents changed`
              : `${formatSize(oldSize)} → ${formatSize(newSize)}`
            : newSize !== null
              ? `added, ${formatSize(newSize)}`
              : oldSize !== null && `removed, was ${formatSize(oldSize)}`}
        </span>
      </p>
      {installed === false && (
        <p className="lfs-diff-note">
          Git LFS isn't installed, so files stored in it are only small pointer files on disk. Install Git LFS to get
          their contents.
        </p>
      )}
    </div>
  )
}

export default LfsDiff
