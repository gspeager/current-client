import { Check } from 'lucide-react'
import './Checkbox.scss'

interface CheckboxProps {
  checked: boolean
  onChange: (checked: boolean) => void
  label?: string
  ariaLabel?: string
  disabled?: boolean
  onClick?: (e: React.MouseEvent) => void
}

function Checkbox({ checked, onChange, label, ariaLabel, disabled = false, onClick }: CheckboxProps) {
  return (
    <label className="checkbox" onClick={onClick}>
      <input
        type="checkbox"
        checked={checked}
        disabled={disabled}
        onChange={(e) => onChange(e.target.checked)}
        aria-label={ariaLabel}
      />
      <span className="checkbox-box">{checked && <Check size={12} strokeWidth={3} />}</span>
      {label && <span className="checkbox-label">{label}</span>}
    </label>
  )
}

export default Checkbox
