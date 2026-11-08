import { render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { TidyService, type TidyPlanInfo, type TidyPreviewInfo } from '@current-client-bindings/app'
import TidyUpDialog from './TidyUpDialog'

const commit = (subject: string, overrides = {}) => ({
  sha: `${subject}0000000`,
  subject,
  message: subject,
  suggested: '',
  ...overrides,
})

function plan(overrides: Partial<TidyPlanInfo> = {}): TidyPlanInfo {
  return {
    head: 'head',
    baseName: 'main',
    commits: [commit('aaa'), commit('bbb'), commit('ccc')],
    dirty: false,
    hasMerges: false,
    ...overrides,
  }
}

const preview = (overrides: Partial<TidyPreviewInfo> = {}): TidyPreviewInfo => ({
  result: [{ subject: 'aaa', folded: 0 }],
  outcomes: [],
  predicted: true,
  forcePush: false,
  upstream: '',
  ...overrides,
})

function renderDialog(info = plan(), previewInfo = preview()) {
  vi.mocked(TidyService.Plan).mockResolvedValue(info)
  vi.mocked(TidyService.Preview).mockResolvedValue(previewInfo)
  vi.mocked(TidyService.Run).mockResolvedValue()
  const onDone = vi.fn()
  render(<TidyUpDialog repoPath="/repo" onDone={onDone} onClose={vi.fn()} />)
  return onDone
}

const subjects = () =>
  screen
    .getAllByRole('listitem')
    .map((li) => li.querySelector('.tidy-subject')?.textContent)
    .filter(Boolean)
const action = (subject: string) => screen.getByRole('combobox', { name: new RegExp(`Action for .* ${subject}`) })

describe('TidyUpDialog', () => {
  it('reorders by its buttons and rewrites with the new order', async () => {
    const user = userEvent.setup()
    const onDone = renderDialog()
    await screen.findByText('3 commits since main, oldest first.')
    expect(screen.getByRole('button', { name: 'Rewrite' })).toBeDisabled()

    await user.click(screen.getByRole('button', { name: /Move .* aaa down/ }))
    expect(subjects()).toEqual(['bbb', 'aaa', 'ccc'])

    await user.click(await screen.findByRole('button', { name: 'Rewrite' }))
    expect(TidyService.Run).toHaveBeenCalledWith('/repo', 'head', [
      { sha: 'bbb0000000', action: 'keep', message: '' },
      { sha: 'aaa0000000', action: 'keep', message: '' },
      { sha: 'ccc0000000', action: 'keep', message: '' },
    ])
    await vi.waitFor(() => expect(onDone).toHaveBeenCalled())
  })

  it('moves a row from the keyboard through its handle', async () => {
    renderDialog()
    const handle = await screen.findByRole('button', { name: /^Move .* aaa$/ })
    expect(handle).toHaveAttribute('aria-roledescription', 'sortable')
  })

  it('edits a reworded message in place, and needs one', async () => {
    const user = userEvent.setup()
    renderDialog()
    await screen.findByText(/3 commits/)

    await user.selectOptions(action('bbb'), 'reword')
    const box = screen.getByRole('textbox', { name: /Message for .* bbb/ })
    await user.clear(box)
    expect(screen.getByText('A reworded commit needs a message.')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Rewrite' })).toBeDisabled()

    await user.type(box, 'better')
    await user.click(screen.getByRole('button', { name: 'Rewrite' }))
    expect(vi.mocked(TidyService.Run).mock.calls[0][2][1]).toEqual({
      sha: 'bbb0000000',
      action: 'reword',
      message: 'better',
    })
  })

  it("doesn't offer folding on the first commit", async () => {
    renderDialog()
    await screen.findByText(/3 commits/)
    expect(within(action('aaa')).queryByRole('option', { name: 'Squash' })).not.toBeInTheDocument()
    expect(within(action('bbb')).getByRole('option', { name: 'Squash' })).toBeInTheDocument()
  })

  it('refuses a fold with nothing to fold into, once moving makes it first', async () => {
    const user = userEvent.setup()
    renderDialog()
    await screen.findByText(/3 commits/)
    await user.selectOptions(action('bbb'), 'fixup')
    await user.click(screen.getByRole('button', { name: /Move .* bbb up/ }))
    expect(screen.getByText('The first commit kept has nothing to fold into.')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Rewrite' })).toBeDisabled()
  })

  it('refuses dropping everything', async () => {
    const user = userEvent.setup()
    renderDialog()
    await screen.findByText(/3 commits/)
    for (const subject of ['aaa', 'bbb', 'ccc']) await user.selectOptions(action(subject), 'drop')
    expect(screen.getByText('At least one commit has to stay.')).toBeInTheDocument()
  })

  it('previews the result, a predicted conflict and a force push', async () => {
    const user = userEvent.setup()
    renderDialog(
      plan(),
      preview({
        result: [{ subject: 'aaa', folded: 1 }],
        outcomes: [{ sha: 'bbb0000000', outcome: 'conflict', files: ['f.txt'] }],
        forcePush: true,
        upstream: 'origin/topic',
      }),
    )
    await screen.findByText(/3 commits/)
    await user.selectOptions(action('ccc'), 'squash')

    expect(await screen.findByText('Makes 1 commit:')).toBeInTheDocument()
    expect(screen.getByText('(+1 folded in)')).toBeInTheDocument()
    expect(screen.getByText('Conflict')).toBeInTheDocument()
    expect(screen.getByText('f.txt')).toBeInTheDocument()
    expect(screen.getByText(/--force-with-lease/)).toBeInTheDocument()
  })

  it('starts a fixup! commit set to Fixup, and enables Rewrite', async () => {
    renderDialog(plan({ commits: [commit('aaa'), commit('fixup! aaa', { suggested: 'fixup' })] }))
    await screen.findByText(/2 commits/)
    expect(action('fixup! aaa')).toHaveValue('fixup')
    expect(screen.getByRole('button', { name: 'Rewrite' })).toBeEnabled()
  })

  it('blocks a merge in the range and changes in the working tree', async () => {
    renderDialog(plan({ hasMerges: true, dirty: true }))
    expect(await screen.findByText(/include a merge/)).toBeInTheDocument()
    expect(screen.getByText(/Commit or stash/)).toBeInTheDocument()
  })

  it('shows why a rewrite failed', async () => {
    const user = userEvent.setup()
    renderDialog()
    vi.mocked(TidyService.Run).mockRejectedValue(new Error('The branch has moved.'))
    await screen.findByText(/3 commits/)
    await user.selectOptions(action('aaa'), 'drop')
    await user.click(screen.getByRole('button', { name: 'Rewrite' }))
    expect(await screen.findByText('The branch has moved.')).toBeInTheDocument()
  })
})
