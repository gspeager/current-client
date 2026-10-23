import { useEffect, useRef, useState } from 'react'
import { ChevronDown, ChevronUp, FileCode, Image } from 'lucide-react'
import type { FileDiff, LineSelection, ReviewThread } from '@current-client-bindings/app'
import SegmentedControl from '../../components/controls/SegmentedControl'
import ImageDiff, { type ImageSources } from './ImageDiff'
import { imageMimeType } from './imageFiles'
import { useDiffWrap } from './diffWrap'
import LfsDiff from './LfsDiff'
import { lfsPointerChange } from './lfsPointer'
import SubmoduleDiff from './SubmoduleDiff'
import { submoduleChange } from './submoduleChange'
import { buildDisplayRows, diffStats } from './diffRows'
import { conflictRows, type ConflictView } from './conflictRows'
import DiffPane, { lineKey, type HunkActions } from './DiffPane'
import ReviewThreadBlock from './ReviewThreadBlock'
import { toggled } from '../../lib/sets'
import { useWindowKeydown } from '../../lib/useWindowKeydown'
import './DiffViewer.scss'

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
  // Review threads for this file: pinned to their lines, outdated ones listed below.
  threads?: ReviewThread[]
  outdatedThreads?: ReviewThread[]
  // Lets a submodule's change list the commits between its two versions.
  repoPath?: string
  // For a conflicted file: each side in its own column, or the working file as it is.
  conflictView?: ConflictView
}

function DiffViewer({
  diff,
  path,
  viewMode,
  baseSide,
  targetSide,
  onForceLoad,
  images,
  repoPath,
  threads,
  outdatedThreads,
  conflictView = 'result',
  ...hunkActions
}: DiffViewerProps) {
  const [showImage, setShowImage] = useState(true)
  const { wrap } = useDiffWrap()
  const paneRef = useRef<HTMLDivElement>(null)
  const [expandedGroups, setExpandedGroups] = useState<Set<number>>(new Set())
  const [currentHunkIndex, setCurrentHunkIndex] = useState(0)
  const [currentBlock, setCurrentBlock] = useState(0)
  const [selectedLines, setSelectedLines] = useState<Set<string>>(new Set())
  const lastToggled = useRef<string | null>(null)

  useEffect(() => {
    setExpandedGroups(new Set())
    setCurrentHunkIndex(0)
    setCurrentBlock(0)
    setSelectedLines(new Set())
    lastToggled.current = null
  }, [diff])

  const goToBlock = (index: number) => {
    setCurrentBlock(index)
    paneRef.current?.querySelectorAll('.diff-conflict-block-start')[index]?.scrollIntoView({ block: 'center' })
  }

  // Shift-click selects every changed line between the last one toggled and this one.
  const toggleLine = (key: string, range: boolean) => {
    const order = diff.hunks.flatMap((h) => h.lines.map(lineKey).filter((k): k is string => k !== null))
    const from = lastToggled.current ? order.indexOf(lastToggled.current) : -1
    const to = order.indexOf(key)
    setSelectedLines((prev) => {
      if (!range || from === -1) return toggled(prev, key)
      return new Set([...prev, ...order.slice(Math.min(from, to), Math.max(from, to) + 1)])
    })
    lastToggled.current = key
  }

  const selectionByHunk = new Map<string, LineSelection>(
    diff.hunks.map((h) => [
      h.raw,
      {
        added: h.lines.filter((l) => selectedLines.has(`+${l.newLine}`) && l.kind === 'added').map((l) => l.newLine),
        removed: h.lines
          .filter((l) => selectedLines.has(`-${l.oldLine}`) && l.kind === 'removed')
          .map((l) => l.oldLine),
      },
    ]),
  )
  const canSelectLines = !diff.conflicted && Boolean(hunkActions.onStageHunk ?? hunkActions.onUnstageHunk)

  useWindowKeydown((e) => {
    if (!e.altKey || (e.key !== 'ArrowDown' && e.key !== 'ArrowUp')) return
    if (diff.conflicted) {
      if (blocks.blockStarts.length === 0) return
      e.preventDefault()
      goToBlock(
        e.key === 'ArrowDown'
          ? Math.min(blocks.blockStarts.length - 1, currentBlock + 1)
          : Math.max(0, currentBlock - 1),
      )
      return
    }
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

  const toggleGroup = (index: number) => setExpandedGroups((prev) => toggled(prev, index))
  const blocks = diff.conflicted ? conflictRows(diff.hunks, conflictView) : { rows: [], blockStarts: [] }

  const submodule = submoduleChange(diff)
  if (submodule) return <SubmoduleDiff change={submodule} path={path} repoPath={repoPath} />
  const lfs = lfsPointerChange(diff)
  if (lfs) return <LfsDiff change={lfs} />

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
  // A conflicted file shows its two sides, or the working file with the markers
  // inline; either way there's no hunk to stage on its own.
  const mode = diff.conflicted ? (conflictView === 'side-by-side' ? 'split' : 'unified') : viewMode
  // Threads pin to a new-side line the diff shows; the rest, and all of them
  // when each side scrolls on its own, are listed under the diff.
  const sidesApart = mode === 'split' && !wrap
  const shownLines = new Set(diff.hunks.flatMap((h) => h.lines.map((l) => l.newLine).filter(Boolean)))
  const threadsByLine = new Map<number, ReviewThread[]>()
  const unplaced: ReviewThread[] = []
  for (const thread of threads ?? []) {
    if (!sidesApart && thread.line && shownLines.has(thread.line)) {
      threadsByLine.set(thread.line, [...(threadsByLine.get(thread.line) ?? []), thread])
    } else {
      unplaced.push(thread)
    }
  }

  return (
    <div>
      {imageToggle}
      {diff.conflicted && conflictView === 'result' && (
        <p className="diff-conflict-note">Conflicted. Edit the marked sections to resolve, then mark it resolved.</p>
      )}
      {mode === 'split' && baseSide && targetSide && (
        <div className="diff-side-headers">
          <div className="diff-side-header">
            <span className="diff-side-header-dot diff-side-header-dot-base" />
            <span className="diff-side-header-label">
              {baseSide.label}
              {baseSide.sha && <span className="diff-side-header-sha">{baseSide.sha}</span>}
            </span>
            {!diff.conflicted && (
              <span className="diff-side-header-count diff-side-header-count-removed">{stats.removed} removed</span>
            )}
          </div>
          <div className="diff-side-header">
            <span className="diff-side-header-dot diff-side-header-dot-target" />
            <span className="diff-side-header-label">{targetSide.label}</span>
            {!diff.conflicted && (
              <span className="diff-side-header-count diff-side-header-count-added">{stats.added} added</span>
            )}
          </div>
        </div>
      )}
      <div
        className={`diff-viewer${mode === 'split' ? '' : ' diff-viewer-unified'}${wrap ? '' : ' diff-viewer-nowrap'}`}
      >
        <DiffPane
          mode={mode}
          rows={diff.conflicted ? blocks.rows : buildDisplayRows(diff.hunks)}
          blockStarts={diff.conflicted ? new Set(blocks.blockStarts) : undefined}
          resultView={diff.conflicted && conflictView === 'result'}
          path={path}
          expandedGroups={expandedGroups}
          onToggleGroup={toggleGroup}
          currentHunkIndex={currentHunkIndex}
          scrollRef={paneRef}
          lineSelect={
            canSelectLines ? { selected: selectedLines, onToggle: toggleLine, byHunk: selectionByHunk } : undefined
          }
          threadsByLine={threadsByLine}
          {...(diff.conflicted ? {} : hunkActions)}
        />
        {(unplaced.length > 0 || (outdatedThreads?.length ?? 0) > 0) && (
          <section className="review-threads-outdated" aria-label="Threads not shown at a line">
            {unplaced.length > 0 && (
              <span className="review-threads-outdated-label">Threads on lines not in this diff</span>
            )}
            {unplaced.map((thread) => (
              <ReviewThreadBlock key={thread.id} thread={thread} where={`line ${thread.line}`} />
            ))}
            {(outdatedThreads?.length ?? 0) > 0 && (
              <span className="review-threads-outdated-label">Outdated: the code has changed since</span>
            )}
            {outdatedThreads?.map((thread) => (
              <ReviewThreadBlock
                key={thread.id}
                thread={thread}
                where={thread.oldLine ? `old line ${thread.oldLine}` : `line ${thread.line}`}
              />
            ))}
          </section>
        )}
      </div>
      {diff.conflicted && blocks.blockStarts.length > 0 && (
        <div className="diff-conflict-footer">
          <span>
            Block {Math.min(currentBlock, blocks.blockStarts.length - 1) + 1} of {blocks.blockStarts.length}
          </span>
          <button
            type="button"
            onClick={() => goToBlock(Math.max(0, currentBlock - 1))}
            disabled={currentBlock === 0}
            aria-label="Previous block"
            title="Previous block"
          >
            <ChevronUp size={14} strokeWidth={1.75} />
          </button>
          <button
            type="button"
            onClick={() => goToBlock(Math.min(blocks.blockStarts.length - 1, currentBlock + 1))}
            disabled={currentBlock >= blocks.blockStarts.length - 1}
            aria-label="Next block"
            title="Next block"
          >
            <ChevronDown size={14} strokeWidth={1.75} />
          </button>
        </div>
      )}
    </div>
  )
}

export default DiffViewer
