import { render, screen } from '@testing-library/react'
import { FileDiff, SubmoduleService } from '@current-client-bindings/app'
import DiffViewer from './DiffViewer'
import { submoduleChange } from './submoduleChange'

const A = 'aaaaaaa1111111111111111111111111111111111'
const B = 'bbbbbbb2222222222222222222222222222222222'

function diffOf(lines: { kind: string; content: string }[]) {
  return new FileDiff({
    hunks: [
      { header: '@@ -1 +1 @@', raw: '', lines: lines.map((l) => ({ ...l, oldLine: 1, newLine: 1, moved: false })) },
    ],
  })
}

describe('submoduleChange', () => {
  it('reads the two commits and whether the new side is dirty', () => {
    const moved = diffOf([
      { kind: 'removed', content: `Subproject commit ${A}` },
      { kind: 'added', content: `Subproject commit ${B}-dirty` },
    ])
    expect(submoduleChange(moved)).toEqual({ from: A, to: B, dirty: true })
    expect(submoduleChange(diffOf([{ kind: 'added', content: `Subproject commit ${B}` }]))).toEqual({
      from: null,
      to: B,
      dirty: false,
    })
  })

  it('ignores ordinary diffs', () => {
    expect(submoduleChange(diffOf([{ kind: 'added', content: 'Subproject commit is a phrase' }]))).toBeNull()
    expect(submoduleChange(new FileDiff({ hunks: [] }))).toBeNull()
  })
})

describe('DiffViewer for a submodule', () => {
  it('shows the move and the commits between the two versions', async () => {
    vi.mocked(SubmoduleService.Commits).mockResolvedValue([
      { sha: 'ccc3333', subject: 'lib: new thing', added: true },
      { sha: 'ddd4444', subject: 'lib: dropped', added: false },
    ])
    const diff = diffOf([
      { kind: 'removed', content: `Subproject commit ${A}` },
      { kind: 'added', content: `Subproject commit ${B}` },
    ])
    render(<DiffViewer diff={diff} path="vendor/lib" viewMode="split" repoPath="/repo" />)

    expect(screen.getByText('aaaaaaa → bbbbbbb')).toBeInTheDocument()
    expect(await screen.findByText('lib: new thing')).toBeInTheDocument()
    expect(screen.getByText('lib: dropped')).toBeInTheDocument()
    expect(SubmoduleService.Commits).toHaveBeenCalledWith('/repo', 'vendor/lib', A, B)
  })

  it('says to update when the commits are unavailable, and notes local changes', async () => {
    vi.mocked(SubmoduleService.Commits).mockRejectedValue(new Error('bad revision'))
    const diff = diffOf([
      { kind: 'removed', content: `Subproject commit ${A}` },
      { kind: 'added', content: `Subproject commit ${B}-dirty` },
    ])
    render(<DiffViewer diff={diff} path="vendor/lib" viewMode="split" repoPath="/repo" />)

    expect(screen.getByText('Has uncommitted changes inside it.')).toBeInTheDocument()
    expect(
      await screen.findByText('Update the submodule to see the commits between these versions.'),
    ).toBeInTheDocument()
  })
})
