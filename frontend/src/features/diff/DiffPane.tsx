import { Fragment, type Key, type ReactNode, type RefObject } from 'react'
import type { DiffLine, LineSelection } from '@current-client-bindings/app'
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

function splitRow(pair: PairedLine, side: Side, language: string | null, key: Key, select?: LineSelect) {
  const line = side === 'left' ? pair.left : pair.right
  return (
    <div key={key} className={`${rowClass(line)} diff-row-split-${side}${selectedClass(line, select)}`}>
      <LineNumber line={line} side={side} select={select} />
      <span className="diff-marker">{marker(line)}</span>
      <DiffContent pair={pair} side={side} language={language} />
    </div>
  )
}

function unifiedRow({ pair, side }: UnifiedRow, language: string | null, key: Key, select?: LineSelect) {
  const activeLine = side === 'left' ? pair.left : pair.right
  const isContext = pair.left !== null && pair.left === pair.right
  return (
    <div key={key} className={`${rowClass(activeLine)} diff-row-unified${selectedClass(activeLine, select)}`}>
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
}

// Split mode is one grid rather than two synced panes, so hunk headers can span both columns.
function DiffPane({
  mode,
  rows,
  path,
  expandedGroups,
  onToggleGroup,
  currentHunkIndex,
  scrollRef,
  lineSelect,
  ...hunkActions
}: DiffPaneProps) {
  const language = languageForPath(path)
  const renderPair = (pair: PairedLine, key: Key): ReactNode[] =>
    mode === 'split'
      ? [
          splitRow(pair, 'left', language, `${key}-l`, lineSelect),
          splitRow(pair, 'right', language, `${key}-r`, lineSelect),
        ]
      : unifiedRowsForPair(pair).map((row, k) => unifiedRow(row, language, `${key}-${k}`, lineSelect))

  let hunkIndex = -1
  return (
    <div className={`diff-pane diff-pane-${mode}`} ref={scrollRef}>
      {rows.flatMap<ReactNode>((row, i) => {
        if ('header' in row) {
          hunkIndex++
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
            return row.collapsed.flatMap((pair, j) => renderPair(pair, `${i}-${j}`))
          }
          return (
            <button key={i} type="button" className="diff-collapsed" onClick={() => onToggleGroup(i)}>
              {row.collapsed.length} unchanged lines
            </button>
          )
        }
        return renderPair(row.pair, i)
      })}
    </div>
  )
}

export default DiffPane
