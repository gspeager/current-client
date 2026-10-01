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
