import { render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { HostService, HostInfo } from '@current-client-bindings/app'
import { DialogContext, type Dialogs } from '../../lib/useDialogs'
import IntegrationsSection from './IntegrationsSection'

function host(overrides: Partial<HostInfo>): HostInfo {
  return new HostInfo({
    remote: 'origin',
    hostname: 'gitlab.example.com',
    project: 'group/app',
    provider: 'gitlab',
    detectedProvider: 'gitlab',
    apiUrl: 'https://gitlab.example.com/api/v4',
    defaultApiUrl: 'https://gitlab.example.com/api/v4',
    ...overrides,
  })
}

function renderSection(dialogs: Partial<Dialogs> = {}) {
  const all: Dialogs = {
    confirm: vi.fn().mockResolvedValue(true),
    prompt: vi.fn().mockResolvedValue(null),
    signIn: vi.fn().mockResolvedValue(null),
    ...dialogs,
  }
  render(
    <DialogContext.Provider value={all}>
      <IntegrationsSection repoPath="/repos/app" />
    </DialogContext.Provider>,
  )
  return all
}

describe('IntegrationsSection', () => {
  it('shows a managed host with its switch disabled and labeled', async () => {
    vi.mocked(HostService.List).mockResolvedValue([host({ enabled: false, managed: true })])
    renderSection()

    const row = await screen.findByLabelText('gitlab.example.com')
    expect(within(row).getByText('Managed by your organization')).toBeInTheDocument()
    expect(within(row).getByRole('checkbox', { name: 'Integration for gitlab.example.com' })).toBeDisabled()
  })

  it('lists each host off by default, with its remote, provider and API address', async () => {
    vi.mocked(HostService.List).mockResolvedValue([
      host({}),
      host({
        remote: '',
        project: '',
        hostname: 'github.example.com',
        provider: 'github',
        apiUrl: 'https://github.example.com/api/v3',
      }),
    ])
    renderSection()

    const gitlab = await screen.findByLabelText('gitlab.example.com')
    expect(within(gitlab).getByText('group/app')).toBeInTheDocument()
    expect(within(gitlab).getByRole('checkbox', { name: 'Integration for gitlab.example.com' })).not.toBeChecked()
    expect(within(gitlab).getByRole('combobox', { name: 'Provider for gitlab.example.com' })).toHaveValue('gitlab')
    expect(within(gitlab).getByRole('textbox', { name: 'API address for gitlab.example.com' })).toHaveValue(
      'https://gitlab.example.com/api/v4',
    )
    expect(within(gitlab).getByRole('button', { name: 'Test connection' })).toBeDisabled()
    expect(within(screen.getByLabelText('github.example.com')).getByText('Added by hand')).toBeInTheDocument()
    expect(HostService.List).toHaveBeenCalledWith(['/repos/app'])
  })

  it('turns a host on and off through its switch', async () => {
    vi.mocked(HostService.List).mockResolvedValue([host({})])
    vi.mocked(HostService.SetEnabled).mockResolvedValue()
    renderSection()

    await userEvent.click(await screen.findByRole('checkbox', { name: 'Integration for gitlab.example.com' }))
    expect(HostService.SetEnabled).toHaveBeenCalledWith('gitlab.example.com', true)
    expect(HostService.List).toHaveBeenCalledTimes(2)
  })

  it('saves a pasted token through a masked prompt and offers to remove it', async () => {
    vi.mocked(HostService.List)
      .mockResolvedValueOnce([host({})])
      .mockResolvedValueOnce([host({ hasToken: true })])
    vi.mocked(HostService.SaveToken).mockResolvedValue()
    const prompt = vi.fn().mockResolvedValue('glpat-123')
    renderSection({ prompt })

    await userEvent.click(await screen.findByRole('button', { name: 'Set token…' }))
    expect(prompt).toHaveBeenCalledWith(expect.objectContaining({ secret: true, confirmLabel: 'Save' }))
    expect(HostService.SaveToken).toHaveBeenCalledWith('gitlab.example.com', 'glpat-123')
    expect(await screen.findByRole('button', { name: 'Remove token' })).toBeInTheDocument()
    expect(screen.getByText('Token saved with Git’s credential helper.')).toBeInTheDocument()
  })

  it('tests the connection of a host that is on and shows the answer', async () => {
    vi.mocked(HostService.List).mockResolvedValue([host({ enabled: true })])
    vi.mocked(HostService.TestConnection).mockResolvedValue('Connected to https://gitlab.example.com/api/v4 as ada.')
    renderSection()

    await userEvent.click(await screen.findByRole('button', { name: 'Test connection' }))
    expect(await screen.findByRole('status')).toHaveTextContent(
      'Connected to https://gitlab.example.com/api/v4 as ada.',
    )
  })

  it('shows a rejected token as the message, not the host’s answer', async () => {
    vi.mocked(HostService.List).mockResolvedValue([host({ enabled: true })])
    vi.mocked(HostService.TestConnection).mockRejectedValue(
      new Error('The token for gitlab.example.com was rejected. Update it in Settings → Integrations.'),
    )
    renderSection()

    await userEvent.click(await screen.findByRole('button', { name: 'Test connection' }))
    expect(await screen.findByRole('status')).toHaveTextContent('The token for gitlab.example.com was rejected.')
  })

  it('changes the provider and the API address', async () => {
    vi.mocked(HostService.List).mockResolvedValue([
      host({
        provider: 'github',
        apiUrl: 'https://gitlab.example.com/api/v3',
        defaultApiUrl: 'https://gitlab.example.com/api/v3',
      }),
    ])
    vi.mocked(HostService.SetHost).mockResolvedValue()
    renderSection()

    await userEvent.selectOptions(
      await screen.findByRole('combobox', { name: 'Provider for gitlab.example.com' }),
      'gitlab',
    )
    expect(HostService.SetHost).toHaveBeenCalledWith('gitlab.example.com', 'gitlab', '')

    const address = screen.getByRole('textbox', { name: 'API address for gitlab.example.com' })
    await userEvent.clear(address)
    await userEvent.type(address, 'https://gitlab.example.com/gitlab/api/v4{Enter}')
    expect(HostService.SetHost).toHaveBeenLastCalledWith(
      'gitlab.example.com',
      'github',
      'https://gitlab.example.com/gitlab/api/v4',
    )
  })

  it('adds a host by name and warns when Git skips certificate checks', async () => {
    vi.mocked(HostService.List).mockResolvedValue([host({ sslVerifyOff: true })])
    vi.mocked(HostService.Add).mockResolvedValue()
    const prompt = vi.fn().mockResolvedValue('gitlab.internal.example')
    renderSection({ prompt })

    expect(await screen.findByText(/http.sslVerify off/)).toBeInTheDocument()
    await userEvent.click(screen.getByRole('button', { name: 'Add host…' }))
    expect(HostService.Add).toHaveBeenCalledWith('gitlab.internal.example', '')
  })
})
