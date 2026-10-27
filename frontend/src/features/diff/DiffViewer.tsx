import { useEffect, useRef, useState, type ReactNode } from 'react'
import { ChevronDown, ChevronUp, ClipboardCheck, FileCode, Image } from 'lucide-react'
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
import DiffPane, { lineKey, lineRef, type HunkActions } from './DiffPane'
import ReviewThreadBlock from './ReviewThreadBlock'
import PendingCommentBlock from '../review/PendingCommentBlock'
import ReviewComposer from '../review/ReviewComposer'
import ReviewDialog from '../review/ReviewDialog'
import { useReviewActions } from '../review/useReviewActions'
import { reviewTarget, useReviewDraft } from '../../lib/reviewDraft'
import { plural } from '../../lib/plural'
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
  // The change this diff belongs to, when the host is on: lines take review
  // comments, threads take replies, and a review can be submitted.
  review?: { repoPath: string; number: number; prefix: string }
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
  review,
  ...hunkActions
}: DiffViewerProps) {
  const [showImage, setShowImage] = useState(true)
  const target = review ? reviewTarget(review.repoPath, review.number) : undefined
  const draft = useReviewDraft(target)
  const actions = useReviewActions(review?.repoPath, review?.number)
  const [composer, setComposer] = useState<{ line: number; oldLine: number } | null>(null)
  const [reviewOpen, setReviewOpen] = useState(false)
  const [reviewed, setReviewed] = useState(false)
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
    setComposer(null)
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
  const blocksByLine = new Map<string, ReactNode[]>()
  const place = (ref: string, node: ReactNode) => blocksByLine.set(ref, [...(blocksByLine.get(ref) ?? []), node])
  const unplaced: ReviewThread[] = []
  for (const thread of threads ?? []) {
    if (!sidesApart && thread.line && shownLines.has(thread.line)) {
      place(
        lineRef(thread.line, 0),
        <ReviewThreadBlock key={thread.id} thread={thread} actions={actions?.threadActions} />,
      )
    } else {
      unplaced.push(thread)
    }
  }
  const drafted = draft.comments.filter((c) => c.path === path)
  if (!sidesApart && target) {
    for (const c of drafted) {
      place(
        lineRef(c.line, c.oldLine),
        <PendingCommentBlock key={c.id} comment={c} onRemove={() => draft.remove(target, c.id)} />,
      )
    }
    if (composer && actions) {
      place(
        lineRef(composer.line, composer.oldLine),
        <ReviewComposer
          key="composer"
          placeholder={composer.line ? `Comment on line ${composer.line}` : `Comment on old line ${composer.oldLine}`}
          onSend={(body) => actions.comment(path, composer.line, composer.oldLine, body).then(() => setComposer(null))}
          onAddToReview={(body) => {
            draft.add(target, { path, line: composer.line, oldLine: composer.oldLine, body })
            setComposer(null)
          }}
          onCancel={() => setComposer(null)}
        />,
      )
    }
  }
  const canComment = !!review && !!actions?.capabilities.lineComments && !diff.conflicted && !sidesApart

  return (
    <div>
      {imageToggle}
      {review && actions && (
        <div className="diff-review-bar" role="toolbar" aria-label="Review">
          <ClipboardCheck size={12} strokeWidth={1.75} />
          <span className="diff-review-bar-change">
            {review.prefix}
            {review.number}
          </span>
          <span className="diff-review-bar-note">
            {draft.comments.length > 0
              ? `${plural(draft.comments.length, 'drafted comment')}, not sent yet`
              : reviewed
                ? 'Review submitted.'
                : canComment
                  ? 'Comment on a line from its gutter, now or as part of a review.'
                  : 'Threads take replies here.'}
          </span>
          {draft.comments.length > 0 && (
            <button type="button" className="diff-review-bar-button" onClick={() => target && draft.clear(target)}>
              Discard drafts
            </button>
          )}
          <button type="button" className="diff-review-bar-button" onClick={() => setReviewOpen(true)}>
            Submit review…
          </button>
        </div>
      )}
      {reviewOpen && review && (
        <ReviewDialog
          repoPath={review.repoPath}
          number={review.number}
          prefix={review.prefix}
          onSubmitted={() => {
            setReviewOpen(false)
            setReviewed(true)
          }}
          onClose={() => setReviewOpen(false)}
        />
      )}
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
            <span className="diff-side-header-label">
              {targetSide.label}
              {targetSide.sha && <span className="diff-side-header-sha">{targetSide.sha}</span>}
            </span>
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
          blocksByLine={blocksByLine}
          onCommentLine={canComment ? (line, oldLine) => setComposer({ line, oldLine }) : undefined}
          {...(diff.conflicted ? {} : hunkActions)}
        />
        {(unplaced.length > 0 || (outdatedThreads?.length ?? 0) > 0) && (
          <section className="review-threads-outdated" aria-label="Threads not shown at a line">
            {unplaced.length > 0 && (
              <span className="review-threads-outdated-label">Threads on lines not in this diff</span>
            )}
            {unplaced.map((thread) => (
              <ReviewThreadBlock
                key={thread.id}
                thread={thread}
                where={`line ${thread.line}`}
                actions={actions?.threadActions}
              />
            ))}
            {(outdatedThreads?.length ?? 0) > 0 && (
              <span className="review-threads-outdated-label">Outdated: the code has changed since</span>
            )}
            {outdatedThreads?.map((thread) => (
              <ReviewThreadBlock
                key={thread.id}
                thread={thread}
                where={thread.oldLine ? `old line ${thread.oldLine}` : `line ${thread.line}`}
                actions={actions?.threadActions}
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
