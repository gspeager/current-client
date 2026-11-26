import { useEffect, useLayoutEffect, useMemo, useRef, useState, type ReactNode } from 'react'
import { ChevronDown, ChevronUp, ClipboardCheck, FileCode, Image, List } from 'lucide-react'
import {
  DiffService,
  type FileDiff,
  type LineSelection,
  type ReadThroughNote,
  type ReviewThread,
} from '@current-client-bindings/app'
import ContextMenu, { menuAnchor, type ContextMenuState } from '../../components/controls/ContextMenu'
import SegmentedControl from '../../components/controls/SegmentedControl'
import { useAsyncData } from '../../lib/useAsyncData'
import DependencySummary from './DependencySummary'
import ImageDiff, { type ImageSources } from './ImageDiff'
import { imageMimeType } from './imageFiles'
import { useDiffWrap } from './diffWrap'
import LfsDiff from './LfsDiff'
import { lfsPointerChange } from './lfsPointer'
import SubmoduleDiff from './SubmoduleDiff'
import { submoduleChange } from './submoduleChange'
import { buildDisplayRows, diffStats, isLongDiff, lineRef } from './diffRows'
import { conflictRows, type ConflictView } from './conflictRows'
import { formatSize } from './fileSize'
import DiffPane, { lineKey, type DiffPaneHandle, type HunkActions } from './DiffPane'
import ReviewThreadBlock from './ReviewThreadBlock'
import DiffMinimap from './DiffMinimap'
import PendingCommentBlock from '../review/PendingCommentBlock'
import ReviewComposer from '../review/ReviewComposer'
import ReviewDialog from '../review/ReviewDialog'
import ReviewNoteBlock, { noteWhere, type LineNotes } from '../review/ReviewNoteBlock'
import { useReviewActions } from '../review/useReviewActions'
import { issueDraftFromLine, useNewIssue } from '../../lib/issuePanel'
import { reviewTarget, useReviewDraft } from '../../lib/reviewDraft'
import { plural } from '../../lib/plural'
import { toggled } from '../../lib/sets'
import { useWindowKeydown } from '../../lib/useWindowKeydown'
import { useRovingRows } from '../../lib/useRovingRows'
import { NARROW_SPLIT_WIDTH, useWindowNarrowerThan } from '../../lib/useWindowNarrowerThan'
import './DiffViewer.scss'
import { matchesShortcut } from '../../lib/shortcuts'

// A line of the new version, or with no line, of the old one.
export interface DiffFocus {
  line: number
  oldLine?: number
}

interface DiffSideLabel {
  label: string
  sha?: string
  // A branch's lane color, where the side is a branch.
  color?: string
}

const paneRowOf = (el: HTMLElement) => Number(el.closest<HTMLElement>('[data-index]')?.dataset.index ?? 0)

// The base side counts its removed lines, the target its added ones.
function SideHeader({ side, which, count }: { side: DiffSideLabel; which: 'base' | 'target'; count?: number }) {
  const change = which === 'base' ? 'removed' : 'added'
  return (
    <div className="diff-side-header">
      <span
        className={`diff-side-header-dot diff-side-header-dot-${which}`}
        style={side.color ? { background: side.color } : undefined}
      />
      <span className="diff-side-header-label">
        {side.label}
        {side.sha && <span className="diff-side-header-sha">{side.sha}</span>}
      </span>
      {count !== undefined && (
        <span className={`diff-side-header-count diff-side-header-count-${change}`}>
          {count} {change}
        </span>
      )}
    </div>
  )
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
  // Notes on this file's lines, kept on this computer.
  notes?: LineNotes
  // Where the first diff shown opens, and where the diff has since been scrolled.
  initialScrollTop?: number
  onScrollTopChange?: (top: number) => void
  // A line to bring into view; a new object asks again.
  focusLine?: DiffFocus | null
}

function DiffViewer({
  diff,
  path,
  viewMode,
  initialScrollTop = 0,
  onScrollTopChange,
  focusLine = null,
  baseSide,
  targetSide,
  onForceLoad,
  images,
  repoPath,
  threads,
  outdatedThreads,
  conflictView = 'result',
  review,
  notes,
  ...hunkActions
}: DiffViewerProps) {
  const [showImage, setShowImage] = useState(true)
  const target = review ? reviewTarget(review.repoPath, review.number) : undefined
  const draft = useReviewDraft(target)
  const actions = useReviewActions(review?.repoPath, review?.number)
  const [composer, setComposer] = useState<{ line: number; oldLine: number } | null>(null)
  const takesNotes = !!notes
  const [reviewOpen, setReviewOpen] = useState(false)
  const [reviewed, setReviewed] = useState(false)
  const newIssue = useNewIssue()
  const [menu, setMenu] = useState<ContextMenuState | null>(null)
  const { wrap } = useDiffWrap()
  const tooNarrowForSplit = useWindowNarrowerThan(NARROW_SPLIT_WIDTH)
  const paneRef = useRef<HTMLDivElement>(null)
  const handle = useRef<DiffPaneHandle>(null)
  // A conflicted file shows its two sides, or the working file with the markers
  // inline; either way there's no hunk to stage on its own.
  const blocks = useMemo(
    () => (diff.conflicted ? conflictRows(diff.hunks, conflictView) : { rows: [], blockStarts: [] }),
    [diff, conflictView],
  )
  const displayRows = useMemo(() => (diff.conflicted ? blocks.rows : buildDisplayRows(diff.hunks)), [diff, blocks])
  const blockStarts = useMemo(() => (diff.conflicted ? new Set(blocks.blockStarts) : undefined), [diff, blocks])
  const stats = useMemo(() => diffStats(diff), [diff])
  const shownLines = useMemo(() => {
    const refs = diff.hunks.flatMap((h) => h.lines.map((l) => lineRef(l.kind === 'removed' ? 0 : l.newLine, l.oldLine)))
    return new Set(refs)
  }, [diff])
  const long = useMemo(() => isLongDiff(displayRows), [displayRows])
  const setLineStops = useRovingRows('.diff-line-select', {
    compareRows: (a, b) => paneRowOf(a) - paneRowOf(b),
    scrollToEdge: long ? (edge) => handle.current?.revealEdge(edge) : undefined,
    scrollPast: long
      ? (row, forward, next) =>
          (!next || paneRowOf(next) !== paneRowOf(row)) && !!handle.current?.revealChange(paneRowOf(row), forward)
      : undefined,
  })
  useEffect(() => setLineStops(paneRef.current))
  const [expandedGroups, setExpandedGroups] = useState<Set<number>>(new Set())
  const [currentHunkIndex, setCurrentHunkIndex] = useState(0)
  const [currentBlock, setCurrentBlock] = useState(0)
  const [selectedLines, setSelectedLines] = useState<Set<string>>(new Set())
  const lastToggled = useRef<string | null>(null)
  const initialScrollApplied = useRef(false)
  const [showRawDiff, setShowRawDiff] = useState(false)
  const fileVersions = JSON.stringify([images?.before, images?.after])
  const { data: dependencies } = useAsyncData(
    () =>
      images
        ? DiffService.GetDependencySummary(images.repoPath, diff.oldPath || path, path, images.before, images.after)
        : null,
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [images?.repoPath, path, fileVersions],
    { refreshKey: diff },
  )

  useEffect(() => setShowRawDiff(false), [path])

  // Once per request: a reload of the diff shouldn't pull the view back.
  const focused = useRef<DiffFocus | null>(null)
  useEffect(() => {
    if (!focusLine || focused.current === focusLine) return
    if (handle.current?.revealLine(focusLine.line, focusLine.oldLine)) focused.current = focusLine
  }, [focusLine, diff])

  // The pane only exists once the diff has lines to show.
  useLayoutEffect(() => {
    const pane = paneRef.current
    if (!pane || initialScrollApplied.current) return
    initialScrollApplied.current = true
    if (!initialScrollTop) return
    const sides = pane.querySelectorAll<HTMLElement>('.diff-side')
    for (const scroller of sides.length > 0 ? sides : [pane]) scroller.scrollTop = initialScrollTop
  })

  useEffect(() => {
    setExpandedGroups(new Set())
    setCurrentHunkIndex(0)
    setCurrentBlock(0)
    setSelectedLines(new Set())
    lastToggled.current = null
    // What's read through keeps moving, and a note being typed stays.
    if (!takesNotes) setComposer(null)
  }, [diff, takesNotes])

  const stepBlock = (step: number) => {
    const index = Math.min(blocks.blockStarts.length - 1, Math.max(0, currentBlock + step))
    setCurrentBlock(index)
    handle.current?.revealBlock(index)
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

  const selectionByHunk = useMemo(() => {
    const byHunk = new Map<string, LineSelection>()
    if (selectedLines.size === 0) return byHunk
    for (const h of diff.hunks) {
      const chosen = h.lines.filter((l) => selectedLines.has(lineKey(l) ?? ''))
      byHunk.set(h.raw, {
        added: chosen.filter((l) => l.kind === 'added').map((l) => l.newLine),
        removed: chosen.filter((l) => l.kind === 'removed').map((l) => l.oldLine),
      })
    }
    return byHunk
  }, [diff, selectedLines])
  const canSelectLines = !diff.conflicted && Boolean(hunkActions.onStageHunk ?? hunkActions.onUnstageHunk)

  useWindowKeydown((e) => {
    // S, U and D act on the current hunk while focus is in the diff.
    if (!e.metaKey && !e.ctrlKey && !e.altKey && paneRef.current?.contains(document.activeElement)) {
      const hunk = diff.hunks[currentHunkIndex]
      const key = e.key.toLowerCase()
      if (hunk && key === 's') hunkActions.onStageHunk?.(hunk.raw)
      if (hunk && key === 'u') hunkActions.onUnstageHunk?.(hunk.raw)
      if (hunk && key === 'd') hunkActions.onDiscardHunk?.(hunk.raw)
    }
    const forward = matchesShortcut(e, 'F7') || matchesShortcut(e, 'alt+ArrowDown')
    const back = matchesShortcut(e, 'shift+F7') || matchesShortcut(e, 'alt+ArrowUp')
    if (!forward && !back) return
    if (diff.conflicted) {
      if (blocks.blockStarts.length === 0) return
      e.preventDefault()
      stepBlock(forward ? 1 : -1)
      return
    }
    const total = diff.hunks.length
    if (total === 0) return
    e.preventDefault()
    setCurrentHunkIndex((i) => {
      const next = forward ? Math.min(total - 1, i + 1) : Math.max(0, i - 1)
      handle.current?.revealHunk(next)
      return next
    })
  })

  const toggleGroup = (index: number) => setExpandedGroups((prev) => toggled(prev, index))

  const submodule = submoduleChange(diff)
  if (submodule) return <SubmoduleDiff change={submodule} path={path} repoPath={repoPath} />
  const lfs = lfsPointerChange(diff)
  if (lfs) return <LfsDiff change={lfs} />

  const summarized = dependencies?.available && dependencies.changes.length > 0 && !diff.conflicted && !focusLine
  const summaryToggle = summarized && (
    <div className="diff-image-toggle">
      <SegmentedControl
        value={showRawDiff ? 'raw' : 'summary'}
        onChange={(value) => setShowRawDiff(value === 'raw')}
        options={[
          { value: 'summary', label: 'Summary', icon: List },
          { value: 'raw', label: 'Raw diff', icon: FileCode },
        ]}
      />
    </div>
  )
  if (summarized && !showRawDiff) {
    return (
      <div>
        {summaryToggle}
        <DependencySummary changes={dependencies.changes} />
      </div>
    )
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
      <div>
        {summaryToggle}
        <p className="diff-empty-state">
          This file is large ({formatSize(diff.sizeBytes)}) — the diff wasn't loaded automatically.{' '}
          {onForceLoad && (
            <button type="button" className="diff-empty-state-action" onClick={onForceLoad}>
              Load anyway
            </button>
          )}
        </p>
      </div>
    )
  }

  if (diff.binary) {
    return <p className="diff-empty-state">Binary file changed.</p>
  }

  if (diff.hunks.length === 0) {
    return <p className="diff-empty-state">No differences.</p>
  }

  const chosenMode = diff.conflicted ? (conflictView === 'side-by-side' ? 'split' : 'unified') : viewMode
  const mode = tooNarrowForSplit ? 'unified' : chosenMode
  // Threads pin to a new-side line the diff shows; the rest, and all of them
  // when each side scrolls on its own, are listed under the diff.
  const sidesApart = mode === 'split' && !wrap
  const blocksByLine = new Map<string, ReactNode[]>()
  const place = (ref: string, node: ReactNode) => blocksByLine.set(ref, [...(blocksByLine.get(ref) ?? []), node])
  const unplaced: ReviewThread[] = []
  for (const thread of threads ?? []) {
    if (!sidesApart && thread.line && shownLines.has(lineRef(thread.line, 0))) {
      place(
        lineRef(thread.line, 0),
        <ReviewThreadBlock key={thread.id} thread={thread} actions={actions?.threadActions} />,
      )
    } else {
      unplaced.push(thread)
    }
  }
  const drafted = draft.comments.filter((c) => c.path === path)
  const unplacedNotes: ReadThroughNote[] = []
  for (const note of notes?.items ?? []) {
    const ref = lineRef(note.line, note.oldLine)
    if (note.outdated || sidesApart || !shownLines.has(ref)) {
      unplacedNotes.push(note)
    } else {
      place(ref, <ReviewNoteBlock key={note.id} note={note} onEdit={notes!.edit} />)
    }
  }
  if (!sidesApart && notes && composer && (composer.line > 0 || !notes.newOnly)) {
    place(
      lineRef(composer.line, composer.oldLine),
      <ReviewComposer
        key="composer"
        placeholder={`Note on ${noteWhere(composer)}`}
        sendLabel="Add note"
        onSend={(text) => notes.add(composer.line, composer.oldLine, text).then(() => setComposer(null))}
        onCancel={() => setComposer(null)}
      />,
    )
  }
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
  const canComment = (!!notes || (!!review && !!actions?.capabilities.lineComments)) && !diff.conflicted && !sidesApart

  return (
    <div className="diff-view">
      {summaryToggle}
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
                : ''}
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
          <SideHeader side={baseSide} which="base" count={diff.conflicted ? undefined : stats.removed} />
          <SideHeader side={targetSide} which="target" count={diff.conflicted ? undefined : stats.added} />
        </div>
      )}
      <div
        className={`diff-viewer${mode === 'split' ? '' : ' diff-viewer-unified'}${wrap ? '' : ' diff-viewer-nowrap'}`}
      >
        <DiffPane
          mode={mode}
          rows={displayRows}
          blockStarts={blockStarts}
          resultView={diff.conflicted && conflictView === 'result'}
          path={path}
          expandedGroups={expandedGroups}
          onToggleGroup={toggleGroup}
          currentHunkIndex={currentHunkIndex}
          scrollRef={paneRef}
          handle={handle}
          onScrollTopChange={onScrollTopChange}
          lineSelect={
            canSelectLines ? { selected: selectedLines, onToggle: toggleLine, byHunk: selectionByHunk } : undefined
          }
          blocksByLine={blocksByLine}
          lineComment={
            canComment
              ? {
                  label: notes ? 'Add note on' : 'Comment on',
                  removed: !notes?.newOnly,
                  run: (line, oldLine) => setComposer({ line, oldLine }),
                }
              : undefined
          }
          onLineMenu={
            newIssue
              ? (e, line, content) =>
                  setMenu({
                    ...menuAnchor(e),
                    items: [
                      {
                        label: 'Create issue from line…',
                        onClick: () => newIssue(issueDraftFromLine(path, line, content)),
                      },
                    ],
                  })
              : undefined
          }
          {...(diff.conflicted ? {} : hunkActions)}
        />
        <DiffMinimap rows={displayRows} paneRef={paneRef} always={long && !sidesApart} />
      </div>
      {unplacedNotes.length > 0 && (
        <section className="review-threads-outdated" aria-label="Notes not shown at a line">
          <span className="review-threads-outdated-label">Notes on lines not in this diff</span>
          {unplacedNotes.map((note) => (
            <ReviewNoteBlock key={note.id} note={note} onEdit={notes!.edit} />
          ))}
        </section>
      )}
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
      {menu && <ContextMenu state={menu} onClose={() => setMenu(null)} />}
      {diff.conflicted && blocks.blockStarts.length > 0 && (
        <div className="diff-conflict-footer">
          <span>
            Block {Math.min(currentBlock, blocks.blockStarts.length - 1) + 1} of {blocks.blockStarts.length}
          </span>
          <button
            type="button"
            onClick={() => stepBlock(-1)}
            disabled={currentBlock === 0}
            aria-label="Previous block"
            title="Previous block"
          >
            <ChevronUp size={14} strokeWidth={1.75} />
          </button>
          <button
            type="button"
            onClick={() => stepBlock(1)}
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
