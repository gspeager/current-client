import type { DiffLine } from '@current-client-bindings/app'
import { buildDisplayRows, diffStats, pairLines, unifiedRowsForPair } from './diffRows'

function line(kind: 'context' | 'added' | 'removed', content: string, n = 1): DiffLine {
  return { kind, content, oldLine: kind === 'added' ? 0 : n, newLine: kind === 'removed' ? 0 : n, moved: false }
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
