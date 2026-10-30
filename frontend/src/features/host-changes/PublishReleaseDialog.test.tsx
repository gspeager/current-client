import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { HostService, TagService } from '@current-client-bindings/app'
import { DialogProvider } from '../../components/chrome/DialogProvider'
import PublishReleaseDialog from './PublishReleaseDialog'

const draft = {
  tag: 'v0.3.0',
  name: 'v0.3.0',
  notes: '### Added\n\n- Issues.',
  notesFrom: 'changelog',
  since: '',
  remote: 'origin',
}

function renderDialog() {
  render(
    <DialogProvider>
      <PublishReleaseDialog repoPath="/repos/app" tag="v0.3.0" onClose={vi.fn()} />
    </DialogProvider>,
  )
}

describe('PublishReleaseDialog', () => {
  it('prefills from the changelog, pushes the tag, then publishes on click', async () => {
    vi.mocked(HostService.ReleaseDraft).mockResolvedValue(draft)
    vi.mocked(TagService.PushTag).mockResolvedValue()
    vi.mocked(HostService.PublishRelease).mockResolvedValue({ tag: 'v0.3.0', name: '0.3.0', webUrl: 'https://x' })
    renderDialog()

    const notes = await screen.findByRole('textbox', { name: /Notes · From the changelog at v0\.3\.0/ })
    expect(notes).toHaveValue('### Added\n\n- Issues.')
    expect(HostService.PublishRelease).not.toHaveBeenCalled()
    const name = screen.getByRole('textbox', { name: 'Name' })
    await userEvent.clear(name)
    await userEvent.type(name, '0.3.0')
    await userEvent.click(screen.getByRole('button', { name: 'Publish release' }))

    expect(TagService.PushTag).toHaveBeenCalledWith('/repos/app', 'origin', 'v0.3.0', null)
    expect(HostService.PublishRelease).toHaveBeenCalledWith('/repos/app', {
      tag: 'v0.3.0',
      name: '0.3.0',
      notes: '### Added\n\n- Issues.',
    })
    expect(await screen.findByRole('status')).toHaveTextContent('Published 0.3.0.')
  })

  it('names the commits the notes came from, and shows a refusal without publishing', async () => {
    vi.mocked(HostService.ReleaseDraft).mockResolvedValue({ ...draft, notesFrom: 'commits', since: 'v0.2.0' })
    vi.mocked(TagService.PushTag).mockRejectedValue(new Error('the remote has a different v0.3.0'))
    renderDialog()
    await screen.findByRole('textbox', { name: /From the commits since v0\.2\.0/ })
    await userEvent.click(screen.getByRole('button', { name: 'Publish release' }))
    expect(await screen.findByRole('alert')).toHaveTextContent('the remote has a different v0.3.0')
    expect(HostService.PublishRelease).not.toHaveBeenCalled()
  })
})
