import type { ReactNode } from 'react'

interface SettingsGroupProps {
  title: string
  children: ReactNode
}

export function SettingsGroup({ title, children }: SettingsGroupProps) {
  return (
    <section className="settings-group" aria-label={title}>
      <h3 className="settings-group-title">{title}</h3>
      <div className="settings-group-rows">{children}</div>
    </section>
  )
}

interface SettingsRowProps {
  label: string
  hint?: ReactNode
  error?: string | null
  // Puts the control under the label, for controls that need the full width.
  stacked?: boolean
  children: ReactNode
}

function SettingsRow({ label, hint, error, stacked = false, children }: SettingsRowProps) {
  return (
    <div className={stacked ? 'settings-row settings-row-stacked' : 'settings-row'}>
      <div className="settings-row-text">
        <span className="settings-row-label">{label}</span>
        {hint && <p className="settings-hint">{hint}</p>}
      </div>
      <div className="settings-row-control">{children}</div>
      {error && <p className="settings-error">{error}</p>}
    </div>
  )
}

export default SettingsRow
