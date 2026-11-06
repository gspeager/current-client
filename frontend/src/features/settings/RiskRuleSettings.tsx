import Switch from '../../components/forms/Switch'
import SettingsRow from './SettingsRow'

const RULES = [
  { id: 'secret', label: 'Secret-looking strings' },
  { id: 'conflict-marker', label: 'Conflict markers' },
  { id: 'large-file', label: 'Files over 10 MB' },
  { id: 'lockfile', label: 'Lockfile out of step with its manifest' },
  { id: 'generated', label: 'Sweeping changes to generated files' },
]

interface RiskRuleSettingsProps {
  off: string[]
  onChange: (off: string[]) => void
}

function RiskRuleSettings({ off, onChange }: RiskRuleSettingsProps) {
  return (
    <SettingsRow
      label="Staged changes report"
      hint="Listed above the commit message. Findings warn and never block a commit. Everything is checked on this computer, and a matched value is never shown in full."
      stacked
    >
      <div className="settings-risk-rules">
        {RULES.map((rule) => (
          <Switch
            key={rule.id}
            label={rule.label}
            checked={!off.includes(rule.id)}
            onChange={(on) => onChange(on ? off.filter((id) => id !== rule.id) : [...off, rule.id])}
          />
        ))}
      </div>
    </SettingsRow>
  )
}

export default RiskRuleSettings
