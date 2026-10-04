import { useEffect, useRef, useState } from 'react'
import { FileCode, Image } from 'lucide-react'
import type { FileDiff } from '@current-client-bindings/app'
import SegmentedControl from '../../components/controls/SegmentedControl'
import ImageDiff, { type ImageSources } from './ImageDiff'
import { imageMimeType } from './imageFiles'
import { buildDisplayRows, diffStats } from './diffRows'
import DiffPane, { type HunkActions } from './DiffPane'
import './DiffViewer.scss'
import { useWindowKeydown } from '../../lib/useWindowKeydown'

function formatMegabytes(bytes: number): string {
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
}

interface DiffSideLabel {
  label: string
  sha?: string
}

interface DiffViewerProps extends HunkActions {
  diff: FileDiff
  path: string
  viewMode: 'split' | 'unified'
  baseSide?: DiffSideLabel
  targetSide?: DiffSideLabel
  onForceLoad?: () => void
  // Where to read the two versions of an image from; without it images show as binary.
  images?: ImageSources
}

function DiffViewer({
  diff,
  path,
  viewMode,
  baseSide,
  targetSide,
  onForceLoad,
  images,
  ...hunkActions
}: DiffViewerProps) {
  const [showImage, setShowImage] = useState(true)
  const paneRef = useRef<HTMLDivElement>(null)
  const [expandedGroups, setExpandedGroups] = useState<Set<number>>(new Set())
  const [currentHunkIndex, setCurrentHunkIndex] = useState(0)

  useEffect(() => {
    setExpandedGroups(new Set())
    setCurrentHunkIndex(0)
  }, [diff])

  useWindowKeydown((e) => {
    if (!e.altKey || (e.key !== 'ArrowDown' && e.key !== 'ArrowUp')) return
    const total = diff.hunks.length
    if (total === 0) return
    e.preventDefault()
    setCurrentHunkIndex((i) => {
      const next = e.key === 'ArrowDown' ? Math.min(total - 1, i + 1) : Math.max(0, i - 1)
      const headers = paneRef.current?.querySelectorAll('.diff-hunk-header')
      headers?.[next]?.scrollIntoView({ behavior: 'auto', block: 'nearest' })
      return next
    })
  })

  const toggleGroup = (index: number) => {
    setExpandedGroups((prev) => {
      const next = new Set(prev)
      if (next.has(index)) next.delete(index)
      else next.add(index)
      return next
    })
  }

  const image = images && imageMimeType(path) ? <ImageDiff {...images} path={path} refreshKey={diff} /> : null
  if (image && (diff.binary || diff.tooLarge)) return image
  // SVG is text to Git, so it can be read either way.
  const imageToggle = image && (
    <div className="diff-image-toggle">
      <SegmentedControl
        value={showImage ? 'image' : 'text'}
        onChange={(value) => setShowImage(value === 'image')}
        options={[
          { value: 'image', label: 'Image', icon: Image },
          { value: 'text', label: 'Text', icon: FileCode },
        ]}
      />
    </div>
  )
  if (image && showImage) {
    return (
      <div>
        {imageToggle}
        {image}
      </div>
    )
  }

  if (diff.tooLarge) {
    return (
      <p className="diff-empty-state">
        This file is large ({formatMegabytes(diff.sizeBytes)}) — the diff wasn't loaded automatically.{' '}
        {onForceLoad && (
          <button type="button" className="diff-empty-state-action" onClick={onForceLoad}>
            Load anyway
          </button>
        )}
      </p>
    )
  }

  if (diff.binary) {
    return <p className="diff-empty-state">Binary file changed.</p>
  }

  if (diff.hunks.length === 0) {
    return <p className="diff-empty-state">No differences.</p>
  }

  const stats = diffStats(diff)
  // A conflicted file is one working file with both sides inline: there's no
  // second side to split against and no hunk to stage on its own.
  const mode = diff.conflicted ? 'unified' : viewMode

  return (
    <div>
      {imageToggle}
      {diff.conflicted && (
        <p className="diff-conflict-note">Conflicted. Edit the marked sections to resolve, then stage the file.</p>
      )}
      {mode === 'split' && baseSide && targetSide && (
        <div className="diff-side-headers">
          <div className="diff-side-header">
            <span className="diff-side-header-dot diff-side-header-dot-base" />
            <span className="diff-side-header-label">
              {baseSide.label}
              {baseSide.sha && <span className="diff-side-header-sha">{baseSide.sha}</span>}
            </span>
            <span className="diff-side-header-count diff-side-header-count-removed">{stats.removed} removed</span>
          </div>
          <div className="diff-side-header">
            <span className="diff-side-header-dot diff-side-header-dot-target" />
            <span className="diff-side-header-label">{targetSide.label}</span>
            <span className="diff-side-header-count diff-side-header-count-added">{stats.added} added</span>
          </div>
        </div>
      )}
      <div className={mode === 'split' ? 'diff-viewer' : 'diff-viewer diff-viewer-unified'}>
        <DiffPane
          mode={mode}
          rows={buildDisplayRows(diff.hunks)}
          path={path}
          expandedGroups={expandedGroups}
          onToggleGroup={toggleGroup}
          currentHunkIndex={currentHunkIndex}
          scrollRef={paneRef}
          {...(diff.conflicted ? {} : hunkActions)}
        />
      </div>
    </div>
  )
}

export default DiffViewer
