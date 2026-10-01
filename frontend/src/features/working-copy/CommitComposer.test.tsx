import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { IdentityService, type CurrentUserInfo } from '@current-client-bindings/app'
import { DialogProvider } from '../../components/chrome/DialogProvider'
import CommitComposer from './CommitComposer'

const REPO = '/repos/app'

function user(name: string, email: string): CurrentUserInfo {
  return { name, email, initials: '', color: '' }
}

function renderComposer(showTypePicker = false) {
  render(
    <DialogProvider>
      <CommitComposer repoPath={REPO} stagedCount={1} onCommitted={vi.fn()} showTypePicker={showTypePicker} />
    </DialogProvider>,
  )
}

describe('CommitComposer identity', () => {
  it('asks for a name and email before the first commit, then allows it', async () => {
    vi.mocked(IdentityService.GetCurrentUser)
      .mockResolvedValueOnce(user('', ''))
      .mockResolvedValue(user('Ada Lovelace', 'ada@example.com'))
    vi.mocked(IdentityService.SetGlobalUser).mockResolvedValue()
    renderComposer()
    await userEvent.type(screen.getByPlaceholderText('Summary'), 'First commit')

    expect(await screen.findByText('Commits need a name and email.')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /Commit 1 file/ })).toBeDisabled()

    await userEvent.type(screen.getByRole('textbox', { name: 'Name' }), ' Ada Lovelace ')
    await userEvent.type(screen.getByRole('textbox', { name: 'Email' }), 'ada@example.com')
    await userEvent.click(screen.getByRole('button', { name: 'Save identity' }))

    expect(IdentityService.SetGlobalUser).toHaveBeenCalledWith('Ada Lovelace', 'ada@example.com')
    await vi.waitFor(() => expect(screen.queryByText('Commits need a name and email.')).not.toBeInTheDocument())
    expect(screen.getByRole('button', { name: /Commit 1 file/ })).toBeEnabled()
  })

  it('can save the identity for this repository only', async () => {
    vi.mocked(IdentityService.GetCurrentUser).mockResolvedValue(user('Ada Lovelace', ''))
    vi.mocked(IdentityService.SetRepoUser).mockResolvedValue()
    renderComposer()

    await userEvent.type(await screen.findByRole('textbox', { name: 'Email' }), 'ada@work.example')
    await userEvent.click(screen.getByRole('checkbox', { name: 'This repository only' }))
    await userEvent.click(screen.getByRole('button', { name: 'Save identity' }))

    expect(IdentityService.SetRepoUser).toHaveBeenCalledWith(REPO, 'Ada Lovelace', 'ada@work.example')
    expect(IdentityService.SetGlobalUser).not.toHaveBeenCalled()
  })

  it('shows no prompt when an identity is configured', async () => {
    vi.mocked(IdentityService.GetCurrentUser).mockResolvedValue(user('Ada Lovelace', 'ada@example.com'))
    renderComposer()

    await vi.waitFor(() => expect(IdentityService.GetCurrentUser).toHaveBeenCalled())
    expect(screen.queryByText('Commits need a name and email.')).not.toBeInTheDocument()
  })
})

describe('CommitComposer type picker', () => {
  beforeEach(() => {
    vi.mocked(IdentityService.GetCurrentUser).mockResolvedValue(user('Ada Lovelace', 'ada@example.com'))
  })

  const subject = () => screen.getByPlaceholderText('Summary')
  const type = () => screen.getByRole('combobox', { name: 'Commit type' })

  it('writes, replaces and removes the prefix while the subject stays editable', async () => {
    renderComposer(true)
    await userEvent.type(subject(), 'filter by type')

    await userEvent.selectOptions(type(), 'feat')
    expect(subject()).toHaveValue('feat: filter by type')

    await userEvent.type(screen.getByRole('textbox', { name: 'Scope' }), 'history')
    expect(subject()).toHaveValue('feat(history): filter by type')

    await userEvent.selectOptions(type(), 'fix')
    expect(subject()).toHaveValue('fix(history): filter by type')

    await userEvent.type(subject(), ' again')
    expect(subject()).toHaveValue('fix(history): filter by type again')

    await userEvent.selectOptions(type(), 'No type')
    expect(subject()).toHaveValue('filter by type again')
    expect(screen.getByRole('textbox', { name: 'Scope' })).toBeDisabled()
  })

  it('follows a prefix typed by hand, keeping its breaking mark', async () => {
    renderComposer(true)
    await userEvent.type(subject(), 'feat!: drop v0')

    expect(type()).toHaveValue('feat')
    await userEvent.selectOptions(type(), 'refactor')
    expect(subject()).toHaveValue('refactor!: drop v0')
  })

  it('is hidden unless requested', () => {
    renderComposer()

    expect(screen.queryByRole('combobox', { name: 'Commit type' })).toBeNull()
  })
})
