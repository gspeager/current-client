import { describe, expect, it } from 'vitest'
import type { GraphNode, RefInfo } from '@current-client-bindings/app'
import { assignLaneColors, ownerBranch } from './laneOwners'

const node = (sha: string, lane: number, parentLanes: number[], convergingLanes: number[] = []) =>
  ({ sha, lane, parentLanes, convergingLanes, passThrough: [], hasChild: false }) as unknown as GraphNode
const ref = (kind: string, name: string) => ({ kind, name, sha: '' }) as RefInfo
const branchColor = (name: string, current: boolean) => (current ? 'head' : `branch:${name}`)
const laneColor = (lane: number) => `lane:${lane}`

describe('ownerBranch', () => {
  it('prefers HEAD, then the trunk, and treats a remote branch as its local name', () => {
    expect(ownerBranch([ref('branch', 'a'), ref('branch', 'b')], 'b')).toBe('b')
    expect(ownerBranch([ref('branch', 'a'), ref('remote', 'origin/main')])).toBe('main')
    expect(ownerBranch([ref('tag', 'v1')])).toBeNull()
  })
})

describe('assignLaneColors', () => {
  it('colors a lane from the branch tip above it until the next tip', () => {
    const refs: Record<string, RefInfo[]> = { a: [ref('branch', 'work')], c: [ref('branch', 'main')] }
    const rows = assignLaneColors(
      [node('a', 0, [0]), node('b', 0, [0]), node('c', 0, [0])],
      (sha) => refs[sha],
      undefined,
      branchColor,
      laneColor,
    )
    expect(rows.map((r) => r.node)).toEqual(['branch:work', 'branch:work', 'branch:main'])
    expect(rows[2].before.get(0)).toBe('branch:work')
  })

  it('gives HEAD its own color and keeps a side lane until it converges', () => {
    const refs: Record<string, RefInfo[]> = { a: [ref('branch', 'side')], b: [ref('branch', 'cur')] }
    const rows = assignLaneColors(
      [node('a', 1, [0]), node('b', 0, [0], [1])],
      (sha) => refs[sha],
      'cur',
      branchColor,
      laneColor,
    )
    expect(rows[1].before.get(1)).toBe('branch:side')
    expect(rows[1].node).toBe('head')
    expect(rows[1].after.has(1)).toBe(false)
  })

  it('falls back to the lane position for a lane with no tip', () => {
    const rows = assignLaneColors([node('a', 0, [0, 2])], () => undefined, undefined, branchColor, laneColor)
    expect(rows[0].node).toBe('lane:0')
    expect(rows[0].after.get(2)).toBe('lane:2')
  })
})
