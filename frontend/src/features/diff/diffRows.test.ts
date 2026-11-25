import type { DiffLine } from '@current-client-bindings/app'
import {
  buildDisplayRows,
  buildPaneRows,
  columnsOf,
  copiedText,
  diffStats,
  isLongDiff,
  longestLine,
  pairLines,
  unifiedRowsForPair,
} from './diffRows'

function line(kind: 'context' | 'added' | 'removed', content: string, n = 1): DiffLine {
  return {
    kind,
    content,
    oldLine: kind === 'added' ? 0 : n,
    newLine: kind === 'removed' ? 0 : n,
    moved: false,
    inOurs: false,
    inTheirs: false,
  }
}

describe('pairLines', () => {
  it('puts a context line on both sides', () => {
    const context = line('context', 'same')
    expect(pairLines([context])).toEqual([{ left: context, right: context }])
  })

  it('pairs removed and added runs side by side, padding the shorter one', () => {
    const [r1, r2, a1] = [line('removed', 'old 1'), line('removed', 'old 2'), line('added', 'new 1')]
    expect(pairLines([r1, r2, a1])).toEqual([
      { left: r1, right: a1 },
      { left: r2, right: null },
    ])
  })

  it('keeps each conflict line as its own row', () => {
    const lines = [line('context', 'a'), { ...line('context', 'M'), kind: 'conflict-ours' }]
    expect(pairLines(lines)).toEqual([
      { left: lines[0], right: lines[0] },
      { left: lines[1], right: lines[1] },
    ])
  })
})

describe('buildDisplayRows', () => {
  const hunk = (lines: DiffLine[]) => ({ header: '@@ -1 +1 @@', raw: 'raw', lines })

  it('starts each hunk with its header', () => {
    const rows = buildDisplayRows([hunk([line('added', 'x')])])
    expect(rows[0]).toEqual({ header: '@@ -1 +1 @@', raw: 'raw' })
  })

  it('collapses a long unchanged run but keeps a short one', () => {
    const context = (count: number) => Array.from({ length: count }, (_, i) => line('context', `c${i}`, i + 1))
    const long = buildDisplayRows([hunk([line('added', 'x'), ...context(9)])])
    const short = buildDisplayRows([hunk([line('added', 'x'), ...context(8)])])

    expect(long[2]).toMatchObject({ collapsed: expect.any(Array) })
    expect(long).toHaveLength(3)
    expect(short).toHaveLength(10)
  })
})

describe('unifiedRowsForPair', () => {
  it('renders a context pair once and a changed pair as two rows', () => {
    const context = line('context', 'same')
    expect(unifiedRowsForPair({ left: context, right: context })).toHaveLength(1)
    expect(unifiedRowsForPair({ left: line('removed', 'a'), right: line('added', 'b') })).toEqual([
      expect.objectContaining({ side: 'left' }),
      expect.objectContaining({ side: 'right' }),
    ])
  })
})

describe('diffStats', () => {
  it('counts added and removed lines across hunks', () => {
    const diff = {
      conflicted: false,
      oldPath: 'a',
      newPath: 'a',
      binary: false,
      tooLarge: false,
      sizeBytes: 0,
      hunks: [
        { header: 'h1', raw: 'r1', lines: [line('added', 'x'), line('removed', 'y'), line('context', 'z')] },
        { header: 'h2', raw: 'r2', lines: [line('added', 'w')] },
      ],
    }
    expect(diffStats(diff)).toEqual({ added: 2, removed: 1, hunks: 2 })
  })
})

describe('buildPaneRows', () => {
  const hunk = {
    header: '@@',
    raw: 'raw',
    lines: [
      line('removed', 'old', 1),
      line('added', 'new', 1),
      ...Array.from({ length: 9 }, (_, i) => line('context', `same ${i}`, i + 2)),
    ],
  }
  const rows = buildDisplayRows([hunk])

  it('lists headers, pairs, collapsed lines and the blocks under a line', () => {
    const paneRows = buildPaneRows(rows, new Set(), (ref) => ref === '+1')
    expect(paneRows.map((row) => row.kind)).toEqual(['header', 'pair', 'blocks', 'collapsed'])
    expect(paneRows[3]).toMatchObject({ group: 2, count: 9 })
  })

  it('lists every line of an expanded group, under keys of their own', () => {
    const paneRows = buildPaneRows(rows, new Set([2]), () => false)
    expect(paneRows).toHaveLength(11)
    expect(new Set(paneRows.map((row) => row.key)).size).toBe(11)
  })

  it('numbers the hunks', () => {
    const paneRows = buildPaneRows(buildDisplayRows([hunk, hunk]), new Set(), () => false)
    expect(paneRows.flatMap((row) => (row.kind === 'header' ? [row.hunk] : []))).toEqual([0, 1])
  })
})

describe('a long diff', () => {
  it('counts collapsed lines, since they can be expanded', () => {
    const context = (n: number) =>
      buildDisplayRows([{ header: '', raw: '', lines: Array(n).fill(line('context', 'x')) }])
    expect(isLongDiff(context(400))).toBe(false)
    expect(isLongDiff(context(600))).toBe(true)
  })

  it('measures its longest line with tabs filled out', () => {
    expect(columnsOf('\tab')).toBe(10)
    expect(columnsOf('abc\td')).toBe(9)
    const rows = buildDisplayRows([
      { header: '', raw: '', lines: [line('removed', 'short'), line('added', '\t\tlonger')] },
    ])
    expect(longestLine(rows)).toBe(22)
  })
})

describe('copiedText', () => {
  const rows = buildPaneRows(
    buildDisplayRows([
      { header: '@@', raw: '', lines: [line('removed', 'old'), line('added', 'new'), line('context', 'same', 2)] },
    ]),
    new Set(),
    () => false,
  )

  it('takes the lines as the layout shows them, without headers', () => {
    expect(copiedText(rows, 'unified')).toBe('old\nnew\nsame')
    expect(copiedText(rows, 'split')).toBe('old\nnew\nsame\nsame')
  })

  it('takes one side where each side is its own', () => {
    expect(copiedText(rows, 'split', 'left')).toBe('old\nsame')
    expect(copiedText(rows, 'split', 'right')).toBe('new\nsame')
  })
})
