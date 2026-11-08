import type { GraphNode, RefInfo } from '@current-client-bindings/app'

export interface RowLaneColors {
  before: Map<number, string>
  after: Map<number, string>
  node: string
}

const MAIN_NAMES = new Set(['main', 'master', 'trunk'])

export function withoutRemote(ref: RefInfo): string {
  return ref.kind === 'remote' ? ref.name.slice(ref.name.indexOf('/') + 1) : ref.name
}

// The branch a commit's row is drawn for: HEAD's, then the trunk, then the first listed.
export function ownerBranch(refs: RefInfo[], currentBranch?: string): string | null {
  const names = refs.filter((r) => r.kind === 'branch' || r.kind === 'remote').map(withoutRemote)
  return names.find((n) => n === currentBranch) ?? names.find((n) => MAIN_NAMES.has(n)) ?? names[0] ?? null
}

// A lane carries the color of the nearest branch tip above it, so a branch
// has one color across the sidebar, its pill, its nodes and its line. Lanes
// with no tip in the list, such as a merged branch that was deleted, take a
// color by position.
export function assignLaneColors(
  nodes: GraphNode[],
  refsFor: (sha: string) => RefInfo[] | undefined,
  currentBranch: string | undefined,
  branchColor: (name: string, isCurrent: boolean) => string,
  laneColor: (lane: number) => string,
): RowLaneColors[] {
  const owners = new Map<number, string>()
  return nodes.map((node) => {
    const before = new Map(owners)
    const owner = ownerBranch(refsFor(node.sha) ?? [], currentBranch)
    const color = owner ? branchColor(owner, owner === currentBranch) : (owners.get(node.lane) ?? laneColor(node.lane))
    node.convergingLanes.forEach((lane) => owners.delete(lane))
    owners.set(node.lane, color)
    for (const lane of node.parentLanes) {
      if (!owners.has(lane)) owners.set(lane, laneColor(lane))
    }
    return { before, after: new Map(owners), node: color }
  })
}
