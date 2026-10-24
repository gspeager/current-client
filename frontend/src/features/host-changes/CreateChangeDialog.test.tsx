import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { ChangeDraft, HostService, RemoteService } from '@current-client-bindings/app'
import { Change } from '@current-client-bindings/core/hostapi'
import { DialogProvider } from '../../components/chrome/DialogProvider'
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
