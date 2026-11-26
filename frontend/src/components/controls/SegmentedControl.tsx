import clsx from 'clsx'
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
  // As wide as its container, the options sharing the width.
  fill?: boolean
}

function SegmentedControl<T extends string>({ value, onChange, options, fill = false }: SegmentedControlProps<T>) {
  return (
    <div className={clsx('segmented-control', fill && 'segmented-control-fill')}>
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
            aria-pressed={value === option.value}
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
