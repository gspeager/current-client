import { render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { HostService, IdentityService, SettingsService } from '@current-client-bindings/app'
import { DialogContext, type Dialogs } from '../../lib/useDialogs'
import type { useSettings } from './useSettings'
import SettingsPanel from './SettingsPanel'

const dialogs: Dialogs = { confirm: vi.fn(), prompt: vi.fn(), signIn: vi.fn() }

function renderPanel(settings: ReturnType<typeof useSettings>, hostIntegration = true) {
  return render(
    <DialogContext.Provider value={dialogs}>
      <SettingsPanel settings={settings} hostIntegration={hostIntegration} onClose={vi.fn()} />
    </DialogContext.Provider>,
  )
}

function settingsWith(overrides: Partial<NonNullable<ReturnType<typeof useSettings>['settings']>> = {}) {
  return {
    settings: {
      theme: 'dark',
      gitExecutablePath: '',
      defaultRepoLocation: '',
      diffIgnoreWhitespaceDefault: false,
      editorPath: '',
      autoFetchIntervalMinutes: 0,
      laneColorTheme: 'default',
      ...overrides,
    },
    error: null,
    setTheme: vi.fn(),
    setGitExecutablePath: vi.fn().mockResolvedValue(undefined),
    setDefaultRepoLocation: vi.fn(),
    setDiffIgnoreWhitespaceDefault: vi.fn(),
    setEditorPath: vi.fn().mockResolvedValue(undefined),
    setAutoFetchIntervalMinutes: vi.fn(),
    setLaneColorTheme: vi.fn(),
    setDisableConventionalCommits: vi.fn(),
  } as unknown as ReturnType<typeof useSettings>
}

beforeEach(() => {
  vi.mocked(IdentityService.GetGlobalUser).mockResolvedValue({ name: 'Ada', email: 'ada@example.com' })
  vi.mocked(SettingsService.GetIdentityColorOverrides).mockResolvedValue({})
  vi.mocked(HostService.List).mockResolvedValue([])
})

describe('SettingsPanel', () => {
  it('groups every setting and keeps each control reachable by its label', async () => {
    renderPanel(settingsWith())

    for (const group of [
      'Appearance',
      'Git',
      'Tools and folders',
      'Diff',
      'Network',
      'Integrations',
      'Identity colors',
      'Support',
    ]) {
      expect(screen.getByRole('region', { name: group })).toBeInTheDocument()
    }
    expect(
      within(screen.getByRole('region', { name: 'Appearance' })).getByRole('button', { name: /Dark/ }),
    ).toBeInTheDocument()
    expect(screen.getByRole('combobox', { name: 'Lane colors' })).toBeInTheDocument()
    expect(await screen.findByDisplayValue('Ada')).toBeInTheDocument()
    expect(screen.getByRole('textbox', { name: 'Email' })).toHaveValue('ada@example.com')
    for (const name of [
      'Choose git executable',
      'Choose editor',
      'Choose default repository location',
      'Copy diagnostics',
    ]) {
      expect(screen.getByRole('button', { name })).toBeInTheDocument()
    }
    expect(screen.getByRole('checkbox', { name: 'Ignore whitespace' })).toBeInTheDocument()
    expect(screen.getByRole('combobox', { name: 'Background fetch' })).toHaveValue('0')
    expect(screen.getByRole('button', { name: 'Add host…' })).toBeInTheDocument()
  })

  it('shows a managed background fetch disabled, with the label', () => {
    renderPanel(settingsWith({ autoFetchIntervalMinutes: 5, autoFetchManaged: true }))

    const select = screen.getByRole('combobox', { name: 'Background fetch' })
    expect(select).toBeDisabled()
    expect(select).toHaveValue('5')
    expect(screen.getByText('Managed by your organization.')).toBeInTheDocument()
  })

  it('has no Integrations group in a build without host integration', () => {
    renderPanel(settingsWith(), false)

    expect(screen.queryByRole('region', { name: 'Integrations' })).not.toBeInTheDocument()
    expect(HostService.List).not.toHaveBeenCalled()
  })

  it('shows a chosen path so its end stays visible, with a way back to the default', async () => {
    const settings = settingsWith({ gitExecutablePath: 'C:\\Program Files\\Git\\cmd\\git.exe' })
    renderPanel(settings)

    const git = within(screen.getByRole('region', { name: 'Git' }))
    expect(git.getByTitle('C:\\Program Files\\Git\\cmd\\git.exe')).toHaveClass('path-text')
    await userEvent.click(git.getByRole('button', { name: 'Use PATH' }))

    expect(settings.setGitExecutablePath).toHaveBeenCalledWith('')
  })

  it('changes background fetch from its menu', async () => {
    const settings = settingsWith()
    renderPanel(settings)

    await userEvent.selectOptions(screen.getByRole('combobox', { name: 'Background fetch' }), 'Every 15 minutes')

    expect(settings.setAutoFetchIntervalMinutes).toHaveBeenCalledWith(15)
  })
  it('turns Conventional Commits off, and it starts on', async () => {
    const settings = settingsWith()
    renderPanel(settings)

    const toggle = screen.getByRole('checkbox', { name: 'Conventional Commits' })
    expect(toggle).toBeChecked()
    await userEvent.click(toggle)

    expect(settings.setDisableConventionalCommits).toHaveBeenCalledWith(true)
  })
})
