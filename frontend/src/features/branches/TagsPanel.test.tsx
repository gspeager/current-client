import { render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { HostService, RemoteService, TagInfo, TagService } from '@current-client-bindings/app'
import { DialogProvider } from '../../components/chrome/DialogProvider'
import TagsPanel from './TagsPanel'
import { defaultSections } from '../../test/sections'

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
      <TagsPanel sections={defaultSections()} repoPath={REPO} />
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

  it("holds the row's actions in its context menu, Copy name included", async () => {
    vi.mocked(TagService.PushTag).mockResolvedValue()
    renderPanel()

    await userEvent.pointer({ keys: '[MouseRight]', target: await screen.findByTitle('v1.0') })
    const menu = document.querySelector<HTMLElement>('.context-menu')!
    expect([...menu.querySelectorAll('button')].map((i) => i.textContent)).toEqual([
      'Push tag',
      'Copy name',
      'Delete on remote…',
      'Delete…',
    ])
    await userEvent.click(within(menu).getByRole('button', { name: 'Push tag' }))
    await vi.waitFor(() => expect(TagService.PushTag).toHaveBeenCalledWith(REPO, 'origin', 'v1.0', null))
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

    expect(await screen.findByText('No remote configured.')).toBeInTheDocument()
    expect(screen.queryByRole('alertdialog')).not.toBeInTheDocument()
  })

  it('offers Publish release on host only where the host can publish one', async () => {
    vi.mocked(HostService.CanPublishReleases).mockResolvedValue(true)
    vi.mocked(HostService.ReleaseDraft).mockResolvedValue({
      tag: 'v1.0',
      name: 'v1.0',
      notes: '- First.',
      notesFrom: 'commits',
      since: '',
      remote: 'origin',
    })
    renderPanel()
    await userEvent.click(await screen.findByRole('button', { name: 'Publish release for v1.0 on host' }))
    expect(await screen.findByRole('dialog', { name: 'Publish release' })).toHaveTextContent(
      'From every commit up to the tag',
    )
    expect(HostService.PublishRelease).not.toHaveBeenCalled()
  })

  it('leaves Publish release out with the host off', async () => {
    vi.mocked(HostService.CanPublishReleases).mockResolvedValue(false)
    renderPanel()
    await screen.findByRole('button', { name: 'Push tag v1.0' })
    expect(screen.queryByRole('button', { name: 'Publish release for v1.0 on host' })).not.toBeInTheDocument()
  })
})
