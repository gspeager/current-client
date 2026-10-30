import { Fragment, useLayoutEffect, useRef, type Key, type MouseEvent, type ReactNode, type RefObject } from 'react'
import { MessageSquarePlus } from 'lucide-react'
import type { DiffLine, LineSelection } from '@current-client-bindings/app'
import { useDiffWrap } from './diffWrap'
import { unifiedRowsForPair, type DisplayRow, type PairedLine, type UnifiedRow } from './diffRows'
import { languageForPath, tokenizeLine } from './syntaxHighlight'
import { diffLineWords } from './wordDiff'

type Side = 'left' | 'right'

// With lines, an action applies to just those lines of the hunk.
export interface HunkActions {
  onStageHunk?: (hunkRaw: string, lines?: LineSelection) => void
  onUnstageHunk?: (hunkRaw: string, lines?: LineSelection) => void
  onDiscardHunk?: (hunkRaw: string, lines?: LineSelection) => void
  // Discarding selected lines is only possible in the working tree.
  canDiscardLines?: boolean
}

// Identifies a changed line for selection: added lines by new line number,
// removed lines by old. Other lines can't be selected.
export function lineKey(line: DiffLine | null): string | null {
  if (line?.kind === 'added') return `+${line.newLine}`
  if (line?.kind === 'removed') return `-${line.oldLine}`
  return null
}

// Names a line for blocks placed under it: a new-side line as +n, a removed
// line as -n.
export function lineRef(line: number, oldLine: number): string {
  return line ? `+${line}` : `-${oldLine}`
}

type CommentOnLine = (line: number, oldLine: number) => void

// The comment button sits over the gutter, shown while the row is hovered or
// it has focus, so the row grid stays as it is.
function CommentButton({ line, oldLine, onComment }: { line: number; oldLine: number; onComment: CommentOnLine }) {
  const where = line ? `line ${line}` : `old line ${oldLine}`
  return (
    <button
      type="button"
      className="diff-line-comment"
      onClick={() => onComment(line, oldLine)}
      aria-label={`Comment on ${where}`}
      title={`Comment on ${where}`}
    >
      <MessageSquarePlus size={12} strokeWidth={1.75} />
    </button>
  )
}

interface LineSelect {
  selected: Set<string>
  onToggle: (key: string, range: boolean) => void
}

function LineNumber({ line, side, select }: { line: DiffLine | null; side: Side; select?: LineSelect }) {
  const key = lineKey(line)
  const n = lineNumber(line, side)
  if (!select || !key) return <span className="diff-line-number">{n}</span>
  return (
    <button
      type="button"
      className="diff-line-number diff-line-select"
      aria-label={`Select ${line?.kind} line ${n}`}
      aria-pressed={select.selected.has(key)}
      onClick={(e) => select.onToggle(key, e.shiftKey)}
    >
      {n}
    </button>
  )
}

function selectedClass(line: DiffLine | null, select?: LineSelect): string {
  const key = lineKey(line)
  return key && select?.selected.has(key) ? ' diff-row-selected' : ''
}

function rowClass(line: DiffLine | null): string {
  if (!line) return 'diff-row diff-row-empty'
  if (line.moved) return 'diff-row diff-row-moved'
  if (line.kind === 'added') return 'diff-row diff-row-added'
  if (line.kind === 'removed') return 'diff-row diff-row-removed'
  if (line.kind.startsWith('conflict-')) return `diff-row diff-row-${line.kind}`
  return 'diff-row'
}

function marker(line: DiffLine | null): string {
  if (line?.kind === 'added') return '+'
  if (line?.kind === 'removed') return '-'
  return ''
}

function lineNumber(line: DiffLine | null, side: Side): string {
  if (!line) return ''
  const n = side === 'left' ? line.oldLine : line.newLine
  return n ? String(n) : ''
}

function renderTokens(text: string, language: string | null, keyPrefix: Key): ReactNode[] {
  return tokenizeLine(text, language).map((span, i) =>
    span.role ? (
      <span key={`${keyPrefix}-${i}`} className={`token-${span.role}`}>
        {span.text}
      </span>
    ) : (
      span.text
    ),
  )
}

function DiffContent({ pair, side, language }: { pair: PairedLine; side: Side; language: string | null }) {
  const line = side === 'left' ? pair.left : pair.right
  if (!line) return <span className="diff-content" />

  const isModifiedPair = pair.left?.kind === 'removed' && pair.right?.kind === 'added'
  if (!isModifiedPair) {
    return <span className="diff-content">{renderTokens(line.content, language, 'l')}</span>
  }

  const words = diffLineWords(pair.left!.content, pair.right!.content)
  const wordClass = side === 'left' ? 'diff-word-removed' : 'diff-word-added'

  return (
    <span className="diff-content">
      {(side === 'left' ? words.left : words.right).map((word, i) =>
        word.changed ? (
          <span key={i} className={wordClass}>
            {renderTokens(word.text, language, i)}
          </span>
        ) : (
          <Fragment key={i}>{renderTokens(word.text, language, i)}</Fragment>
        ),
      )}
    </span>
  )
}

function splitRow(
  pair: PairedLine,
  side: Side,
  language: string | null,
  key: Key,
  select?: LineSelect,
  extraClass = '',
  onComment?: CommentOnLine,
) {
  const line = side === 'left' ? pair.left : pair.right
  // The new side takes comments; the old side only for a line that is gone.
  const commentable = onComment && line && (side === 'right' ? line.newLine > 0 : line.kind === 'removed')
  return (
    <div
      key={key}
      className={`${rowClass(line)} diff-row-split-${side}${selectedClass(line, select)}${extraClass}`}
      data-line={line?.newLine || undefined}
    >
      <LineNumber line={line} side={side} select={select} />
      <span className="diff-marker">{marker(line)}</span>
      <DiffContent pair={pair} side={side} language={language} />
      {commentable && (
        <CommentButton
          line={side === 'right' ? line.newLine : 0}
          oldLine={side === 'right' ? 0 : line.oldLine}
          onComment={onComment}
        />
      )}
    </div>
  )
}

function unifiedRow(
  { pair, side }: UnifiedRow,
  language: string | null,
  key: Key,
  select?: LineSelect,
  extraClass = '',
  singleNumber = false,
  onComment?: CommentOnLine,
) {
  const activeLine = side === 'left' ? pair.left : pair.right
  const isContext = pair.left !== null && pair.left === pair.right
  const removed = !isContext && side === 'left' && activeLine?.kind === 'removed'
  const commentable = onComment && activeLine && (removed || (activeLine.newLine ?? 0) > 0)
  if (singleNumber) {
    return (
      <div key={key} className={`${rowClass(activeLine)} diff-row-result${extraClass}`}>
        <LineNumber line={pair.right} side="right" />
        <span className="diff-marker">{marker(activeLine)}</span>
        <DiffContent pair={pair} side={side} language={language} />
      </div>
    )
  }
  return (
    <div
      key={key}
      className={`${rowClass(activeLine)} diff-row-unified${selectedClass(activeLine, select)}${extraClass}`}
      data-line={activeLine?.newLine || undefined}
    >
      {isContext || side === 'left' ? (
        <LineNumber line={pair.left} side="left" select={select} />
      ) : (
        <span className="diff-line-number" />
      )}
      {isContext || side === 'right' ? (
        <LineNumber line={pair.right} side="right" select={select} />
      ) : (
        <span className="diff-line-number" />
      )}
      <span className="diff-marker">{marker(activeLine)}</span>
      <DiffContent pair={pair} side={side} language={language} />
      {commentable && (
        <CommentButton
          line={removed ? 0 : activeLine.newLine}
          oldLine={removed ? activeLine.oldLine : 0}
          onComment={onComment}
        />
      )}
    </div>
  )
}

function HunkActionButtons({
  raw,
  lines,
  onStageHunk,
  onUnstageHunk,
  onDiscardHunk,
  canDiscardLines,
}: HunkActions & { raw: string; lines?: LineSelection }) {
  const toggle = onStageHunk ?? onUnstageHunk
  if (!toggle) return null
  const count = lines ? lines.added.length + lines.removed.length : 0
  const what = count === 0 ? 'hunk' : count === 1 ? '1 line' : `${count} lines`
  const run = (action: (hunkRaw: string, lines?: LineSelection) => void) => (count ? action(raw, lines) : action(raw))
  return (
    <span className="diff-hunk-actions">
      <button type="button" className="diff-stage-hunk" onClick={() => run(toggle)}>
        {onStageHunk ? `Stage ${what}` : `Unstage ${what}`}
      </button>
      {onDiscardHunk && (count === 0 || canDiscardLines) && (
        <button type="button" className="diff-discard-hunk" onClick={() => run(onDiscardHunk)}>
          {count ? `Discard ${what}` : 'Discard'}
        </button>
      )}
    </span>
  )
}

// Wrap lines off in split view: each side scrolls on its own and scrolling one
// moves the other. Both take the wider side's width so they scroll the same
// distance and their rows stay aligned.
function SyncedSides({ left, right }: { left: ReactNode; right: ReactNode }) {
  const leftRef = useRef<HTMLDivElement>(null)
  const rightRef = useRef<HTMLDivElement>(null)

  useLayoutEffect(() => {
    const contents = [leftRef.current!, rightRef.current!].map((side) => side.firstElementChild as HTMLElement)
    contents.forEach((content) => (content.style.minWidth = ''))
    const width = Math.max(...contents.map((content) => content.offsetWidth))
    contents.forEach((content) => (content.style.minWidth = `max(100%, ${width}px)`))
  })

  const follow = (from: RefObject<HTMLDivElement>, to: RefObject<HTMLDivElement>) => () => {
    to.current!.scrollTop = from.current!.scrollTop
    to.current!.scrollLeft = from.current!.scrollLeft
  }

  return (
    <>
      <div className="diff-side" ref={leftRef} onScroll={follow(leftRef, rightRef)}>
        <div className="diff-side-content">{left}</div>
      </div>
      <div className="diff-side" ref={rightRef} onScroll={follow(rightRef, leftRef)}>
        <div className="diff-side-content">{right}</div>
      </div>
    </>
  )
}

interface DiffPaneProps extends HunkActions {
  mode: 'split' | 'unified'
  rows: DisplayRow[]
  path: string
  expandedGroups: Set<number>
  onToggleGroup: (index: number) => void
  currentHunkIndex: number
  scrollRef: RefObject<HTMLDivElement>
  // Present when lines can be selected for hunk actions.
  lineSelect?: LineSelect & { byHunk: Map<string, LineSelection> }
  // Blocks placed under a line, by lineRef: review threads, drafted
  // comments, the composer.
  blocksByLine?: Map<string, ReactNode[]>
  // Present when a line can take a review comment.
  onCommentLine?: CommentOnLine
  // Present when a new-side line's context menu has something to offer.
  onLineMenu?: (e: MouseEvent, line: number, content: string) => void
  // Rows that open a conflict block, marked so the footer can walk to them.
  blockStarts?: Set<number>
  // The conflicted file as it is: one line-number column, the working file's.
  resultView?: boolean
}

// Split mode is one grid, so hunk headers can span both columns, unless lines
// don't wrap and each side scrolls sideways on its own.
function DiffPane({
  mode,
  rows,
  path,
  expandedGroups,
  onToggleGroup,
  currentHunkIndex,
  scrollRef,
  lineSelect,
  blocksByLine,
  onCommentLine,
  onLineMenu,
  blockStarts,
  resultView = false,
  ...hunkActions
}: DiffPaneProps) {
  const { wrap } = useDiffWrap()
  const language = languageForPath(path)
  // Blocks go after the pair; with each side rendered on its own they can't
  // span both, so DiffViewer lists them below the diff instead, and no line
  // takes a comment.
  const blockRows = (pair: PairedLine, side?: Side): ReactNode[] => {
    if (side || !blocksByLine) return []
    const refs: string[] = []
    if (pair.left && pair.left !== pair.right && pair.left.kind === 'removed') refs.push(lineRef(0, pair.left.oldLine))
    if (pair.right?.newLine) refs.push(lineRef(pair.right.newLine, 0))
    return refs.flatMap((ref) => blocksByLine.get(ref) ?? [])
  }
  const comment = onCommentLine && !resultView ? onCommentLine : undefined
  const renderPair = (pair: PairedLine, key: Key, side?: Side, extraClass = ''): ReactNode[] => [
    ...(side
      ? [splitRow(pair, side, language, key, lineSelect, extraClass)]
      : mode === 'split'
        ? [
            splitRow(pair, 'left', language, `${key}-l`, lineSelect, extraClass, comment),
            splitRow(pair, 'right', language, `${key}-r`, lineSelect, extraClass, comment),
          ]
        : unifiedRowsForPair(pair).map((row, k) =>
            unifiedRow(row, language, `${key}-${k}`, lineSelect, k === 0 ? extraClass : '', resultView, comment),
          )),
    ...blockRows(pair, side),
  ]

  // With a side, renders just that side; the right one gets blank rows where
  // the left has a hunk header or collapsed lines.
  const renderRows = (side?: Side) => {
    let hunkIndex = -1
    return rows.flatMap<ReactNode>((row, i) => {
      if ('header' in row) {
        hunkIndex++
        if (side === 'right') return <div key={i} className="diff-side-spacer" />
        return (
          <div
            key={i}
            className={
              hunkIndex === currentHunkIndex ? 'diff-hunk-header diff-hunk-header-current' : 'diff-hunk-header'
            }
          >
            <span>{row.header}</span>
            <HunkActionButtons raw={row.raw} lines={lineSelect?.byHunk.get(row.raw)} {...hunkActions} />
          </div>
        )
      }
      if ('collapsed' in row) {
        if (expandedGroups.has(i)) {
          return row.collapsed.flatMap((pair, j) => renderPair(pair, `${i}-${j}`, side))
        }
        if (side === 'right') return <div key={i} className="diff-side-spacer" />
        return (
          <button key={i} type="button" className="diff-collapsed" onClick={() => onToggleGroup(i)}>
            {row.collapsed.length} unchanged lines
          </button>
        )
      }
      return renderPair(row.pair, i, side, blockStarts?.has(i) ? ' diff-conflict-block-start' : '')
    })
  }

  const lineMenu = (e: MouseEvent) => {
    const row = (e.target as HTMLElement).closest<HTMLElement>('[data-line]')
    if (!onLineMenu || !row) return
    e.preventDefault()
    onLineMenu(e, Number(row.dataset.line), row.querySelector('.diff-content')?.textContent ?? '')
  }

  if (mode === 'split' && !wrap) {
    return (
      <div className="diff-pane diff-pane-sides" ref={scrollRef} onContextMenu={lineMenu}>
        <SyncedSides left={renderRows('left')} right={renderRows('right')} />
      </div>
    )
  }

  return (
    <div className={`diff-pane diff-pane-${mode}`} ref={scrollRef} onContextMenu={lineMenu}>
      {renderRows()}
    </div>
  )
}

export default DiffPane
