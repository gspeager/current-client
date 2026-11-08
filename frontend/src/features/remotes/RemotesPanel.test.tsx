import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { RemoteService } from '@current-client-bindings/app'
import { DialogProvider } from '../../components/chrome/DialogProvider'
import RemotesPanel from './RemotesPanel'
import { defaultSections } from '../../test/sections'

const REPO = '/repos/app'

function renderPanel(pruneOnFetch: boolean) {
  vi.mocked(RemoteService.List).mockResolvedValue([
    { name: 'origin', fetchUrl: 'https://example.com/app.git', pushUrl: '' },
    { name: 'upstream', fetchUrl: 'https://example.com/upstream.git', pushUrl: '' },
  ])
  const onFetched = vi.fn()
  render(
    <DialogProvider>
      <RemotesPanel sections={defaultSections()} repoPath={REPO} onFetched={onFetched} pruneOnFetch={pruneOnFetch} />
    </DialogProvider>,
  )
  return onFetched
}

describe('RemotesPanel', () => {
  it('fetches all remotes without pruning when Prune is off', async () => {
    vi.mocked(RemoteService.FetchAll).mockResolvedValue()
    const onFetched = renderPanel(false)

    await userEvent.click(await screen.findByRole('button', { name: /fetch all/i }))

    await vi.waitFor(() => expect(onFetched).toHaveBeenCalled())
    expect(RemoteService.FetchAll).toHaveBeenCalledWith(REPO, null)
    expect(RemoteService.FetchAllPrune).not.toHaveBeenCalled()
  })

  it('prunes when fetching all remotes with Prune on', async () => {
    vi.mocked(RemoteService.FetchAllPrune).mockResolvedValue()
    const onFetched = renderPanel(true)

    await userEvent.click(await screen.findByRole('button', { name: /fetch all/i }))

    await vi.waitFor(() => expect(onFetched).toHaveBeenCalled())
    expect(RemoteService.FetchAllPrune).toHaveBeenCalledWith(REPO, null)
    expect(RemoteService.FetchAll).not.toHaveBeenCalled()
  })

  describe('with a sign-in in the URL', () => {
    const shownUrl = 'https://<token>@git.example.com/a.git'

    function renderWithUrl(fetchUrl: string) {
      vi.mocked(RemoteService.List).mockResolvedValue([{ name: 'origin', fetchUrl, pushUrl: fetchUrl }])
      render(
        <DialogProvider>
          <RemotesPanel sections={defaultSections()} repoPath={REPO} />
        </DialogProvider>,
      )
    }

    it.each([
      [shownUrl, 'git.example.com'],
      ['ssh://git:<token>@git.example.com/a.git', 'git.example.com'],
      ['ssh://git@git.example.com:2222/a.git', 'git.example.com:2222'],
      ['https://www.example.com/a.git', 'example.com'],
      ['git@git.example.com:a.git', 'git.example.com'],
    ])('labels %s as %s', async (fetchUrl, host) => {
      renderWithUrl(fetchUrl)

      const label = await screen.findByText(host)
      expect(label).toHaveClass('remote-row-host')
      expect(label.closest('li')).toHaveAttribute('title', fetchUrl)
    })

    it('has nothing to save until the URL changes', async () => {
      renderWithUrl(shownUrl)

      await userEvent.click(await screen.findByRole('button', { name: 'Edit origin' }))

      expect(screen.getByRole('textbox')).toHaveValue(shownUrl)
      expect(screen.getByRole('button', { name: 'Save' })).toBeDisabled()
      expect(RemoteService.EditRemote).not.toHaveBeenCalled()
    })

    it('shows why an edit that keeps the placeholder was refused', async () => {
      const refusal = new Error("Replace <token> with the sign-in, or remove it to use Git's credential helper.")
      vi.mocked(RemoteService.EditRemote).mockRejectedValue(refusal)
      renderWithUrl(shownUrl)

      await userEvent.click(await screen.findByRole('button', { name: 'Edit origin' }))
      await userEvent.type(screen.getByRole('textbox'), '2')
      await userEvent.click(screen.getByRole('button', { name: 'Save' }))

      expect(RemoteService.EditRemote).toHaveBeenCalledWith(REPO, 'origin', `${shownUrl}2`)
      expect(await screen.findByText(refusal.message)).toBeInTheDocument()
    })
  })
})
