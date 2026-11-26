import clsx from 'clsx'
import Kbd from '../../components/controls/Kbd'
import {
  Fragment,
  useCallback,
  useImperativeHandle,
  useLayoutEffect,
  useMemo,
  useRef,
  type ClipboardEvent,
  type CSSProperties,
  type Key,
  type MouseEvent,
  type ReactNode,
  type Ref,
  type RefObject,
  type UIEvent,
} from 'react'
import { ChevronsUpDown, MessageSquarePlus } from 'lucide-react'
import type { DiffLine, LineSelection } from '@current-client-bindings/app'
import { plural } from '../../lib/plural'
import { useDiffWrap } from './diffWrap'
import {
  buildPaneRows,
  columnsOf,
  copiedText,
  isLongDiff,
  longestLine,
  unifiedRowsForPair,
  type DisplayRow,
  type PairedLine,
  type PaneRow,
  type UnifiedRow,
} from './diffRows'
import { languageForPath, tokenizeLine } from './syntaxHighlight'
import { selectedRows, useDiffWindow } from './useDiffWindow'
import { diffLineWords } from './wordDiff'
import './DiffPane.scss'

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

// The comment button sits over the gutter, shown while the row is hovered or
// it has focus, so the row grid stays as it is.
export interface LineComment {
  // What the button says before the line.
  label: string
  // Whether a removed line takes one.
  removed: boolean
  run: (line: number, oldLine: number) => void
}

function CommentButton({ line, oldLine, comment }: { line: number; oldLine: number; comment: LineComment }) {
  const where = line ? `line ${line}` : `old line ${oldLine}`
  return (
    <button
      type="button"
      className="diff-line-comment"
      onClick={() => comment.run(line, oldLine)}
      aria-label={`${comment.label} ${where}`}
      title={`${comment.label} ${where}`}
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
  comment?: LineComment,
) {
  const line = side === 'left' ? pair.left : pair.right
  // The new side takes comments; the old side only for a line that is gone.
  const commentable =
    comment && line && (side === 'right' ? line.newLine > 0 : comment.removed && line.kind === 'removed')
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
          comment={comment}
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
  comment?: LineComment,
) {
  const activeLine = side === 'left' ? pair.left : pair.right
  const isContext = pair.left !== null && pair.left === pair.right
  const removed = !isContext && side === 'left' && activeLine?.kind === 'removed'
  const commentable = comment && activeLine && (removed ? comment.removed : (activeLine.newLine ?? 0) > 0)
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
          comment={comment}
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
  const what = count === 0 ? 'hunk' : plural(count, 'line')
  const toggleLabel = `${onStageHunk ? 'Stage' : 'Unstage'} ${what}`
  const discardLabel = count ? `Discard ${what}` : 'Discard'
  const run = (action: (hunkRaw: string, lines?: LineSelection) => void) => (count ? action(raw, lines) : action(raw))
  return (
    <span className="diff-hunk-actions">
      <button type="button" className="diff-stage-hunk" aria-label={toggleLabel} onClick={() => run(toggle)}>
        {toggleLabel}
        <Kbd>{onStageHunk ? 'S' : 'U'}</Kbd>
      </button>
      {onDiscardHunk && (count === 0 || canDiscardLines) && (
        <button
          type="button"
          className="diff-discard-hunk"
          aria-label={discardLabel}
          onClick={() => run(onDiscardHunk)}
        >
          {discardLabel}
          <Kbd>D</Kbd>
        </button>
      )}
    </span>
  )
}

// Wrap lines off in split view: each side scrolls on its own and scrolling one
// moves the other. Both take the wider side's width so they scroll the same
// distance and their rows stay aligned.
function SyncedSides({
  left,
  right,
  leftRef,
}: {
  left: ReactNode
  right: ReactNode
  leftRef: RefObject<HTMLDivElement>
}) {
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

// Brings rows into view, whether or not they're drawn.
export interface DiffPaneHandle {
  // False when the diff doesn't show the line.
  revealLine: (line: number, oldLine?: number) => boolean
  revealHunk: (hunk: number) => void
  revealBlock: (block: number) => void
  revealEdge: (edge: 'first' | 'last') => void
  // The nearest row with a changed line, after or before this one. False when
  // it's already drawn, or there is none.
  revealChange: (from: number, forward: boolean) => boolean
}

interface DiffPaneProps extends HunkActions {
  mode: 'split' | 'unified'
  rows: DisplayRow[]
  path: string
  expandedGroups: Set<number>
  onToggleGroup: (index: number) => void
  currentHunkIndex: number
  scrollRef: RefObject<HTMLDivElement>
  handle?: Ref<DiffPaneHandle>
  // Present when lines can be selected for hunk actions.
  lineSelect?: LineSelect & { byHunk: Map<string, LineSelection> }
  // Blocks placed under a line, by lineRef: review threads, drafted
  // comments, the composer.
  blocksByLine?: Map<string, ReactNode[]>
  // Present when a line can take a review comment.
  lineComment?: LineComment
  // Present when a new-side line's context menu has something to offer.
  onLineMenu?: (e: MouseEvent, line: number, content: string) => void
  // Rows that open a conflict block, marked so the footer can walk to them.
  blockStarts?: Set<number>
  // The conflicted file as it is: one line-number column, the working file's.
  resultView?: boolean
  onScrollTopChange?: (top: number) => void
}

const ROW_HEIGHT = 20
// Only for rows that haven't been drawn; a drawn row is measured.
const BLOCKS_ESTIMATE = 120
const CHARACTER_WIDTH = 7.8
const UNIFIED_GUTTERS = 102
const SPLIT_GUTTERS = 62

// Split mode is a grid for each row, so a hunk header can span both columns,
// unless lines don't wrap and each side scrolls sideways on its own.
function DiffPane(props: DiffPaneProps) {
  const {
    mode,
    rows,
    path,
    expandedGroups,
    onToggleGroup,
    currentHunkIndex,
    scrollRef,
    handle,
    lineSelect,
    blocksByLine,
    lineComment,
    onLineMenu,
    blockStarts,
    resultView = false,
    onScrollTopChange,
    ...hunkActions
  } = props
  const { wrap } = useDiffWrap()
  const leftRef = useRef<HTMLDivElement>(null)
  const sidesApart = mode === 'split' && !wrap
  const scroller = () => (sidesApart ? leftRef.current : scrollRef.current)
  const language = languageForPath(path)

  // With each side rendered on its own a block can't span both, so DiffViewer
  // lists them below the diff instead, and no line takes a comment.
  const paneRows = useMemo(
    () => buildPaneRows(rows, expandedGroups, (ref) => !sidesApart && !!blocksByLine?.has(ref), blockStarts),
    [rows, expandedGroups, sidesApart, blocksByLine, blockStarts],
  )
  const windowed = useMemo(() => isLongDiff(rows), [rows])
  const longest = useMemo(() => (windowed && !wrap ? longestLine(rows) : 0), [windowed, wrap, rows])

  const linesIn = (text: string, width: number) => {
    if (!wrap || width === 0) return 1
    const room = mode === 'split' ? width / 2 - SPLIT_GUTTERS : width - UNIFIED_GUTTERS
    return Math.max(1, Math.ceil(columnsOf(text) / Math.max(1, Math.floor(room / CHARACTER_WIDTH))))
  }
  const estimate = (row: PaneRow, width: number): number => {
    if (row.kind === 'blocks') return BLOCKS_ESTIMATE
    if (row.kind !== 'pair') return ROW_HEIGHT
    const { left, right } = row.pair
    if (mode === 'split')
      return ROW_HEIGHT * Math.max(linesIn(left?.content ?? '', width), linesIn(right?.content ?? '', width))
    return (
      ROW_HEIGHT *
      unifiedRowsForPair(row.pair).reduce((sum, { side }) => sum + linesIn(row.pair[side]!.content, width), 0)
    )
  }
  const drawn = useDiffWindow({
    windowed,
    count: paneRows.length,
    keyOf: useCallback((index) => paneRows[index].key, [paneRows]),
    layout: `${mode}-${wrap}`,
    heightOf: (index, width) => estimate(paneRows[index], width),
    measured: (index) => wrap || paneRows[index]?.kind === 'blocks',
    keep: useMemo(() => paneRows.flatMap((row, i) => (row.kind === 'blocks' ? [i] : [])), [paneRows]),
    scroller,
    paneRef: scrollRef,
  })

  const revealRow = (index: number, block: 'center' | 'nearest') => {
    if (index >= 0) drawn.reveal(index, block)
    return index >= 0
  }
  useImperativeHandle(handle, () => ({
    revealLine: (line, oldLine = 0) =>
      revealRow(
        paneRows.findIndex(
          (row) =>
            row.kind === 'pair' &&
            (line
              ? row.pair.left?.newLine === line || row.pair.right?.newLine === line
              : row.pair.left?.oldLine === oldLine),
        ),
        'center',
      ),
    revealHunk: (hunk) =>
      revealRow(
        paneRows.findIndex((row) => row.kind === 'header' && row.hunk === hunk),
        'nearest',
      ),
    revealBlock: (block) => {
      const starts = paneRows.flatMap((row, i) => (row.kind === 'pair' && row.startsBlock ? [i] : []))
      revealRow(starts[block] ?? -1, 'center')
    },
    revealEdge: (edge) => revealRow(edge === 'first' ? 0 : paneRows.length - 1, 'nearest'),
    revealChange: (from, forward) => {
      const step = forward ? 1 : -1
      for (let i = from + step; i >= 0 && i < paneRows.length; i += step) {
        const row = paneRows[i]
        if (row.kind === 'pair' && (lineKey(row.pair.left) || lineKey(row.pair.right))) {
          return !drawn.indexes.includes(i) && revealRow(i, 'nearest')
        }
      }
      return false
    },
  }))

  // Capture, since scroll events don't bubble up from the two sides. Anything
  // else that scrolls inside the pane, such as a reply box, isn't the diff.
  const reportScroll = (e: UIEvent<HTMLDivElement>) => {
    const target = e.target as HTMLElement
    if (target === e.currentTarget || target.classList.contains('diff-side')) onScrollTopChange?.(target.scrollTop)
  }

  const comment = resultView ? undefined : lineComment
  const renderPair = (pair: PairedLine, side: Side | undefined, extraClass: string): ReactNode => {
    if (side) return splitRow(pair, side, language, side, lineSelect, extraClass)
    if (mode === 'split') {
      return (['left', 'right'] as const).map((s) => splitRow(pair, s, language, s, lineSelect, extraClass, comment))
    }
    return unifiedRowsForPair(pair).map((row, k) =>
      unifiedRow(row, language, k, lineSelect, k === 0 ? extraClass : '', resultView, comment),
    )
  }

  // With a side, renders just that side; the right one gets blank rows where
  // the left has a hunk header or collapsed lines.
  const renderRow = (row: PaneRow, side?: Side): ReactNode => {
    switch (row.kind) {
      case 'header':
        if (side === 'right') return <div className="diff-side-spacer" />
        return (
          <div className={clsx('diff-hunk-header', row.hunk === currentHunkIndex && 'diff-hunk-header-current')}>
            <span>{row.header}</span>
            <HunkActionButtons raw={row.raw} lines={lineSelect?.byHunk.get(row.raw)} {...hunkActions} />
          </div>
        )
      case 'collapsed':
        if (side === 'right') return <div className="diff-side-spacer" />
        return (
          <button type="button" className="diff-collapsed" onClick={() => onToggleGroup(row.group)}>
            <ChevronsUpDown size={12} strokeWidth={1.75} />
            {row.count} unchanged lines
          </button>
        )
      case 'blocks':
        return row.refs.flatMap((ref) => blocksByLine?.get(ref) ?? [])
      case 'pair':
        return renderPair(row.pair, side, row.startsBlock ? ' diff-conflict-block-start' : '')
    }
  }

  // Scrolling redraws the pane with the same props, so a row already built is
  // used again: highlighting and comparing words is most of its cost.
  // eslint-disable-next-line react-hooks/exhaustive-deps
  const built = useMemo(() => new Map<string, ReactNode>(), [props, wrap, paneRows])
  const renderRows = (side?: Side) => {
    const elements: ReactNode[] = []
    let end = 0
    const gap = (height: number, key: string) => {
      if (height > 0) elements.push(<div key={key} className="diff-gap" style={{ height }} />)
    }
    for (const index of drawn.indexes) {
      const row = paneRows[index]
      if (!row) continue
      gap(drawn.startOf(index) - end, `gap-${row.key}`)
      end = drawn.endOf(index)
      const cacheKey = `${side ?? ''}${row.key}`
      if (!built.has(cacheKey)) {
        built.set(
          cacheKey,
          <div key={row.key} className="diff-item" data-index={index} ref={drawn.measureRow(index)}>
            {renderRow(row, side)}
          </div>,
        )
      }
      elements.push(built.get(cacheKey))
    }
    gap(drawn.total - end, 'gap-end')
    return elements
  }

  const lineMenu = (e: MouseEvent) => {
    const row = (e.target as HTMLElement).closest<HTMLElement>('[data-line]')
    if (!onLineMenu || !row) return
    e.preventDefault()
    onLineMenu(e, Number(row.dataset.line), row.querySelector('.diff-content')?.textContent ?? '')
  }

  // A selection across rows that aren't drawn has no text in the page to copy.
  const copy = (e: ClipboardEvent) => {
    const selected = windowed ? selectedRows(scrollRef.current) : null
    if (!selected) return
    const [first, last] = selected
    const inView = new Set(drawn.indexes)
    let complete = true
    for (let i = first; i <= last && complete; i++) complete = inView.has(i)
    if (complete) return
    e.preventDefault()
    const side = !sidesApart
      ? undefined
      : leftRef.current?.contains(document.getSelection()!.anchorNode)
        ? 'left'
        : 'right'
    e.clipboardData.setData('text/plain', copiedText(paneRows.slice(first, last + 1), mode, side))
  }

  return (
    <div
      className={clsx('diff-pane', `diff-pane-${sidesApart ? 'sides' : mode}`, windowed && 'diff-pane-long')}
      ref={scrollRef}
      tabIndex={0}
      style={longest ? ({ '--longest-line': `${longest}ch` } as CSSProperties) : undefined}
      onContextMenu={lineMenu}
      onCopy={copy}
      onScrollCapture={reportScroll}
    >
      {sidesApart ? (
        <SyncedSides left={renderRows('left')} right={renderRows('right')} leftRef={leftRef} />
      ) : (
        renderRows()
      )}
    </div>
  )
}

export default DiffPane
