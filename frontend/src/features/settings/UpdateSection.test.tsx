import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { PlatformService, UpdateService } from '@current-client-bindings/app'
import UpdateSection from './UpdateSection'

describe('UpdateSection', () => {
  beforeEach(() => {
    vi.mocked(PlatformService.AppVersion).mockResolvedValue('0.4.0')
    vi.mocked(UpdateService.ReleasesURL).mockResolvedValue('https://github.com/gspeager/current-client/releases')
    vi.mocked(PlatformService.OpenWebPage).mockResolvedValue(undefined)
  })

  it('checks only on the click, and offers the releases page when a newer version exists', async () => {
    vi.mocked(UpdateService.Check).mockResolvedValue({
      state: 'newer',
      current: '0.4.0',
      latest: '0.5.0',
      releasedAt: '2026-10-30',
      latestUrl: 'https://github.com/gspeager/current-client/releases/tag/v0.5.0',
    } as never)
    render(<UpdateSection />)
    expect(await screen.findByText('0.4.0')).toBeInTheDocument()
    expect(screen.getByText('https://github.com/gspeager/current-client/releases')).toBeInTheDocument()
    expect(UpdateService.Check).not.toHaveBeenCalled()

    await userEvent.click(screen.getByRole('button', { name: 'Check for updates' }))
    expect(await screen.findByRole('status')).toHaveTextContent('Version 0.5.0 is available, released 2026-10-30.')
    expect(UpdateService.Check).toHaveBeenCalledTimes(1)
    await userEvent.click(screen.getByRole('button', { name: 'Open releases page' }))
    expect(PlatformService.OpenWebPage).toHaveBeenCalledWith(
      'https://github.com/gspeager/current-client/releases/tag/v0.5.0',
    )
  })

  it('says when up to date, and keeps the releases link when the check fails', async () => {
    vi.mocked(UpdateService.Check).mockResolvedValueOnce({
      state: 'up-to-date',
      current: '0.4.0',
      latest: '0.4.0',
      releasedAt: '',
      latestUrl: '',
    } as never)
    vi.mocked(UpdateService.Check).mockRejectedValueOnce(
      new Error(
        'Couldn’t reach github.com. Check your connection, or look at the releases page from another computer.',
      ),
    )
    render(<UpdateSection />)
    await userEvent.click(await screen.findByRole('button', { name: 'Check for updates' }))
    expect(await screen.findByRole('status')).toHaveTextContent('You’re up to date (0.4.0).')
    expect(screen.queryByRole('button', { name: 'Open releases page' })).toBeNull()

    await userEvent.click(screen.getByRole('button', { name: 'Check for updates' }))
    await waitFor(() => expect(screen.getByText(/Couldn’t reach github.com/)).toBeInTheDocument())
    expect(screen.getByText('https://github.com/gspeager/current-client/releases')).toBeInTheDocument()
    await userEvent.click(screen.getByRole('button', { name: 'Open' }))
    expect(PlatformService.OpenWebPage).toHaveBeenCalledWith('https://github.com/gspeager/current-client/releases')
  })
})
