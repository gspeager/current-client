import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { GitService } from '@current-client-bindings/app'
import GitSettingsSection from './GitSettingsSection'

const entries = [
  { key: 'user.email', value: 'ada@example.com', scope: 'global', origin: '/home/ada/.gitconfig' },
  { key: 'user.email', value: 'ada@work.example', scope: 'local', origin: '.git/config' },
]

describe('GitSettingsSection', () => {
  it('shows flags, and every setting with its scope and origin on request', async () => {
    const user = userEvent.setup()
    vi.mocked(GitService.GetGitSettings).mockResolvedValue({
      entries,
      flags: [{ key: 'user.email', message: 'The author email is ada@work.example.' }],
    })
    render(<GitSettingsSection repoPath="/repo" />)

    expect(await screen.findByText('The author email is ada@work.example.')).toBeInTheDocument()
    expect(GitService.GetGitSettings).toHaveBeenCalledWith('/repo')
    expect(screen.queryByRole('table')).not.toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: 'Show all 2 settings' }))

    expect(screen.getByRole('table')).toBeInTheDocument()
    expect(screen.getByText('ada@example.com')).toBeInTheDocument()
    expect(screen.getByText('local')).toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: 'Hide settings' }))
    expect(screen.queryByRole('table')).not.toBeInTheDocument()
  })

  it('reads them again when the identity is saved', async () => {
    vi.mocked(GitService.GetGitSettings).mockResolvedValue({ entries, flags: [] })
    const { rerender } = render(<GitSettingsSection repoPath="/repo" refreshKey={0} />)
    await screen.findByText('Nothing unusual.')
    vi.mocked(GitService.GetGitSettings).mockClear()
    vi.mocked(GitService.GetGitSettings).mockResolvedValue({
      entries,
      flags: [{ key: 'user.email', message: 'Changed.' }],
    })

    rerender(<GitSettingsSection repoPath="/repo" refreshKey={1} />)

    expect(await screen.findByText('Changed.')).toBeInTheDocument()
    expect(GitService.GetGitSettings).toHaveBeenCalledTimes(1)
  })

  it('says when there is nothing to flag', async () => {
    vi.mocked(GitService.GetGitSettings).mockResolvedValue({ entries, flags: [] })
    render(<GitSettingsSection repoPath="/repo" />)

    expect(await screen.findByText('Nothing unusual.')).toBeInTheDocument()
  })

  it("shows the reason it can't read them", async () => {
    vi.mocked(GitService.GetGitSettings).mockRejectedValue(
      new Error('showing where Git settings come from needs Git 2.26 or newer'),
    )
    render(<GitSettingsSection repoPath="/repo" />)

    expect(await screen.findByText(/needs Git 2\.26 or newer/)).toBeInTheDocument()
  })
})
