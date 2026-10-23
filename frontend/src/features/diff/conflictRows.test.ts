import type { DiffLine } from '@current-client-bindings/app'
import { conflictRows } from './conflictRows'

const line = (kind: string, content: string, inOurs: boolean, inTheirs: boolean, newLine = 0): DiffLine => ({
  kind,
  oldLine: 0,
  newLine,
  content,
  moved: false,
  inOurs,
  inTheirs,
})

const lines = [
  line('context', 'shared', true, true, 1),
  line('removed', 'ours dropped', true, false),
  line('context', 'theirs added', false, true, 2),
  line('conflict-marker', '<<<<<<< HEAD', false, false, 3),
  line('conflict-ours', 'A', true, false, 4),
  line('conflict-ours', 'B', true, false, 5),
  line('conflict-marker', '=======', false, false, 6),
  line('conflict-theirs', 'X', false, true, 7),
  line('conflict-marker', '>>>>>>> feature', false, false, 8),
  line('context', 'tail', true, true, 9),
]
const hunks = [{ header: '@@@ -1,4 -1,3 +1,9 @@@', raw: '', lines }]

describe('conflictRows', () => {
  it('keeps the working file as it is in the Result view, with no headers, and marks where blocks start', () => {
    const { rows, blockStarts } = conflictRows(hunks, 'result')
    expect(rows).toHaveLength(lines.length)
    expect(rows.every((r) => 'pair' in r)).toBe(true)
    expect(blockStarts).toEqual([3])
  })

  it('pairs ours against theirs in the Side by side view and drops the markers', () => {
    const { rows, blockStarts } = conflictRows(hunks, 'side-by-side')
    const text = rows.map((r) => ('pair' in r ? [r.pair.left?.content ?? null, r.pair.right?.content ?? null] : r))
    expect(text).toEqual([
      ['shared', 'shared'],
      ['ours dropped', 'theirs added'],
      ['A', 'X'],
      ['B', null],
      ['tail', 'tail'],
    ])
    expect(blockStarts).toEqual([2])
  })
})
