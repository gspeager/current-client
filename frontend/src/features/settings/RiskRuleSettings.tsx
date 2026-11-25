import Switch from '../../components/forms/Switch'
import { RISK_RULES } from '../../lib/riskRules'
import SettingsRow from './SettingsRow'

interface RiskRuleSettingsProps {
  off: string[]
  onChange: (off: string[]) => void
}

function RiskRuleSettings({ off, onChange }: RiskRuleSettingsProps) {
  return (
    <SettingsRow
      label="Worth a second look"
      hint="Listed above the commit message for what's staged, and in Read through for a whole branch. Nothing is blocked. Everything is checked on this computer, and a matched secret is never shown in full."
      stacked
    >
      <div className="settings-risk-rules">
        {RISK_RULES.map((rule) => (
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
