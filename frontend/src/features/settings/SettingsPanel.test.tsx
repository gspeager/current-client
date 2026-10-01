import { render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { IdentityService, SettingsService } from '@current-client-bindings/app'
import type { useSettings } from './useSettings'
import SettingsPanel from './SettingsPanel'

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
})

describe('SettingsPanel', () => {
  it('groups every setting and keeps each control reachable by its label', async () => {
    render(<SettingsPanel settings={settingsWith()} onClose={vi.fn()} />)

    for (const group of ['Appearance', 'Git', 'Tools and folders', 'Diff', 'Network', 'Identity colors', 'Support']) {
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
  })

  it('shows a chosen path so its end stays visible, with a way back to the default', async () => {
    const settings = settingsWith({ gitExecutablePath: 'C:\\Program Files\\Git\\cmd\\git.exe' })
    render(<SettingsPanel settings={settings} onClose={vi.fn()} />)

    const git = within(screen.getByRole('region', { name: 'Git' }))
    expect(git.getByTitle('C:\\Program Files\\Git\\cmd\\git.exe')).toHaveClass('path-text')
    await userEvent.click(git.getByRole('button', { name: 'Use PATH' }))

    expect(settings.setGitExecutablePath).toHaveBeenCalledWith('')
  })

  it('changes background fetch from its menu', async () => {
    const settings = settingsWith()
    render(<SettingsPanel settings={settings} onClose={vi.fn()} />)

    await userEvent.selectOptions(screen.getByRole('combobox', { name: 'Background fetch' }), 'Every 15 minutes')

    expect(settings.setAutoFetchIntervalMinutes).toHaveBeenCalledWith(15)
  })
  it('turns Conventional Commits off, and it starts on', async () => {
    const settings = settingsWith()
    render(<SettingsPanel settings={settings} onClose={vi.fn()} />)

    const toggle = screen.getByRole('checkbox', { name: 'Conventional Commits' })
    expect(toggle).toBeChecked()
    await userEvent.click(toggle)

    expect(settings.setDisableConventionalCommits).toHaveBeenCalledWith(true)
  })
})
