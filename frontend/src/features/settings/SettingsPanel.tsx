import { useRef, useState, type UIEvent } from 'react'
import { Monitor, Moon, Sun } from 'lucide-react'
import { SettingsService } from '@current-client-bindings/app'
import SegmentedControl from '../../components/controls/SegmentedControl'
import Switch from '../../components/forms/Switch'
import { errorMessage } from '../../lib/errors'
import { LANE_THEME_NAMES, LANE_THEMES, toLaneThemeName, type LaneThemeName } from '../../lib/laneColor'
import type { useSettings } from './useSettings'
import DiagnosticsSection from './DiagnosticsSection'
import UpdateSection from './UpdateSection'
import IdentityColorSettings from './IdentityColorSettings'
import IdentitySettings from './IdentitySettings'
import IntegrationsSection from './IntegrationsSection'
import TrackersSection from './TrackersSection'
import Modal from '../../components/chrome/Modal'
import PathSettingRow from './PathSettingRow'
import SettingsRow, { SettingsGroup, settingsGroupId } from './SettingsRow'
import './SettingsPanel.scss'

const LANE_THEME_LABELS: Record<LaneThemeName, string> = {
  default: 'Default',
  vivid: 'Vivid',
  muted: 'Muted',
}

const AUTO_FETCH_OPTIONS = [
  { minutes: 0, label: 'Off' },
  { minutes: 5, label: 'Every 5 minutes' },
  { minutes: 15, label: 'Every 15 minutes' },
  { minutes: 30, label: 'Every 30 minutes' },
  { minutes: 60, label: 'Every hour' },
]

interface SettingsPanelProps {
  settings: ReturnType<typeof useSettings>
  repoPath?: string | null
  // false in a build made without host integration; the group is then absent.
  hostIntegration?: boolean
  // false when an embedding app has its own releases page and version.
  updateCheck?: boolean
  onClose: () => void
  onHostsChanged?: () => void
}

function SettingsPanel({
  settings,
  repoPath = null,
  hostIntegration = true,
  updateCheck = true,
  onClose,
  onHostsChanged,
}: SettingsPanelProps) {
  const {
    settings: values,
    error,
    setTheme,
    setGitExecutablePath,
    setDefaultRepoLocation,
    setDiffIgnoreWhitespaceDefault,
    setEditorPath,
    setAutoFetchIntervalMinutes,
    setLaneColorTheme,
    setDisableConventionalCommits,
    setPruneOnFetch,
  } = settings
  const [gitPathError, setGitPathError] = useState<string | null>(null)
  const [editorPathError, setEditorPathError] = useState<string | null>(null)
  const [section, setSection] = useState('Appearance')
  const groups = [
    'Appearance',
    'Git',
    'Tools and folders',
    'Diff',
    'Network',
    ...(hostIntegration ? ['Integrations'] : []),
    'Identity colors',
    'Support',
  ]

  // The scroll a pick causes keeps the pick, even for a group that can't reach the top.
  const picked = useRef<string | null>(null)

  const showGroup = (title: string) => {
    picked.current = title
    setSection(title)
    document.getElementById(settingsGroupId(title))?.scrollIntoView({ block: 'start' })
  }

  // The reading line slides to the bottom as the body nears its end, so the
  // last groups, which never reach the top, still get their turn.
  const followScroll = (e: UIEvent<HTMLDivElement>) => {
    if (picked.current) {
      picked.current = null
      return
    }
    const body = e.currentTarget
    const progress = body.scrollTop / Math.max(1, body.scrollHeight - body.clientHeight)
    const line = body.getBoundingClientRect().top + 8 + progress * (body.clientHeight - 8)
    const passed = groups.filter(
      (title) => (document.getElementById(settingsGroupId(title))?.getBoundingClientRect().top ?? Infinity) <= line,
    )
    setSection(passed[passed.length - 1] ?? groups[0])
  }

  // Picks a file or folder, then saves it; an empty choice (dialog cancelled) does nothing.
  const choosePath =
    (pick: () => Promise<string>, save: (path: string) => Promise<void> | void, setError: (e: string | null) => void) =>
    () => {
      pick()
        .then((path) => {
          if (!path) return
          setError(null)
          return save(path)
        })
        .catch((err: unknown) => setError(errorMessage(err)))
    }

  const resetPath = (save: (path: string) => Promise<void>, setError: (e: string | null) => void) => () => {
    setError(null)
    save('').catch((err: unknown) => setError(errorMessage(err)))
  }

  if (!values) {
    return null
  }

  const activeLaneTheme = toLaneThemeName(values.laneColorTheme)

  return (
    <Modal title="Settings" onClose={onClose} className="settings-panel">
      <div className="settings-layout">
        <nav className="settings-rail" aria-label="Settings sections">
          {groups.map((title) => (
            <button
              key={title}
              type="button"
              className={title === section ? 'settings-rail-item settings-rail-item-active' : 'settings-rail-item'}
              onClick={() => showGroup(title)}
            >
              {title}
            </button>
          ))}
        </nav>
        <div className="settings-body" onScroll={followScroll}>
          {error && <p className="settings-error">{error}</p>}

          <SettingsGroup title="Appearance">
            <SettingsRow label="Theme" hint="System follows the operating system's light or dark setting.">
              <SegmentedControl
                value={values.theme}
                onChange={setTheme}
                options={[
                  { value: 'system', label: 'System', icon: Monitor },
                  { value: 'dark', label: 'Dark', icon: Moon },
                  { value: 'light', label: 'Light', icon: Sun },
                ]}
              />
            </SettingsRow>
            <SettingsRow label="Lane colors" hint="Branch colors in the commit graph, branch pills and sidebar dots.">
              <select
                className="settings-field"
                value={activeLaneTheme}
                onChange={(e) => setLaneColorTheme(e.target.value)}
                aria-label="Lane colors"
              >
                {LANE_THEME_NAMES.map((name) => (
                  <option key={name} value={name}>
                    {LANE_THEME_LABELS[name]}
                  </option>
                ))}
              </select>
              <div className="settings-lane-swatches" aria-hidden="true">
                {LANE_THEMES[activeLaneTheme].map((color) => (
                  <span key={color} className="settings-lane-swatch" style={{ background: color }} />
                ))}
              </div>
            </SettingsRow>
          </SettingsGroup>

          <SettingsGroup title="Git">
            <IdentitySettings />
            <PathSettingRow
              label="Git executable"
              hint="Git 2.23 or newer."
              path={values.gitExecutablePath}
              fallback="Found on PATH"
              error={gitPathError}
              onChoose={choosePath(SettingsService.PickGitExecutable, setGitExecutablePath, setGitPathError)}
              onReset={resetPath(setGitExecutablePath, setGitPathError)}
              resetLabel="Use PATH"
            />
            <SettingsRow
              label="Conventional Commits"
              hint="Adds the Changelog tab to the tab bar and the type picker to the commit form. History still shows types in repositories that use the format."
            >
              <Switch
                checked={!values.disableConventionalCommits}
                onChange={(on) => setDisableConventionalCommits(!on)}
                ariaLabel="Conventional Commits"
              />
            </SettingsRow>
          </SettingsGroup>

          <SettingsGroup title="Tools and folders">
            <PathSettingRow
              label="Editor"
              hint="Opened by Open in Editor."
              path={values.editorPath}
              fallback="VS Code (code, on PATH)"
              error={editorPathError}
              onChoose={choosePath(SettingsService.PickEditorExecutable, setEditorPath, setEditorPathError)}
              onReset={resetPath(setEditorPath, setEditorPathError)}
              resetLabel="Use default"
            />
            <PathSettingRow
              label="Default repository location"
              hint="Where the Open and Clone dialogs start."
              path={values.defaultRepoLocation}
              fallback="Not set"
              onChoose={choosePath(SettingsService.PickDefaultRepoLocation, setDefaultRepoLocation, () => undefined)}
            />
          </SettingsGroup>

          <SettingsGroup title="Diff">
            <SettingsRow label="Ignore whitespace" hint="The default for new diffs; each diff can still switch it.">
              <Switch
                checked={values.diffIgnoreWhitespaceDefault}
                onChange={setDiffIgnoreWhitespaceDefault}
                ariaLabel="Ignore whitespace"
              />
            </SettingsRow>
          </SettingsGroup>

          <SettingsGroup title="Network">
            <SettingsRow
              label="Prune on fetch"
              hint="Fetch also forgets remote branches that are gone from the remote. Fetch and prune in the header's Fetch menu does it once."
            >
              <Switch checked={values.pruneOnFetch} onChange={setPruneOnFetch} ariaLabel="Prune on fetch" />
            </SettingsRow>
            <SettingsRow
              label="Background fetch"
              hint={
                values.autoFetchManaged
                  ? 'Managed by your organization.'
                  : 'Off by default. When off, a remote is only contacted on an explicit fetch, pull or push.'
              }
            >
              <select
                className="settings-field"
                value={values.autoFetchIntervalMinutes}
                onChange={(e) => setAutoFetchIntervalMinutes(Number(e.target.value))}
                aria-label="Background fetch"
                disabled={values.autoFetchManaged}
              >
                {AUTO_FETCH_OPTIONS.map((o) => (
                  <option key={o.minutes} value={o.minutes}>
                    {o.label}
                  </option>
                ))}
              </select>
            </SettingsRow>
          </SettingsGroup>

          {hostIntegration && (
            <SettingsGroup title="Integrations">
              <IntegrationsSection repoPath={repoPath} onChanged={onHostsChanged} />
              <TrackersSection onChanged={onHostsChanged} />
            </SettingsGroup>
          )}

          <SettingsGroup title="Identity colors">
            <IdentityColorSettings />
          </SettingsGroup>

          <SettingsGroup title="Support">
            {updateCheck && <UpdateSection />}
            <DiagnosticsSection />
          </SettingsGroup>
        </div>
      </div>
    </Modal>
  )
}

export default SettingsPanel
