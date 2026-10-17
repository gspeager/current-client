import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { GitService, PlatformService } from '@current-client-bindings/app'
import GitMissing from './GitMissing'

const TOOLS_MISSING = "Apple's Command Line Tools aren't installed. Git comes with them."

describe('GitMissing', () => {
  it("offers Apple's installer when the Command Line Tools are missing, and checks again", async () => {
    vi.mocked(GitService.CommandLineToolsMissing).mockResolvedValue(true)
    vi.mocked(GitService.InstallCommandLineTools).mockResolvedValue()
    const onCheckAgain = vi.fn()
    render(<GitMissing error={TOOLS_MISSING} onCheckAgain={onCheckAgain} onSetPath={vi.fn()} />)

    await userEvent.click(await screen.findByRole('button', { name: 'Install Command Line Tools' }))
    expect(GitService.InstallCommandLineTools).toHaveBeenCalled()
    expect(screen.getByText(/takes a few minutes/)).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Download Git' })).not.toBeInTheDocument()

    await userEvent.click(screen.getByRole('button', { name: 'Check again' }))
    expect(onCheckAgain).toHaveBeenCalled()
  })

  it('offers the Git download and a path otherwise', async () => {
    vi.mocked(GitService.CommandLineToolsMissing).mockResolvedValue(false)
    vi.mocked(PlatformService.OpenGitDownloadPage).mockResolvedValue()
    const onSetPath = vi.fn()
    render(<GitMissing error="Git not found." onCheckAgain={vi.fn()} onSetPath={onSetPath} />)

    await userEvent.click(screen.getByRole('button', { name: 'Download Git' }))
    expect(PlatformService.OpenGitDownloadPage).toHaveBeenCalled()
    await userEvent.click(screen.getByRole('button', { name: 'Set git executable path…' }))
    expect(onSetPath).toHaveBeenCalled()
    expect(screen.queryByRole('button', { name: 'Install Command Line Tools' })).not.toBeInTheDocument()
  })
})
