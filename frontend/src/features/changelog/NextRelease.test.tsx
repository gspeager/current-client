import { render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { ChangelogService, HostService, TagService } from '@current-client-bindings/app'
import { DialogProvider } from '../../components/chrome/DialogProvider'
import NextRelease from './NextRelease'

const result = (over = {}) => ({
  ref: 'HEAD',
  sha: 'abc1234def',
  version: '0.5.0',
  previous: 'v0.4.0',
  target: 'main',
  tag: 'v0.5.0',
  taggedAs: '',
  checks: [
    { label: 'Working tree', level: 'ok', value: 'Clean', evidence: [], asOf: null },
    { label: 'build/config.yml', level: 'warn', value: 'No 0.5.0', evidence: ['line 3: still 0.4.0'], asOf: null },
    { label: 'Commits since v0.4.0', level: 'info', value: '2 commits', evidence: ['abc1234 feat: x'], asOf: null },
    { label: 'CI on the candidate', level: 'ok', value: 'Passed', evidence: [], asOf: new Date().toISOString() },
  ],
  ...over,
})

function renderBlock(onTagged = vi.fn()) {
  render(
    <DialogProvider>
      <NextRelease repoPath="/r/app" repoVersion={0} onTagged={onTagged} />
    </DialogProvider>,
  )
  return onTagged
}

describe('NextRelease', () => {
  it('lists the checks, what needs attention with its evidence, and the rest in a tooltip', async () => {
    vi.mocked(ChangelogService.ReleaseReadiness).mockResolvedValue(result() as never)
    vi.mocked(HostService.CanPublishReleases).mockResolvedValue(false)
    renderBlock()

    const checks = await screen.findByRole('list', { name: 'Checks' })
    expect(ChangelogService.ReleaseReadiness).toHaveBeenCalledWith('/r/app', '', '', '')
    const config = within(checks).getByText('build/config.yml').closest('li')!
    expect(config).toHaveTextContent('line 3: still 0.4.0')
    expect(within(config).getByRole('img', { name: 'Needs attention' })).toBeInTheDocument()
    const commits = within(checks).getByText('Commits since v0.4.0').closest('li')!
    expect(commits).not.toHaveTextContent('abc1234')
    expect(commits).toHaveAttribute('title', 'abc1234 feat: x')
    expect(within(checks).getByText('CI on the candidate').closest('li')).toHaveTextContent('just now')
    expect(screen.getByText(/from v0\.4\.0 · candidate HEAD/)).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Publish release…' })).toBeDisabled()
  })

  it('tags the candidate with the proposed version, and then makes Publish the next step', async () => {
    vi.mocked(ChangelogService.ReleaseReadiness)
      .mockResolvedValueOnce(result() as never)
      .mockResolvedValue(result({ taggedAs: 'v0.5.0' }) as never)
    vi.mocked(HostService.CanPublishReleases).mockResolvedValue(true)
    vi.mocked(TagService.CreateTag).mockResolvedValue()
    const onTagged = renderBlock()

    const tag = await screen.findByRole('button', { name: 'Tag v0.5.0…' })
    expect(tag).toHaveClass('next-release-step-primary')
    await userEvent.click(tag)
    const dialog = await screen.findByRole('alertdialog', { name: 'Tag v0.5.0' })
    await userEvent.click(within(dialog).getByRole('button', { name: 'Tag' }))
    await vi.waitFor(() =>
      expect(TagService.CreateTag).toHaveBeenCalledWith('/r/app', 'v0.5.0', 'Release 0.5.0', 'abc1234def'),
    )
    await vi.waitFor(() => expect(onTagged).toHaveBeenCalled())
    const publish = await screen.findByRole('button', { name: 'Publish release…' })
    await vi.waitFor(() => expect(publish).toBeEnabled())
    expect(publish).toHaveClass('next-release-step-primary')
    expect(screen.getByRole('button', { name: 'Tag v0.5.0…' })).toBeDisabled()
  })

  it('checks again with new inputs, and saves the version files', async () => {
    vi.mocked(ChangelogService.ReleaseReadiness).mockResolvedValue(result() as never)
    vi.mocked(ChangelogService.VersionFiles).mockResolvedValue([])
    vi.mocked(ChangelogService.SetVersionFiles).mockResolvedValue()
    renderBlock()
    await screen.findByRole('list', { name: 'Checks' })

    await userEvent.click(screen.getByRole('button', { name: 'Edit' }))
    const version = screen.getByRole('textbox', { name: 'Version' })
    await userEvent.clear(version)
    await userEvent.type(version, '1.0.0')
    await userEvent.click(screen.getByRole('button', { name: 'Check' }))
    expect(ChangelogService.ReleaseReadiness).toHaveBeenLastCalledWith('/r/app', 'HEAD', '1.0.0', 'main')

    await userEvent.type(screen.getByRole('textbox', { name: /Version files/ }), 'build/config.yml')
    await userEvent.click(screen.getByRole('button', { name: 'Save and check' }))
    expect(ChangelogService.SetVersionFiles).toHaveBeenCalledWith('/r/app', ['build/config.yml'])
  })
})
