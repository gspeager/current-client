import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import {
  CommitService,
  ConflictService,
  ConflictStateInfo,
  IdentityService,
  type CurrentUserInfo,
} from '@current-client-bindings/app'
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
    expect(screen.getByRole('button', { name: 'Set a name and email to commit' })).toBeDisabled()

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
    expect(screen.queryByRole('textbox', { name: 'Scope' })).not.toBeInTheDocument()
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

describe('CommitComposer after a squash merge', () => {
  it('starts the description with the squashed commits and leaves the summary to write', async () => {
    vi.mocked(IdentityService.GetCurrentUser).mockResolvedValue(user('Ada Lovelace', 'ada@example.com'))
    vi.mocked(CommitService.GetSquashedSubjects).mockResolvedValue(['fix: second thing', 'feat: first thing'])
    const onMessageChange = vi.fn()
    render(
      <DialogProvider>
        <CommitComposer repoPath={REPO} stagedCount={2} onCommitted={vi.fn()} onMessageChange={onMessageChange} />
      </DialogProvider>,
    )

    await vi.waitFor(() =>
      expect(screen.getByPlaceholderText('Description')).toHaveValue('- fix: second thing\n- feat: first thing'),
    )
    expect(screen.getByPlaceholderText('Summary')).toHaveValue('')
    expect(onMessageChange).toHaveBeenCalledWith('\n\n- fix: second thing\n- feat: first thing')
  })

  it('keeps a draft that was already there', async () => {
    vi.mocked(IdentityService.GetCurrentUser).mockResolvedValue(user('Ada Lovelace', 'ada@example.com'))
    render(
      <DialogProvider>
        <CommitComposer repoPath={REPO} stagedCount={2} onCommitted={vi.fn()} initialMessage="feat: my own words" />
      </DialogProvider>,
    )

    expect(await screen.findByDisplayValue('feat: my own words')).toBeInTheDocument()
    expect(CommitService.GetSquashedSubjects).not.toHaveBeenCalled()
  })
})

describe('CommitComposer during a merge', () => {
  const merge = (conflictedPaths: string[]) =>
    new ConflictStateInfo({
      operation: 'merge',
      conflictedPaths,
      ours: 'main',
      theirs: 'feature',
      message: "Merge branch 'feature'\n\nBody line.",
    })

  function renderMerging(conflict: ConflictStateInfo, onCommitted = vi.fn()) {
    vi.mocked(IdentityService.GetCurrentUser).mockResolvedValue(user('Ada', 'ada@example.com'))
    render(
      <DialogProvider>
        <CommitComposer repoPath={REPO} stagedCount={1} onCommitted={onCommitted} showTypePicker conflict={conflict} />
      </DialogProvider>,
    )
    return onCommitted
  }

  it('is the merge commit: prefilled from git, Continue merge disabled while files are conflicted', async () => {
    renderMerging(merge(['src/app.ts']))

    expect(screen.getByText('Merge commit')).toBeInTheDocument()
    await vi.waitFor(() => expect(screen.getByPlaceholderText('Summary')).toHaveValue("Merge branch 'feature'"))
    expect(screen.getByPlaceholderText('Description')).toHaveValue('Body line.')
    const button = screen.getByRole('button', { name: '1 file still conflicted' })
    expect(button).toBeDisabled()
    expect(screen.queryByText('1 file still conflicted.')).not.toBeInTheDocument()
    expect(screen.queryByRole('combobox', { name: 'Commit type' })).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'More commit actions' })).not.toBeInTheDocument()
  })

  it('continues the merge with the message once everything is resolved', async () => {
    vi.mocked(ConflictService.ContinueWith).mockResolvedValue()
    const onCommitted = renderMerging(merge([]))

    await vi.waitFor(() => expect(screen.getByPlaceholderText('Summary')).toHaveValue("Merge branch 'feature'"))
    await userEvent.clear(screen.getByPlaceholderText('Summary'))
    await userEvent.type(screen.getByPlaceholderText('Summary'), 'Bring feature in')
    await userEvent.click(screen.getByRole('button', { name: 'Continue merge' }))

    await vi.waitFor(() =>
      expect(ConflictService.ContinueWith).toHaveBeenCalledWith(REPO, 'Bring feature in\n\nBody line.'),
    )
    await vi.waitFor(() => expect(onCommitted).toHaveBeenCalled())
  })
})

describe('CommitComposer button', () => {
  it('says what is missing instead of a helper line, and hides the scope until a type is set', async () => {
    vi.mocked(IdentityService.GetCurrentUser).mockResolvedValue(user('Ada', 'ada@example.com'))
    render(
      <DialogProvider>
        <CommitComposer repoPath={REPO} stagedCount={0} onCommitted={vi.fn()} showTypePicker />
      </DialogProvider>,
    )

    await vi.waitFor(() => expect(screen.getByRole('button', { name: 'Enter a commit message' })).toBeDisabled())
    expect(screen.queryByText('Enter a commit message.')).not.toBeInTheDocument()
    expect(screen.queryByRole('textbox', { name: 'Scope' })).not.toBeInTheDocument()

    await userEvent.type(screen.getByPlaceholderText('Summary'), 'Add thing')
    expect(screen.getByRole('button', { name: 'Nothing is staged' })).toBeDisabled()

    await userEvent.selectOptions(screen.getByRole('combobox', { name: 'Commit type' }), 'feat')
    expect(screen.getByRole('textbox', { name: 'Scope' })).toBeInTheDocument()
  })
})
