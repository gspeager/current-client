import { render, screen } from '@testing-library/react'
import { FileDiff, GitService } from '@current-client-bindings/app'
import DiffViewer from './DiffViewer'
import { lfsPointerChange } from './lfsPointer'

const VERSION = 'version https://git-lfs.github.com/spec/v1'

function diffOf(lines: [string, string][]) {
  return new FileDiff({
    hunks: [
      {
        header: '@@ -1,3 +1,3 @@',
        raw: '',
        lines: lines.map(([kind, content]) => ({ kind, content, oldLine: 1, newLine: 1, moved: false })),
      },
    ],
  })
}

const changed = diffOf([
  ['context', VERSION],
  ['removed', 'oid sha256:aaaa'],
  ['removed', 'size 1536'],
  ['added', 'oid sha256:bbbb'],
  ['added', 'size 3145728'],
])

describe('lfsPointerChange', () => {
  it('reads the old and new sizes from a pointer diff', () => {
    expect(lfsPointerChange(changed)).toEqual({ oldSize: 1536, newSize: 3145728 })
    expect(
      lfsPointerChange(
        diffOf([
          ['added', VERSION],
          ['added', 'oid sha256:bbbb'],
          ['added', 'size 10'],
        ]),
      ),
    ).toEqual({ oldSize: null, newSize: 10 })
  })

  it('leaves ordinary text alone, even text that mentions LFS', () => {
    expect(
      lfsPointerChange(
        diffOf([
          ['added', VERSION],
          ['added', 'and some prose'],
        ]),
      ),
    ).toBeNull()
    expect(lfsPointerChange(diffOf([['added', 'size 10']]))).toBeNull()
  })
})

describe('DiffViewer for a file stored in Git LFS', () => {
  it('shows the size change instead of the pointer text', async () => {
    vi.mocked(GitService.LFSInstalled).mockResolvedValue(true)
    render(<DiffViewer diff={changed} path="art/cover.psd" viewMode="split" />)

    expect(screen.getByText('1.5 KB → 3.0 MB')).toBeInTheDocument()
    expect(screen.queryByText('oid sha256:aaaa')).not.toBeInTheDocument()
    await vi.waitFor(() => expect(GitService.LFSInstalled).toHaveBeenCalled())
    expect(screen.queryByText(/isn't installed/)).not.toBeInTheDocument()
  })

  it("says when Git LFS isn't installed", async () => {
    vi.mocked(GitService.LFSInstalled).mockResolvedValue(false)
    render(<DiffViewer diff={changed} path="art/cover.psd" viewMode="split" />)

    expect(await screen.findByText(/Git LFS isn't installed/)).toBeInTheDocument()
  })
})
