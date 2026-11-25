import type { DiffHunk, DiffLine, FileDiff } from '@current-client-bindings/app'

export interface PairedLine {
  left: DiffLine | null
  right: DiffLine | null
}

export type DisplayRow = { header: string; raw: string } | { pair: PairedLine } | { collapsed: PairedLine[] }

export interface UnifiedRow {
  pair: PairedLine
  side: 'left' | 'right'
}

interface DiffStats {
  added: number
  removed: number
  hunks: number
}

const COLLAPSE_THRESHOLD = 8

export function diffStats(diff: FileDiff): DiffStats {
  let added = 0
  let removed = 0
  for (const hunk of diff.hunks) {
    for (const line of hunk.lines) {
      if (line.kind === 'added') added++
      else if (line.kind === 'removed') removed++
    }
  }
  return { added, removed, hunks: diff.hunks.length }
}

// Assumes git's unified diff groups each changed region as a run of removed
// lines followed by a run of added lines, never interleaved. Context and
// conflict lines stand alone.
export function pairLines(lines: DiffLine[]): PairedLine[] {
  const rows: PairedLine[] = []
  let i = 0
  while (i < lines.length) {
    const line = lines[i]
    if (line.kind !== 'removed' && line.kind !== 'added') {
      rows.push({ left: line, right: line })
      i++
      continue
    }
    const removed: DiffLine[] = []
    while (i < lines.length && lines[i].kind === 'removed') {
      removed.push(lines[i])
      i++
    }
    const added: DiffLine[] = []
    while (i < lines.length && lines[i].kind === 'added') {
      added.push(lines[i])
      i++
    }
    const count = Math.max(removed.length, added.length)
    for (let j = 0; j < count; j++) {
      rows.push({ left: removed[j] ?? null, right: added[j] ?? null })
    }
  }
  return rows
}

function isPureContext(pair: PairedLine): boolean {
  return pair.left?.kind === 'context' && pair.right?.kind === 'context'
}

function groupCollapsibleRuns(pairs: PairedLine[]): DisplayRow[] {
  const rows: DisplayRow[] = []
  let i = 0
  while (i < pairs.length) {
    if (!isPureContext(pairs[i])) {
      rows.push({ pair: pairs[i] })
      i++
      continue
    }
    let j = i
    while (j < pairs.length && isPureContext(pairs[j])) j++
    const run = pairs.slice(i, j)
    if (run.length > COLLAPSE_THRESHOLD) {
      rows.push({ collapsed: run })
    } else {
      for (const pair of run) rows.push({ pair })
    }
    i = j
  }
  return rows
}

export function buildDisplayRows(hunks: DiffHunk[]): DisplayRow[] {
  const rows: DisplayRow[] = []
  for (const hunk of hunks) {
    rows.push({ header: hunk.header, raw: hunk.raw })
    rows.push(...groupCollapsibleRuns(pairLines(hunk.lines)))
  }
  return rows
}

// A context line is one unified row; a changed pair becomes a removed row then an added row.
export function unifiedRowsForPair(pair: PairedLine): UnifiedRow[] {
  if (pair.left !== null && pair.left === pair.right) {
    return [{ pair, side: 'left' }]
  }
  const rows: UnifiedRow[] = []
  if (pair.left) rows.push({ pair, side: 'left' })
  if (pair.right) rows.push({ pair, side: 'right' })
  return rows
}

// One drawn row of the pane, whatever its layout.
export type PaneRow =
  | { kind: 'header'; key: string; hunk: number; header: string; raw: string }
  | { kind: 'collapsed'; key: string; group: number; count: number }
  | { kind: 'pair'; key: string; pair: PairedLine; startsBlock: boolean }
  | { kind: 'blocks'; key: string; refs: string[] }

// Names a line for blocks placed under it: a new-side line as +n, a removed
// line as -n.
export function lineRef(line: number, oldLine: number): string {
  return line ? `+${line}` : `-${oldLine}`
}

function blockRefs({ left, right }: PairedLine, has: (ref: string) => boolean): string[] {
  const refs: string[] = []
  if (left && left !== right && left.kind === 'removed') refs.push(lineRef(0, left.oldLine))
  if (right?.newLine) refs.push(lineRef(right.newLine, 0))
  return refs.filter(has)
}

export function buildPaneRows(
  rows: DisplayRow[],
  expandedGroups: Set<number>,
  hasBlocks: (ref: string) => boolean,
  blockStarts?: Set<number>,
): PaneRow[] {
  const paneRows: PaneRow[] = []
  const addPair = (pair: PairedLine, key: string, startsBlock: boolean) => {
    paneRows.push({ kind: 'pair', key, pair, startsBlock })
    const refs = blockRefs(pair, hasBlocks)
    if (refs.length > 0) paneRows.push({ kind: 'blocks', key: `${key}-blocks`, refs })
  }
  let hunk = 0
  rows.forEach((row, i) => {
    if ('header' in row) paneRows.push({ kind: 'header', key: `${i}`, hunk: hunk++, header: row.header, raw: row.raw })
    else if ('pair' in row) addPair(row.pair, `${i}`, blockStarts?.has(i) ?? false)
    else if (expandedGroups.has(i)) row.collapsed.forEach((pair, j) => addPair(pair, `${i}-${j}`, false))
    else paneRows.push({ kind: 'collapsed', key: `${i}`, group: i, count: row.collapsed.length })
  })
  return paneRows
}

// Past this many rows a diff draws only what's on screen.
const LONG_DIFF = 500

export function isLongDiff(rows: DisplayRow[]): boolean {
  let count = 0
  for (const row of rows) count += 'collapsed' in row ? row.collapsed.length : 1
  return count > LONG_DIFF
}

const TAB_SIZE = 8

// How many characters wide a line is, with its tabs filled out.
export function columnsOf(text: string): number {
  if (!text.includes('\t')) return text.length
  let columns = 0
  for (const char of text) columns += char === '\t' ? TAB_SIZE - (columns % TAB_SIZE) : 1
  return columns
}

export function longestLine(rows: DisplayRow[]): number {
  let longest = 0
  const measure = ({ left, right }: PairedLine) => {
    if (left) longest = Math.max(longest, columnsOf(left.content))
    if (right && right !== left) longest = Math.max(longest, columnsOf(right.content))
  }
  for (const row of rows) {
    if ('pair' in row) measure(row.pair)
    else if ('collapsed' in row) row.collapsed.forEach(measure)
  }
  return longest
}

// The text of a run of rows, as selecting them copies it: one side's, when
// each side is drawn on its own.
export function copiedText(rows: PaneRow[], mode: 'split' | 'unified', side?: 'left' | 'right'): string {
  return rows
    .flatMap((row) => {
      if (row.kind !== 'pair') return []
      if (mode === 'unified') return unifiedRowsForPair(row.pair).map((unified) => row.pair[unified.side]!.content)
      const sides = side ? [row.pair[side]] : [row.pair.left, row.pair.right]
      return sides.flatMap((line) => (line ? [line.content] : []))
    })
    .join('\n')
}
