import { render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { TrackerService } from '@current-client-bindings/app'
import { DialogProvider } from '../../components/chrome/DialogProvider'
import TrackersSection from './TrackersSection'

const cloud = { site: 'https://example.atlassian.net/', kind: 'jira-cloud', keys: [], repos: [], managed: false }

function renderSection() {
  render(
    <DialogProvider>
      <TrackersSection />
    </DialogProvider>,
  )
}

describe('TrackersSection', () => {
  it('adds a tracker, gives it a repository and removes it', async () => {
    vi.mocked(TrackerService.List)
      .mockResolvedValueOnce({ trackers: [], default: null, openRepos: ['/r/app'] })
      .mockResolvedValueOnce({ trackers: [cloud], default: null, openRepos: ['/r/app'] })
      .mockResolvedValueOnce({ trackers: [{ ...cloud, repos: ['/r/app'] }], default: null, openRepos: ['/r/app'] })
      .mockResolvedValue({ trackers: [], default: null, openRepos: ['/r/app'] })
    vi.mocked(TrackerService.Add).mockResolvedValue(cloud)
    vi.mocked(TrackerService.SetRepos).mockResolvedValue()
    vi.mocked(TrackerService.Remove).mockResolvedValue()
    renderSection()

    await userEvent.click(await screen.findByRole('button', { name: 'Add issue tracker…' }))
    const form = screen.getByRole('form', { name: 'Add issue tracker' })
    await userEvent.type(within(form).getByRole('textbox', { name: 'Site address' }), 'https://example.atlassian.net')
    await userEvent.type(within(form).getByRole('textbox', { name: /Project keys/ }), 'PROJ, ops')
    await userEvent.click(within(form).getByRole('button', { name: 'Add' }))
    expect(TrackerService.Add).toHaveBeenCalledWith('https://example.atlassian.net', 'jira-cloud', ['PROJ', 'ops'])

    const row = await screen.findByRole('generic', { name: 'https://example.atlassian.net/' })
    await userEvent.click(within(row).getByRole('checkbox', { name: 'app' }))
    expect(TrackerService.SetRepos).toHaveBeenCalledWith('https://example.atlassian.net/', ['/r/app'])
    expect(await within(row).findByRole('checkbox', { name: 'app' })).toBeChecked()

    await userEvent.click(within(row).getByRole('button', { name: 'Remove' }))
    await userEvent.click(within(await screen.findByRole('alertdialog')).getByRole('button', { name: 'Remove' }))
    expect(TrackerService.Remove).toHaveBeenCalledWith('https://example.atlassian.net/')
  })

  it('shows the organization’s default as set by it, with nothing to change', async () => {
    vi.mocked(TrackerService.List).mockResolvedValue({
      trackers: [],
      default: { site: 'https://corp.atlassian.net/', kind: 'jira-cloud', keys: ['OPS'], repos: [], managed: true },
      openRepos: [],
    })
    renderSection()
    const row = await screen.findByRole('generic', { name: 'https://corp.atlassian.net/' })
    expect(row).toHaveTextContent('Set by your organization')
    expect(row).toHaveTextContent('Used by every repository without a tracker of its own, for OPS.')
    expect(within(row).queryByRole('button', { name: 'Remove' })).not.toBeInTheDocument()
  })
})
