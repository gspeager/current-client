import { useEffect, useState } from 'react'
import { PlatformService, SettingsService, type Settings } from '@current-client-bindings/app'
import { useCurrentClientRoot } from '../../lib/currentClientRoot'
import { errorMessage } from '../../lib/errors'

function systemPrefersLight(): boolean {
  return window.matchMedia?.('(prefers-color-scheme: light)').matches ?? false
}

// titleBar is false while a host app shows another view, so Current Client doesn't
// theme a window it isn't showing in.
function applyTheme(root: HTMLElement, theme: string, titleBar: boolean) {
  const light = theme === 'system' ? systemPrefersLight() : theme === 'light'
  if (light) {
    root.setAttribute('data-theme', 'light')
  } else {
    root.removeAttribute('data-theme')
  }
  if (titleBar) PlatformService.SetDarkTitleBar(!light).catch(() => undefined)
}

export function useSettings() {
  const [settings, setSettings] = useState<Settings | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [resetNotice, setResetNotice] = useState('')
  const { element: root, active } = useCurrentClientRoot()
  const theme = settings?.theme

  useEffect(() => {
    SettingsService.GetSettings()
      .then(setSettings)
      .catch((err: unknown) => setError(errorMessage(err)))
      .finally(() => SettingsService.TakeSettingsResetNotice().then(setResetNotice, () => undefined))
  }, [])

  // Applies the theme to the Current Client root and the window's title bar, and follows
  // OS changes live while "system" is selected.
  useEffect(() => {
    if (!root || theme === undefined) return
    const apply = () => applyTheme(root, theme, active)
    apply()
    if (theme !== 'system' || !window.matchMedia) return
    const media = window.matchMedia('(prefers-color-scheme: light)')
    media.addEventListener('change', apply)
    return () => media.removeEventListener('change', apply)
  }, [root, theme, active])

  const patch = <K extends keyof Settings>(key: K, value: Settings[K]) =>
    setSettings((prev) => (prev ? { ...prev, [key]: value } : prev))

  // The setting shows its new value at once; a failed save reports and leaves it.
  const setter =
    <K extends keyof Settings>(key: K, save: (value: Settings[K]) => Promise<void>) =>
    (value: Settings[K]) => {
      patch(key, value)
      save(value).catch((err: unknown) => setError(errorMessage(err)))
    }

  // Paths are checked by the backend first, so they only show once saved.
  const checkedSetter =
    (key: 'gitExecutablePath' | 'editorPath', save: (path: string) => Promise<void>) => (path: string) =>
      save(path)
        .then(() => {
          patch(key, path)
          setError(null)
        })
        .catch((err: unknown) => {
          setError(errorMessage(err))
          throw err
        })

  return {
    settings,
    error,
    resetNotice,
    dismissResetNotice: () => setResetNotice(''),
    setNavCollapsed: setter('navCollapsed', SettingsService.SetNavCollapsed),
    setDiffExpanded: setter('diffExpanded', SettingsService.SetDiffExpanded),
    setDiffNoWrap: setter('diffNoWrap', SettingsService.SetDiffNoWrap),
    setPruneOnFetch: setter('pruneOnFetch', SettingsService.SetPruneOnFetch),
    setHistoryBranchFilter: setter('historyBranchFilter', SettingsService.SetHistoryBranchFilter),
    setHistoryFirstParent: setter('historyFirstParent', SettingsService.SetHistoryFirstParent),
    setDisableConventionalCommits: setter('disableConventionalCommits', SettingsService.SetDisableConventionalCommits),
    setChangelogPrefs: setter('changelog', SettingsService.SetChangelogPrefs),
    setTheme: setter('theme', SettingsService.SetTheme),
    setGitExecutablePath: checkedSetter('gitExecutablePath', SettingsService.SetGitExecutablePath),
    setDefaultRepoLocation: setter('defaultRepoLocation', SettingsService.SetDefaultRepoLocation),
    setDiffIgnoreWhitespaceDefault: setter(
      'diffIgnoreWhitespaceDefault',
      SettingsService.SetDiffIgnoreWhitespaceDefault,
    ),
    setEditorPath: checkedSetter('editorPath', SettingsService.SetEditorPath),
    setAutoFetchIntervalMinutes: setter('autoFetchIntervalMinutes', SettingsService.SetAutoFetchIntervalMinutes),
    setLaneColorTheme: setter('laneColorTheme', SettingsService.SetLaneColorTheme),
  }
}
