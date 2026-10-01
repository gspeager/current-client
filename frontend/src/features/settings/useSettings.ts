import { useEffect, useState } from 'react'
import { PlatformService, SettingsService, type Settings } from '@current-client-bindings/app'
import type { ChangelogPrefs } from '@current-client-bindings/internal/config'
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
  const { element: root, active } = useCurrentClientRoot()
  const theme = settings?.theme

  useEffect(() => {
    SettingsService.GetSettings()
      .then(setSettings)
      .catch((err: unknown) => setError(errorMessage(err)))
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

  const setTheme = (theme: string) => {
    setSettings((prev) => (prev ? { ...prev, theme } : prev))
    SettingsService.SetTheme(theme).catch((err: unknown) => setError(errorMessage(err)))
  }

  const setGitExecutablePath = (path: string) =>
    SettingsService.SetGitExecutablePath(path)
      .then(() => {
        setSettings((prev) => (prev ? { ...prev, gitExecutablePath: path } : prev))
        setError(null)
      })
      .catch((err: unknown) => {
        setError(errorMessage(err))
        throw err
      })

  const setDefaultRepoLocation = (path: string) => {
    setSettings((prev) => (prev ? { ...prev, defaultRepoLocation: path } : prev))
    SettingsService.SetDefaultRepoLocation(path).catch((err: unknown) => setError(errorMessage(err)))
  }

  const setDiffIgnoreWhitespaceDefault = (value: boolean) => {
    setSettings((prev) => (prev ? { ...prev, diffIgnoreWhitespaceDefault: value } : prev))
    SettingsService.SetDiffIgnoreWhitespaceDefault(value).catch((err: unknown) => setError(errorMessage(err)))
  }

  const setEditorPath = (path: string) =>
    SettingsService.SetEditorPath(path)
      .then(() => {
        setSettings((prev) => (prev ? { ...prev, editorPath: path } : prev))
        setError(null)
      })
      .catch((err: unknown) => {
        setError(errorMessage(err))
        throw err
      })

  const setAutoFetchIntervalMinutes = (minutes: number) => {
    setSettings((prev) => (prev ? { ...prev, autoFetchIntervalMinutes: minutes } : prev))
    SettingsService.SetAutoFetchIntervalMinutes(minutes).catch((err: unknown) => setError(errorMessage(err)))
  }

  const setLaneColorTheme = (theme: string) => {
    setSettings((prev) => (prev ? { ...prev, laneColorTheme: theme } : prev))
    SettingsService.SetLaneColorTheme(theme).catch((err: unknown) => setError(errorMessage(err)))
  }

  const setNavCollapsed = (collapsed: boolean) => {
    setSettings((prev) => (prev ? { ...prev, navCollapsed: collapsed } : prev))
    SettingsService.SetNavCollapsed(collapsed).catch((err: unknown) => setError(errorMessage(err)))
  }

  const setChangelogPrefs = (changelog: ChangelogPrefs) => {
    setSettings((prev) => (prev ? { ...prev, changelog } : prev))
    SettingsService.SetChangelogPrefs(changelog).catch((err: unknown) => setError(errorMessage(err)))
  }

  const setDisableConventionalCommits = (disable: boolean) => {
    setSettings((prev) => (prev ? { ...prev, disableConventionalCommits: disable } : prev))
    SettingsService.SetDisableConventionalCommits(disable).catch((err: unknown) => setError(errorMessage(err)))
  }

  return {
    settings,
    error,
    setNavCollapsed,
    setDisableConventionalCommits,
    setChangelogPrefs,
    setTheme,
    setGitExecutablePath,
    setDefaultRepoLocation,
    setDiffIgnoreWhitespaceDefault,
    setEditorPath,
    setAutoFetchIntervalMinutes,
    setLaneColorTheme,
  }
}
