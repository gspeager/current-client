import type { LucideIcon } from 'lucide-react'
import './SegmentedControl.scss'

interface SegmentedControlOption<T extends string> {
  value: T
  label: string
  icon?: LucideIcon
}

interface SegmentedControlProps<T extends string> {
  value: T
  onChange: (value: T) => void
  options: SegmentedControlOption<T>[]
}

function SegmentedControl<T extends string>({ value, onChange, options }: SegmentedControlProps<T>) {
  return (
    <div className="segmented-control">
      {options.map((option) => {
        const Icon = option.icon
        return (
          <button
            key={option.value}
            type="button"
            className={
              value === option.value
                ? 'segmented-control-option segmented-control-option-active'
                : 'segmented-control-option'
            }
            onClick={() => onChange(option.value)}
          >
            {Icon && <Icon size={14} strokeWidth={1.75} />}
            {option.label}
          </button>
        )
      })}
    </div>
  )
}

export default SegmentedControl
