import { render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { RemoteService, TagInfo, TagService } from '@current-client-bindings/app'
import { DialogProvider } from '../../components/chrome/DialogProvider'
import TagsPanel from './TagsPanel'

const REPO = '/repos/app'

function renderPanel() {
  vi.mocked(TagService.ListTags).mockResolvedValue([
    new TagInfo({ name: 'v1.0', sha: 'abc123', date: '2026-10-01T10:00:00Z' }),
  ])
  vi.mocked(RemoteService.List).mockResolvedValue([
    { name: 'upstream', fetchUrl: '', pushUrl: '' },
    { name: 'origin', fetchUrl: '', pushUrl: '' },
  ])
  render(
    <DialogProvider>
      <TagsPanel repoPath={REPO} />
    </DialogProvider>,
  )
}

describe('TagsPanel', () => {
  it('deletes a tag on origin after confirming, and keeps the local tag', async () => {
    vi.mocked(TagService.DeleteRemoteTag).mockResolvedValue()
    renderPanel()

    await userEvent.click(await screen.findByRole('button', { name: 'Delete tag v1.0 on remote' }))
    const dialog = await screen.findByRole('alertdialog', { name: 'Delete tag on remote' })
    expect(dialog).toHaveTextContent('Delete tag "v1.0" on origin?')
    await userEvent.click(within(dialog).getByRole('button', { name: 'Delete' }))

    await vi.waitFor(() => expect(TagService.DeleteRemoteTag).toHaveBeenCalledWith(REPO, 'origin', 'v1.0', null))
    expect(TagService.DeleteTag).not.toHaveBeenCalled()
  })

  it('does nothing when the confirmation is cancelled', async () => {
    renderPanel()

    await userEvent.click(await screen.findByRole('button', { name: 'Delete tag v1.0 on remote' }))
    const dialog = await screen.findByRole('alertdialog', { name: 'Delete tag on remote' })
    await userEvent.click(within(dialog).getByRole('button', { name: 'Cancel' }))

    expect(TagService.DeleteRemoteTag).not.toHaveBeenCalled()
  })

  it('explains when there is no remote', async () => {
    renderPanel()
    vi.mocked(RemoteService.List).mockResolvedValue([])

    await userEvent.click(await screen.findByRole('button', { name: 'Delete tag v1.0 on remote' }))

    expect(await screen.findByText('No remote configured to push to.')).toBeInTheDocument()
    expect(screen.queryByRole('alertdialog')).not.toBeInTheDocument()
  })
})
