import { render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { HostService, IdentityService, SettingsService } from '@current-client-bindings/app'
import { DialogContext, type Dialogs } from '../../lib/useDialogs'
import type { useSettings } from './useSettings'
import SettingsPanel from './SettingsPanel'

const dialogs: Dialogs = { confirm: vi.fn(), prompt: vi.fn(), signIn: vi.fn() }

function renderPanel(settings: ReturnType<typeof useSettings>, hostIntegration = true, initialGroup?: string) {
  return render(
    <DialogContext.Provider value={dialogs}>
      <SettingsPanel
        settings={settings}
        hostIntegration={hostIntegration}
        initialGroup={initialGroup}
        onClose={vi.fn()}
      />
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
    setPruneOnFetch: vi.fn(),
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
    const rail = within(screen.getByRole('navigation', { name: 'Settings sections' }))
    const open = async (group: string) => {
      await userEvent.click(rail.getByRole('button', { name: group }))
      return within(screen.getByRole('region', { name: group }))
    }

    const appearance = await open('Appearance')
    expect(appearance.getByRole('button', { name: /Dark/ })).toBeInTheDocument()
    expect(appearance.getByRole('combobox', { name: 'Lane colors' })).toBeInTheDocument()

    const git = await open('Git')
    expect(await git.findByDisplayValue('Ada')).toBeInTheDocument()
    expect(git.getByRole('textbox', { name: 'Email' })).toHaveValue('ada@example.com')
    expect(git.getByRole('button', { name: 'Choose git executable' })).toBeInTheDocument()

    const tools = await open('Tools and folders')
    expect(tools.getByRole('button', { name: 'Choose editor' })).toBeInTheDocument()
    expect(tools.getByRole('button', { name: 'Choose default repository location' })).toBeInTheDocument()

    expect((await open('Diff')).getByRole('checkbox', { name: 'Ignore whitespace' })).toBeInTheDocument()
    expect((await open('Network')).getByRole('combobox', { name: 'Background fetch' })).toHaveValue('0')
    expect((await open('Integrations')).getByRole('button', { name: 'Add host…' })).toBeInTheDocument()
    await open('Identity colors')
    expect((await open('Support')).getByRole('button', { name: 'Copy diagnostics' })).toBeInTheDocument()
  })

  it('shows a managed background fetch disabled, with the label', () => {
    renderPanel(settingsWith({ autoFetchIntervalMinutes: 5, autoFetchManaged: true }), true, 'Network')

    const select = screen.getByRole('combobox', { name: 'Background fetch' })
    expect(select).toBeDisabled()
    expect(select).toHaveValue('5')
    expect(screen.getByText('Managed by organization policy.')).toBeInTheDocument()
  })

  it('has no Integrations group in a build without host integration', () => {
    renderPanel(settingsWith(), false)

    expect(screen.queryByRole('region', { name: 'Integrations' })).not.toBeInTheDocument()
    expect(HostService.List).not.toHaveBeenCalled()
  })

  it('shows a chosen path so its end stays visible, with a way back to the default', async () => {
    const settings = settingsWith({ gitExecutablePath: 'C:\\Program Files\\Git\\cmd\\git.exe' })
    renderPanel(settings, true, 'Git')

    const git = within(screen.getByRole('region', { name: 'Git' }))
    expect(git.getByTitle('C:\\Program Files\\Git\\cmd\\git.exe')).toHaveClass('path-text')
    await userEvent.click(git.getByRole('button', { name: 'Use PATH' }))

    expect(settings.setGitExecutablePath).toHaveBeenCalledWith('')
  })

  it('changes background fetch from its menu', async () => {
    const settings = settingsWith()
    renderPanel(settings, true, 'Network')

    await userEvent.selectOptions(screen.getByRole('combobox', { name: 'Background fetch' }), 'Every 15 minutes')

    expect(settings.setAutoFetchIntervalMinutes).toHaveBeenCalledWith(15)
  })
  it('turns Conventional Commits off, and it starts on', async () => {
    const settings = settingsWith()
    renderPanel(settings, true, 'Git')

    const toggle = screen.getByRole('checkbox', { name: 'Conventional Commits' })
    expect(toggle).toBeChecked()
    await userEvent.click(toggle)

    expect(settings.setDisableConventionalCommits).toHaveBeenCalledWith(true)
  })
})

describe('SettingsPanel network', () => {
  it('keeps prune on fetch as a setting', async () => {
    const settings = settingsWith({ pruneOnFetch: false } as never)
    renderPanel(settings, true, 'Network')

    await userEvent.click(screen.getByRole('checkbox', { name: 'Prune on fetch' }))

    expect(settings.setPruneOnFetch).toHaveBeenCalledWith(true)
  })
})

describe('SettingsPanel rail', () => {
  it('lists every group and shows only the one clicked', async () => {
    renderPanel(settingsWith())

    const rail = screen.getByRole('navigation', { name: 'Settings sections' })
    expect([...rail.querySelectorAll('button')].map((b) => b.textContent)).toEqual([
      'Appearance',
      'Git',
      'Tools and folders',
      'Diff',
      'Network',
      'Integrations',
      'Identity colors',
      'Support',
    ])
    await userEvent.click(within(rail).getByRole('button', { name: 'Integrations' }))

    expect(within(rail).getByRole('button', { name: 'Integrations' })).toHaveClass('settings-rail-item-active')
    expect(screen.getByRole('region', { name: 'Integrations' })).toBeVisible()
    expect(screen.queryByRole('region', { name: 'Appearance' })).not.toBeInTheDocument()
    expect(document.querySelector('.settings-group[aria-label="Appearance"]')).not.toBeVisible()
    expect(screen.getByRole('dialog')).toHaveClass('settings-panel')
  })
})
