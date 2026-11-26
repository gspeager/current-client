import clsx from 'clsx'
import './Switch.scss'

interface SwitchProps {
  checked: boolean
  onChange: (checked: boolean) => void
  label?: string
  ariaLabel?: string
  disabled?: boolean
  // The label first and the toggle at the far end, like a settings row's control.
  trailing?: boolean
}

function Switch({ checked, onChange, label, ariaLabel, disabled = false, trailing = false }: SwitchProps) {
  return (
    <label className={clsx('switch', trailing && 'switch-trailing')}>
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
