import { render } from '@testing-library/react'
import type { GraphNode } from '@current-client-bindings/app'
import CommitGraph from './CommitGraph'

const ROW = 28

function node(overrides: Partial<GraphNode>): GraphNode {
  return { sha: 'abc', lane: 0, hasChild: false, parentLanes: [0], convergingLanes: [], passThrough: [], ...overrides }
}

// The incoming segment is the one in the node's own lane ending at its centre.
function incoming(container: HTMLElement) {
  return [...container.querySelectorAll('line')].filter(
    (l) => l.getAttribute('y1') === '0' && l.getAttribute('y2') === String(ROW / 2),
  )
}

describe('CommitGraph', () => {
  it('starts a branch tip at its node, with nothing above it', () => {
    const { container } = render(<CommitGraph node={node({ hasChild: false })} laneCount={1} rowHeight={ROW} />)
    expect(incoming(container)).toHaveLength(0)
  })

  it('connects a commit to the child above it', () => {
    const { container } = render(<CommitGraph node={node({ hasChild: true })} laneCount={1} rowHeight={ROW} />)
    expect(incoming(container)).toHaveLength(1)
  })
})
