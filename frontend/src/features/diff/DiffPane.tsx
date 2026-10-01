import { Fragment, type Key, type ReactNode, type RefObject } from 'react'
import type { DiffLine } from '@current-client-bindings/app'
import { unifiedRowsForPair, type DisplayRow, type PairedLine, type UnifiedRow } from './diffRows'
import { languageForPath, tokenizeLine } from './syntaxHighlight'
import { diffLineWords } from './wordDiff'

type Side = 'left' | 'right'

export interface HunkActions {
  onStageHunk?: (hunkRaw: string) => void
  onUnstageHunk?: (hunkRaw: string) => void
  onDiscardHunk?: (hunkRaw: string) => void
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

function splitRow(pair: PairedLine, side: Side, language: string | null, key: Key) {
  const line = side === 'left' ? pair.left : pair.right
  return (
    <div key={key} className={`${rowClass(line)} diff-row-split-${side}`}>
      <span className="diff-line-number">{lineNumber(line, side)}</span>
      <span className="diff-marker">{marker(line)}</span>
      <DiffContent pair={pair} side={side} language={language} />
    </div>
  )
}

function unifiedRow({ pair, side }: UnifiedRow, language: string | null, key: Key) {
  const activeLine = side === 'left' ? pair.left : pair.right
  const isContext = pair.left !== null && pair.left === pair.right
  return (
    <div key={key} className={`${rowClass(activeLine)} diff-row-unified`}>
      <span className="diff-line-number">{isContext || side === 'left' ? lineNumber(pair.left, 'left') : ''}</span>
      <span className="diff-line-number">{isContext || side === 'right' ? lineNumber(pair.right, 'right') : ''}</span>
      <span className="diff-marker">{marker(activeLine)}</span>
      <DiffContent pair={pair} side={side} language={language} />
    </div>
  )
}

function HunkActionButtons({ raw, onStageHunk, onUnstageHunk, onDiscardHunk }: HunkActions & { raw: string }) {
  const toggle = onStageHunk ?? onUnstageHunk
  if (!toggle) return null
  return (
    <span className="diff-hunk-actions">
      <button type="button" className="diff-stage-hunk" onClick={() => toggle(raw)}>
        {onStageHunk ? 'Stage hunk' : 'Unstage hunk'}
      </button>
      {onDiscardHunk && (
        <button type="button" className="diff-discard-hunk" onClick={() => onDiscardHunk(raw)}>
          Discard
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
  ...hunkActions
}: DiffPaneProps) {
  const language = languageForPath(path)
  const renderPair = (pair: PairedLine, key: Key): ReactNode[] =>
    mode === 'split'
      ? [splitRow(pair, 'left', language, `${key}-l`), splitRow(pair, 'right', language, `${key}-r`)]
      : unifiedRowsForPair(pair).map((row, k) => unifiedRow(row, language, `${key}-${k}`))

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
              <HunkActionButtons raw={row.raw} {...hunkActions} />
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
