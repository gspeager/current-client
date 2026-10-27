import { render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { TrackerService } from '@current-client-bindings/app'
import { DialogProvider } from '../../components/chrome/DialogProvider'
import TrackersSection from './TrackersSection'

const access = { hostName: 'example.atlassian.net', enabled: false, locked: false, hasToken: false }
const cloud = {
  site: 'https://example.atlassian.net/',
  kind: 'jira-cloud',
  keys: [],
  repos: [],
  managed: false,
  ...access,
}

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
      default: {
        site: 'https://corp.atlassian.net/',
        kind: 'jira-cloud',
        keys: ['OPS'],
        repos: [],
        managed: true,
        ...access,
      },
      openRepos: [],
    })
    renderSection()
    const row = await screen.findByRole('generic', { name: 'https://corp.atlassian.net/' })
    expect(row).toHaveTextContent('Set by your organization')
    expect(row).toHaveTextContent('Used by every repository without a tracker of its own, for OPS.')
    expect(within(row).queryByRole('button', { name: 'Remove' })).not.toBeInTheDocument()
  })

  it('turns the integration on, saves a Cloud token with its email, and tests the connection', async () => {
    vi.mocked(TrackerService.List)
      .mockResolvedValueOnce({ trackers: [cloud], default: null, openRepos: [] })
      .mockResolvedValue({ trackers: [{ ...cloud, enabled: true }], default: null, openRepos: [] })
    vi.mocked(TrackerService.SetEnabled).mockResolvedValue()
    vi.mocked(TrackerService.SaveToken).mockResolvedValue()
    vi.mocked(TrackerService.TestConnection).mockResolvedValue('Signed in to example.atlassian.net as Ada.')
    renderSection()
    const row = await screen.findByRole('generic', { name: 'https://example.atlassian.net/' })
    expect(within(row).getByRole('button', { name: 'Test connection' })).toBeDisabled()
    await userEvent.click(within(row).getByRole('checkbox', { name: 'Integration for https://example.atlassian.net/' }))
    expect(TrackerService.SetEnabled).toHaveBeenCalledWith('https://example.atlassian.net/', true)

    await userEvent.click(within(row).getByRole('button', { name: 'Set token…' }))
    await userEvent.type(await screen.findByRole('textbox'), 'ada@example.com{enter}')
    await userEvent.type(await screen.findByLabelText(/An API token/), 'secret{enter}')
    expect(TrackerService.SaveToken).toHaveBeenCalledWith('https://example.atlassian.net/', 'ada@example.com', 'secret')

    await userEvent.click(await within(row).findByRole('button', { name: 'Test connection' }))
    expect(await within(row).findByRole('status')).toHaveTextContent('Signed in to example.atlassian.net as Ada.')
  })

  it('shows a switch the organization locked as managed', async () => {
    vi.mocked(TrackerService.List).mockResolvedValue({
      trackers: [{ ...cloud, locked: true }],
      default: null,
      openRepos: [],
    })
    renderSection()
    const row = await screen.findByRole('generic', { name: 'https://example.atlassian.net/' })
    expect(row).toHaveTextContent('Managed by your organization')
    expect(within(row).getByRole('checkbox', { name: /Integration for/ })).toBeDisabled()
  })
})
