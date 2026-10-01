import PathText from '../../components/git/PathText'
import SettingsRow from './SettingsRow'

interface PathSettingRowProps {
  label: string
  hint: string
  path: string
  // Shown in place of a path when none is set.
  fallback: string
  error?: string | null
  onChoose: () => void
  onReset?: () => void
  resetLabel?: string
}

function PathSettingRow({ label, hint, path, fallback, error, onChoose, onReset, resetLabel }: PathSettingRowProps) {
  return (
    <SettingsRow label={label} hint={hint} error={error}>
      {path ? (
        <PathText path={path} className="settings-path" />
      ) : (
        <span className="settings-path settings-path-unset">{fallback}</span>
      )}
      <div className="settings-actions">
        <button
          type="button"
          className="settings-button"
          onClick={onChoose}
          aria-label={`Choose ${label.toLowerCase()}`}
        >
          Choose…
        </button>
        {path && onReset && (
          <button type="button" className="settings-button" onClick={onReset}>
            {resetLabel}
          </button>
        )}
      </div>
    </SettingsRow>
  )
}

export default PathSettingRow
