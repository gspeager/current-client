import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import {
  ChangeDraft,
  CodeownersService,
  HostService,
  NoteInfo,
  ReadThroughInfo,
  ReadThroughNote,
  ReadThroughService,
  RemoteService,
} from '@current-client-bindings/app'
import { Change } from '@current-client-bindings/core/hostapi'
import { DialogProvider } from '../../components/chrome/DialogProvider'
import { NotesContext } from '../notes/notes'
import CreateChangeDialog from './CreateChangeDialog'

const draft = new ChangeDraft({
  branch: 'feature',
  remote: 'origin',
  hasUpstream: true,
  target: 'main',
  title: 'feat(login): add the login form',
  description: '- feat(login): add the login form\n\n## Checklist',
  kind: 'merge request',
})

function renderDialog(onCreated = vi.fn()) {
  render(
    <DialogProvider>
      <CreateChangeDialog
        repoPath="/repos/app"
        branch="feature"
        kind="merge request"
        onCreated={onCreated}
        onClose={vi.fn()}
      />
    </DialogProvider>,
  )
  return onCreated
}

describe('CreateChangeDialog', () => {
  it('suggests the code owners of the changed files as reviewers, for the typed target', async () => {
    vi.mocked(HostService.ChangeDraft).mockResolvedValue(draft)
    vi.mocked(CodeownersService.SuggestedReviewers).mockResolvedValue(['@alice', '@docs/team'])
    renderDialog()

    expect(await screen.findByText('@alice @docs/team')).toBeInTheDocument()
    expect(CodeownersService.SuggestedReviewers).toHaveBeenCalledWith('/repos/app', 'feature', 'main')
    expect(screen.getByText('Code owners of the changed files')).toBeInTheDocument()

    const target = screen.getByRole('textbox', { name: 'Target branch' })
    await userEvent.clear(target)
    await userEvent.type(target, 'develop')
    await vi.waitFor(() =>
      expect(CodeownersService.SuggestedReviewers).toHaveBeenLastCalledWith('/repos/app', 'feature', 'develop'),
    )
  })

  it('adds only the review notes chosen, and only when asked', async () => {
    vi.mocked(HostService.ChangeDraft).mockResolvedValue(draft)
    vi.mocked(CodeownersService.SuggestedReviewers).mockResolvedValue([])
    vi.mocked(ReadThroughService.Open).mockResolvedValue(new ReadThroughInfo({ key: 'refs/heads/feature' }))
    vi.mocked(ReadThroughService.Notes).mockResolvedValue([
      new ReadThroughNote({ id: 'n1', path: 'a.go', line: 4, text: 'why?' }),
      new ReadThroughNote({ id: 'n2', path: 'b.go', text: 'private' }),
    ])
    const listed = (target: string) => new NoteInfo({ kind: 'review', target, id: 'n1' })
    const notes = { notes: [listed('refs/heads/feature')], noteFor: () => undefined, edit: vi.fn(), openList: vi.fn() }
    render(
      <DialogProvider>
        <NotesContext.Provider value={notes}>
          <CreateChangeDialog
            repoPath="/repos/app"
            branch="feature"
            kind="merge request"
            onCreated={vi.fn()}
            onClose={vi.fn()}
          />
        </NotesContext.Provider>
      </DialogProvider>,
    )
    const add = await screen.findByRole('button', { name: 'Add to description' })
    const description = screen.getByRole('textbox', { name: 'Description' })
    expect(add).toBeDisabled()
    await userEvent.click(screen.getByRole('checkbox', { name: 'a.go, line 4: why?' }))
    expect(description).toHaveValue(draft.description)
    await userEvent.click(add)
    expect(description).toHaveValue(`${draft.description}\n\n- \`a.go:4\` why?`)
    expect(screen.queryByRole('checkbox', { name: 'a.go, line 4: why?' })).toBeNull()
    expect(screen.getByRole('checkbox', { name: 'b.go, whole file: private' })).not.toBeChecked()
  })

  it('has no notes section for a branch without any', async () => {
    vi.mocked(HostService.ChangeDraft).mockResolvedValue(draft)
    vi.mocked(CodeownersService.SuggestedReviewers).mockResolvedValue([])
    renderDialog()
    await screen.findByRole('textbox', { name: 'Title' })
    expect(screen.queryByRole('button', { name: 'Add to description' })).toBeNull()
    expect(ReadThroughService.Open).not.toHaveBeenCalled()
  })

  it('prefills from the draft and creates with what was entered', async () => {
    vi.mocked(HostService.ChangeDraft).mockResolvedValue(draft)
    vi.mocked(HostService.CreateChange).mockResolvedValue(new Change({ number: 9 }))
    const onCreated = renderDialog()

    const title = await screen.findByRole('textbox', { name: 'Title' })
    expect(title).toHaveValue('feat(login): add the login form')
    expect(screen.getByRole('textbox', { name: 'Description' })).toHaveValue(
      '- feat(login): add the login form\n\n## Checklist',
    )
    await userEvent.click(screen.getByRole('checkbox', { name: 'Draft' }))
    await userEvent.click(screen.getByRole('button', { name: 'Create merge request' }))

    expect(HostService.CreateChange).toHaveBeenCalledWith('/repos/app', 'feature', {
      target: 'main',
      title: 'feat(login): add the login form',
      description: '- feat(login): add the login form\n\n## Checklist',
      draft: true,
    })
    expect(RemoteService.PushSetUpstream).not.toHaveBeenCalled()
    expect(onCreated).toHaveBeenCalledWith(9)
  })

  it('pushes first for a branch without an upstream, and shows the scope message on a refusal', async () => {
    vi.mocked(HostService.ChangeDraft).mockResolvedValue(new ChangeDraft({ ...draft, hasUpstream: false }))
    vi.mocked(RemoteService.PushSetUpstream).mockResolvedValue({
      branch: 'feature',
      upstream: 'origin/feature',
      createLink: '',
      kind: 'merge request',
    })
    vi.mocked(HostService.CreateChange).mockRejectedValue(
      new Error(
        'The token for gitlab.example.com lacks the api scope this needs. Update it in Settings → Integrations.',
      ),
    )
    renderDialog()

    await userEvent.click(await screen.findByRole('button', { name: 'Push and create merge request' }))

    expect(RemoteService.PushSetUpstream).toHaveBeenCalledWith('/repos/app', 'origin', 'feature', null)
    expect(await screen.findByRole('alert')).toHaveTextContent('lacks the api scope')
  })
})
