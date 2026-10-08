import { render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { ChangedFile, CommitInfo, DiffService, FileDiff, HistoryService } from '@current-client-bindings/app'
import { DialogProvider } from '../../components/chrome/DialogProvider'
import CommitDetailPanel from './CommitDetailPanel'

const REPO = '/repos/app'
const commit = new CommitInfo({
  sha: 'abc1234def5678',
  parentShas: ['0001112223334'],
  authorName: 'Ada Lovelace',
  authorEmail: 'ada@example.com',
  date: '2026-10-01T10:00:00Z',
  subject: 'feat: a change with a long description',
  body: 'A paragraph.\n\n'.repeat(80),
})

function renderPanel() {
  vi.mocked(HistoryService.GetChangedFiles).mockResolvedValue([
    new ChangedFile({ status: 'M', path: 'src/app.ts', added: 3, removed: 1 }),
    new ChangedFile({ status: 'A', path: 'src/new.ts', added: 10 }),
  ])
  return render(
    <DialogProvider>
      <CommitDetailPanel
        repoPath={REPO}
        commit={commit}
        colorError={null}
        onColorChange={vi.fn()}
        onCreateBranchHere={vi.fn()}
      />
    </DialogProvider>,
  )
}

// Syntax highlighting splits a diff line into spans, so match the whole line.
const diffLine = (text: string) => (_: string, el: Element | null) =>
  el?.classList.contains('diff-content') === true && el.textContent === text

describe('CommitDetailPanel', () => {
  it('scrolls a long description instead of squeezing out the changed files', async () => {
    renderPanel()

    const fileList = (await screen.findByRole('button', { name: /src\/app\.ts/ })).closest('ul')!
    // With overflow set, a shrinkable flex item could be squashed to nothing.
    expect(getComputedStyle(fileList).flexShrink).toBe('0')
    expect(getComputedStyle(screen.getByText(/A paragraph\./)).flexShrink).toBe('0')
  })

  it("opens a file's diff in a large window, moves between the commit's files, and closes", async () => {
    vi.mocked(DiffService.GetRefDiff).mockImplementation(
      (_repo, path) =>
        Promise.resolve(
          new FileDiff({
            hunks: [
              {
                header: '@@ -1 +1 @@',
                raw: '',
                lines: [{ kind: 'added', oldLine: 0, newLine: 1, content: `in ${path}`, moved: false }],
              },
            ],
          }),
        ) as never,
    )
    renderPanel()

    await userEvent.click(await screen.findByRole('button', { name: /src\/new\.ts/ }))

    const dialog = await screen.findByRole('dialog', { name: 'Commit abc1234' })
    expect(within(dialog).getByText('feat: a change with a long description')).toBeInTheDocument()
    expect(await within(dialog).findByText(diffLine('in src/new.ts'))).toBeInTheDocument()
    expect(DiffService.GetRefDiff).toHaveBeenCalledWith(REPO, 'src/new.ts', '0001112223334', commit.sha, false, false)

    await userEvent.click(within(dialog).getByRole('button', { name: /src\/app\.ts/ }))
    expect(await within(dialog).findByText(diffLine('in src/app.ts'))).toBeInTheDocument()

    await userEvent.keyboard('{Escape}')
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
    // The narrow pane no longer shows a diff of its own.
    expect(screen.queryByText(diffLine('in src/app.ts'))).not.toBeInTheDocument()
  })

  it("says a merge's files are compared with its first parent", async () => {
    vi.mocked(HistoryService.GetChangedFiles).mockResolvedValue([new ChangedFile({ status: 'A', path: 'feature.ts' })])
    const merge = new CommitInfo({ ...commit, parentShas: ['aaaaaaa1111', 'bbbbbbb2222'], body: '' })
    render(
      <DialogProvider>
        <CommitDetailPanel
          repoPath={REPO}
          commit={merge}
          colorError={null}
          onColorChange={vi.fn()}
          onCreateBranchHere={vi.fn()}
        />
      </DialogProvider>,
    )

    expect(await screen.findByRole('button', { name: /feature\.ts/ })).toBeInTheDocument()
    expect(screen.getByText('Compared with the first parent, aaaaaaa.')).toBeInTheDocument()
  })
})
