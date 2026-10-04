import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { RemoteService } from '@current-client-bindings/app'
import { DialogProvider } from '../../components/chrome/DialogProvider'
import RemotesPanel from './RemotesPanel'

const REPO = '/repos/app'

function renderPanel(pruneOnFetch: boolean) {
  vi.mocked(RemoteService.List).mockResolvedValue([
    { name: 'origin', fetchUrl: 'https://example.com/app.git', pushUrl: '' },
    { name: 'upstream', fetchUrl: 'https://example.com/upstream.git', pushUrl: '' },
  ])
  const onFetched = vi.fn()
  render(
    <DialogProvider>
      <RemotesPanel repoPath={REPO} onFetched={onFetched} pruneOnFetch={pruneOnFetch} />
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
})
