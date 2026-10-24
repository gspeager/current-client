import type { DiffHunk, DiffLine } from '@current-client-bindings/app'
import type { DisplayRow } from './diffRows'

export type ConflictView = 'side-by-side' | 'result'

export interface ConflictRows {
  rows: DisplayRow[]
  // Row indexes where each conflict block starts, for walking them.
  blockStarts: number[]
}

const isMarker = (line: DiffLine) => line.kind === 'conflict-marker'
const opensBlock = (line: DiffLine) => isMarker(line) && line.content.startsWith('<<<<<<<')

// The working file as it is, markers included, with no hunk headers.
function resultRows(lines: DiffLine[]): ConflictRows {
  const rows: DisplayRow[] = []
  const blockStarts: number[] = []
  for (const line of lines) {
    if (opensBlock(line)) blockStarts.push(rows.length)
    rows.push({ pair: { left: line, right: line } })
  }
  return { rows, blockStarts }
}

// Ours on the left, theirs on the right. Lines both sides have keep the two
// columns in step; between them each side's own lines are paired by position.
function sideBySideRows(lines: DiffLine[]): ConflictRows {
  const rows: DisplayRow[] = []
  const blockStarts: number[] = []
  let left: DiffLine[] = []
  let right: DiffLine[] = []
  const flush = () => {
    for (let i = 0; i < Math.max(left.length, right.length); i++) {
      rows.push({ pair: { left: left[i] ?? null, right: right[i] ?? null } })
    }
    left = []
    right = []
  }
  for (const line of lines) {
    if (isMarker(line)) {
      if (opensBlock(line)) {
        flush()
        blockStarts.push(rows.length)
      } else if (line.content.startsWith('>>>>>>>')) {
        flush()
      }
      continue
    }
    if (line.inOurs && line.inTheirs) {
      flush()
      rows.push({ pair: { left: line, right: line } })
    } else if (line.inOurs) {
      left.push(line)
    } else if (line.inTheirs) {
      right.push(line)
    }
  }
  flush()
  return { rows, blockStarts: blockStarts.filter((start) => start < rows.length) }
}

export function conflictRows(hunks: DiffHunk[], view: ConflictView): ConflictRows {
  const lines = hunks.flatMap((h) => h.lines)
  return view === 'result' ? resultRows(lines) : sideBySideRows(lines)
}
