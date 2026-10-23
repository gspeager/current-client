import './Switch.scss'

interface SwitchProps {
  checked: boolean
  onChange: (checked: boolean) => void
  label?: string
  ariaLabel?: string
  disabled?: boolean
}

function Switch({ checked, onChange, label, ariaLabel, disabled = false }: SwitchProps) {
  return (
    <label className="switch">
      <input
        type="checkbox"
        checked={checked}
        onChange={(e) => onChange(e.target.checked)}
        aria-label={ariaLabel ?? label}
        disabled={disabled}
      />
      <span className="switch-track">
        <span className="switch-knob" />
      </span>
      {label && <span className="switch-label">{label}</span>}
    </label>
  )
}

export default Switch
