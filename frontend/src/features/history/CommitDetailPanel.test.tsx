import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { ChangedFile, CommitInfo, HistoryService } from '@current-client-bindings/app'
import { Copy } from 'lucide-react'
import { DialogProvider } from '../../components/chrome/DialogProvider'
import CommitDetailPanel from './CommitDetailPanel'
import type { CommitAction } from './useCommitActions'

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

const ACTIONS: CommitAction[] = [
  'Check out',
  'Branch here',
  'Cherry-pick',
  'Revert',
  'Reset to here…',
  'Export patch…',
  'Copy SHA',
].map((label) => ({ id: label, label, icon: Copy, run: vi.fn() }))

function renderPanel() {
  vi.mocked(HistoryService.GetChangedFiles).mockResolvedValue([
    new ChangedFile({ status: 'M', path: 'src/app.ts', added: 3, removed: 1 }),
    new ChangedFile({ status: 'A', path: 'src/new.ts', added: 10 }),
  ])
  return render(
    <DialogProvider>
      <CommitDetailPanel repoPath={REPO} commit={commit} colorError={null} onColorChange={vi.fn()} actions={ACTIONS} />
    </DialogProvider>,
  )
}

describe('CommitDetailPanel', () => {
  it('shows the first four actions as buttons and the rest under More, in order', async () => {
    renderPanel()

    const footer = document.querySelector('.commit-detail-actions')!
    expect([...footer.querySelectorAll('button')].map((b) => b.textContent)).toEqual([
      'Check out',
      'Branch here',
      'Cherry-pick',
      'Revert',
      'More',
    ])
    await userEvent.click(screen.getByRole('button', { name: 'Cherry-pick' }))
    expect(ACTIONS[2].run).toHaveBeenCalled()

    await userEvent.click(screen.getByRole('button', { name: 'More' }))
    const menu = document.querySelector('.context-menu')!
    expect([...menu.querySelectorAll('button')].map((b) => b.textContent)).toEqual([
      'Reset to here…',
      'Export patch…',
      'Copy SHA',
    ])
    await userEvent.click(screen.getByRole('button', { name: 'Reset to here…' }))
    expect(ACTIONS[4].run).toHaveBeenCalled()
  })

  it('counts the changes and names the branches the commit is on', async () => {
    vi.mocked(HistoryService.BranchesContaining).mockResolvedValue(['feature', 'main'])
    renderPanel()

    expect(await screen.findByText('2 files', { selector: '.commit-detail-section-count' })).toBeInTheDocument()
    expect(screen.getByText('Changes').nextElementSibling).toHaveTextContent('2 files')
    expect((await screen.findByText('On branches')).nextElementSibling).toHaveTextContent('feature, main')
  })

  it('scrolls a long description instead of squeezing out the changed files', async () => {
    renderPanel()

    const fileList = (await screen.findByRole('button', { name: /src\/app\.ts/ })).closest('ul')!
    // With overflow set, a shrinkable flex item could be squashed to nothing.
    expect(getComputedStyle(fileList).flexShrink).toBe('0')
    expect(getComputedStyle(screen.getByText(/A paragraph\./)).flexShrink).toBe('0')
  })

  it('hands a clicked file to History and marks the one that is open', async () => {
    const onOpenFile = vi.fn()
    vi.mocked(HistoryService.GetChangedFiles).mockResolvedValue([
      new ChangedFile({ status: 'M', path: 'src/app.ts', added: 3, removed: 1 }),
      new ChangedFile({ status: 'A', path: 'src/new.ts', added: 10 }),
    ])
    render(
      <DialogProvider>
        <CommitDetailPanel
          repoPath={REPO}
          commit={commit}
          colorError={null}
          onColorChange={vi.fn()}
          actions={ACTIONS}
          openedPath="src/new.ts"
          onOpenFile={onOpenFile}
        />
      </DialogProvider>,
    )

    await userEvent.click(await screen.findByRole('button', { name: /src\/app\.ts/ }))
    expect(onOpenFile).toHaveBeenCalledWith('src/app.ts')
    expect(screen.getByRole('button', { name: /src\/new\.ts/ }).closest('.changed-file')).toHaveClass(
      'changed-file-selected',
    )
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
  })

  it("says a merge's files are compared with its first parent", async () => {
    vi.mocked(HistoryService.GetChangedFiles).mockResolvedValue([new ChangedFile({ status: 'A', path: 'feature.ts' })])
    const merge = new CommitInfo({ ...commit, parentShas: ['aaaaaaa1111', 'bbbbbbb2222'], body: '' })
    render(
      <DialogProvider>
        <CommitDetailPanel repoPath={REPO} commit={merge} colorError={null} onColorChange={vi.fn()} actions={ACTIONS} />
      </DialogProvider>,
    )

    expect(await screen.findByRole('button', { name: /feature\.ts/ })).toBeInTheDocument()
    expect(screen.getByText('Compared with the first parent, aaaaaaa.')).toBeInTheDocument()
  })
})
