import { describe, expect, it } from 'vitest'
import { buildAsciiGraph } from './asciiGraph'

describe('buildAsciiGraph', () => {
  it('returns an empty string for no rows', () => {
    expect(buildAsciiGraph([])).toBe('')
  })

  it('places a single-lane history in one column', () => {
    const got = buildAsciiGraph([
      { sha: 'aaaaaaaaaa', subject: 'second', lane: 0, passThrough: [] },
      { sha: 'bbbbbbbbbb', subject: 'first', lane: 0, passThrough: [] },
    ])
    expect(got).toBe('*  aaaaaaa second\n*  bbbbbbb first')
  })

  it('marks pass-through lanes alongside the commit lane', () => {
    const got = buildAsciiGraph([
      { sha: 'aaaaaaaaaa', subject: 'on lane 1', lane: 1, passThrough: [0] },
      { sha: 'bbbbbbbbbb', subject: 'on lane 0', lane: 0, passThrough: [] },
    ])
    // Row "bbbbbbb" is lane 0 with no pass-through, but the graph still has
    // 2 columns overall (lane 1 is used elsewhere), so its second column is
    // a blank space, not omitted.
    expect(got).toBe('| *  aaaaaaa on lane 1\n*    bbbbbbb on lane 0')
  })

  it('truncates the sha to 7 characters', () => {
    const got = buildAsciiGraph([{ sha: '0123456789abcdef', subject: 'x', lane: 0, passThrough: [] }])
    expect(got).toBe('*  0123456 x')
  })
})
