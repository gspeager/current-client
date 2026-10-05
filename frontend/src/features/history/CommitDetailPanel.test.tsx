import { render, screen } from '@testing-library/react'
import { ChangedFile, CommitInfo, HistoryService } from '@current-client-bindings/app'
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

describe('CommitDetailPanel', () => {
  it('scrolls a long description instead of squeezing out the changed files', async () => {
    renderPanel()

    const fileList = (await screen.findByRole('button', { name: /src\/app\.ts/ })).closest('ul')!
    // With overflow set, a shrinkable flex item could be squashed to nothing.
    expect(getComputedStyle(fileList).flexShrink).toBe('0')
    expect(getComputedStyle(screen.getByText(/A paragraph\./)).flexShrink).toBe('0')
  })
})
