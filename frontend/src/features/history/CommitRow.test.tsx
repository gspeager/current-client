import { render, screen, within } from '@testing-library/react'
import { BranchChange, CIMark as CIMarkInfo, type CommitInfo, RefInfo } from '@current-client-bindings/app'
import CommitRow from './CommitRow'

const commit: CommitInfo = {
  sha: 'b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0c1',
  parentShas: [],
  authorName: 'Ada Lovelace',
  authorEmail: 'ada@example.com',
  date: '2026-01-10T12:00:00Z',
  subject: 'Fix login redirect',
  body: '',
  signature: 'none',
  coAuthors: [],
}

function renderRow(props: Partial<Parameters<typeof CommitRow>[0]> = {}) {
  render(
    <CommitRow
      commit={commit}
      laneCount={1}
      rowHeight={38}
      selected={false}
      style={{}}
      onSelect={() => {}}
      onContextMenu={() => {}}
      repoPath="/repos/app"
      {...props}
    />,
  )
  return screen.getByRole('button', { name: /Fix login redirect/ })
}

describe('CommitRow', () => {
  it('puts the hash, CI and time in their own cells, keeping the CI cell when there is no result', () => {
    const row = renderRow()
    const cells = [...row.children].map((c) => c.className)
    expect(cells).toEqual([
      'commit-row-graph',
      'commit-row-commit',
      'commit-row-author',
      'commit-row-sha',
      'commit-row-ci',
      'commit-row-time',
      'commit-row-actions',
    ])
    expect(row.querySelector('.commit-row-sha')).toHaveTextContent('b2c3d4e')
    expect(row.querySelector('.commit-row-ci')).toBeEmptyDOMElement()
  })

  // The stylesheet applies in tests: at rest the time shows and the button
  // is hidden; hover and focus-within swap them, which jsdom cannot drive.
  it('keeps the More actions button hidden in the time’s place until the row is hovered', () => {
    const row = renderRow()
    expect(row.querySelector('.commit-row-time')).toBeVisible()
    expect(within(row).getByRole('button', { name: 'More actions for b2c3d4e', hidden: true })).not.toBeVisible()
  })

  it('shows the CI icon in its cell', () => {
    const row = renderRow({ ci: new CIMarkInfo({ status: 'success', pipelineId: 3 }) })
    expect(within(row.querySelector('.commit-row-ci')!).getByRole('button', { name: 'CI passed' })).toBeInTheDocument()
  })

  it('shows a branch and its upstream at the same commit as one pill naming both, with the upstream’s change', () => {
    const refs = [
      new RefInfo({ kind: 'remote', name: 'origin/feature', sha: commit.sha }),
      new RefInfo({ kind: 'branch', name: 'feature', sha: commit.sha }),
    ]
    const change = new BranchChange({ number: 7, prefix: '#', state: 'open' })
    const row = renderRow({ refs, upstreams: { feature: 'origin/feature' }, changes: { 'origin/feature': change } })
    const pills = row.querySelectorAll('.ref-badge')
    expect(pills).toHaveLength(1)
    expect(pills[0]).toHaveTextContent('feature')
    expect(pills[0]).toHaveAttribute('title', 'feature · origin/feature')
    expect(within(pills[0] as HTMLElement).getByLabelText('origin/feature')).toBeInTheDocument()
    expect(within(row).getByRole('button', { name: '#7 · open' })).toBeInTheDocument()
  })

  it('keeps two pills when the branch and its upstream are not the same commit', () => {
    const refs = [
      new RefInfo({ kind: 'remote', name: 'origin/feature', sha: commit.sha }),
      new RefInfo({ kind: 'branch', name: 'feature', sha: commit.sha }),
    ]
    const row = renderRow({ refs, upstreams: { feature: 'origin/main' } })
    expect([...row.querySelectorAll('.ref-badge')].map((p) => p.textContent)).toEqual(['feature', 'origin/feature'])
  })
})
