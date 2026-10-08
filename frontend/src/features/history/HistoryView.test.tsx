import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { HistoryService, type CommitInfo } from '@current-client-bindings/app'
import { DialogProvider } from '../../components/chrome/DialogProvider'
import { giveElementsLayout } from '../../test/bindings'
import HistoryView from './HistoryView'

const REPO = '/repos/app'

function commit(sha: string, subject: string, parentShas: string[]): CommitInfo {
  return {
    sha,
    parentShas,
    authorName: 'Ada Lovelace',
    authorEmail: 'ada@example.com',
    date: '2026-01-10T12:00:00Z',
    subject,
    body: '',
    signature: 'none',
    coAuthors: [],
  }
}

const history = [
  commit('b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0c1', 'Fix login redirect', [
    'a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0',
  ]),
  commit('a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0', 'Initial commit', []),
]

function renderHistory(props: Partial<Parameters<typeof HistoryView>[0]> = {}) {
  render(
    <DialogProvider>
      <HistoryView repoPath={REPO} {...props} />
    </DialogProvider>,
  )
}

beforeEach(() => giveElementsLayout())

describe('HistoryView', () => {
  it('opens the inspector for a clicked commit and reports the selection', async () => {
    vi.mocked(HistoryService.GetHistory).mockResolvedValue(history)
    const onSelectedShaChange = vi.fn()
    renderHistory({ onSelectedShaChange })

    await userEvent.click(await screen.findByRole('button', { name: /Initial commit/ }))

    expect(onSelectedShaChange).toHaveBeenCalledWith(history[1].sha)
    expect(HistoryService.GetChangedFiles).toHaveBeenCalledWith(REPO, history[1].sha)
    expect(screen.getByRole('button', { name: 'Copy full commit hash' }).parentElement).toHaveTextContent('a1b2c3d')
  })

  it('lists co-authors in the inspector', async () => {
    const paired = {
      ...history[1],
      coAuthors: [
        { name: 'Grace Hopper', email: 'grace@example.com' },
        { name: 'Alan Turing', email: '' },
      ],
    }
    vi.mocked(HistoryService.GetHistory).mockResolvedValue([history[0], paired])
    renderHistory({ initialSelectedSha: paired.sha })

    const label = await screen.findByText('Co-authors')
    expect(label.parentElement).toHaveTextContent('Grace Hopper, Alan Turing')
    expect(screen.getByText('Grace Hopper')).toHaveAttribute('title', 'grace@example.com')
  })

  it('shows no co-authors row for a commit without them', async () => {
    vi.mocked(HistoryService.GetHistory).mockResolvedValue(history)
    renderHistory({ initialSelectedSha: history[0].sha })

    await screen.findByRole('button', { name: 'Copy full commit hash' })
    expect(screen.queryByText('Co-authors')).not.toBeInTheDocument()
  })

  it('starts with the requested commit selected', async () => {
    vi.mocked(HistoryService.GetHistory).mockResolvedValue(history)
    renderHistory({ initialSelectedSha: history[0].sha })

    await screen.findByRole('button', { name: /Fix login redirect/ })
    expect(HistoryService.GetChangedFiles).toHaveBeenCalledWith(REPO, history[0].sha)
  })

  it('opens a commit menu with Shift+F10 on the focused row', async () => {
    vi.mocked(HistoryService.GetHistory).mockResolvedValue(history)
    renderHistory()
    const user = userEvent.setup()

    ;(await screen.findByRole('button', { name: /Initial commit/ })).focus()
    await user.keyboard('{Shift>}{F10}{/Shift}')

    expect(await screen.findByRole('button', { name: 'Copy SHA' })).toHaveFocus()
    expect(screen.getByRole('button', { name: 'Reset (hard) to here' })).toBeInTheDocument()
  })

  it('shows why history could not load', async () => {
    vi.mocked(HistoryService.GetHistory).mockRejectedValue(new Error('Not a git repository.'))
    renderHistory()

    expect(await screen.findByText('Could not load history: Not a git repository.')).toBeInTheDocument()
  })
  describe('Conventional Commits', () => {
    const typed = [commit('c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0c1d2', 'feat(history): filter by type', [])]

    it('mutes the type prefix and filters by type', async () => {
      vi.mocked(HistoryService.GetHistory).mockResolvedValue(typed)
      renderHistory({ conventional: true })

      const prefix = await screen.findByText('feat(history):', { exact: false, selector: '.commit-row-type' })
      expect(prefix.parentElement).toHaveTextContent('feat(history): filter by type')

      await userEvent.selectOptions(screen.getByRole('combobox', { name: 'Type filter' }), 'Breaking changes')
      expect(HistoryService.GetHistory).toHaveBeenLastCalledWith(
        REPO,
        expect.any(Number),
        0,
        expect.objectContaining({ type: '!' }),
      )
    })

    it('shows plain subjects and no type filter without the format', async () => {
      vi.mocked(HistoryService.GetHistory).mockResolvedValue(typed)
      renderHistory()

      await screen.findByText('feat(history): filter by type')
      expect(document.querySelector('.commit-row-type')).toBeNull()
      expect(screen.queryByRole('combobox', { name: 'Type filter' })).toBeNull()
    })
  })
})
