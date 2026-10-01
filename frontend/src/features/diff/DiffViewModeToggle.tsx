import { Columns2, Rows2 } from 'lucide-react'
import SegmentedControl from '../../components/controls/SegmentedControl'

export type DiffViewMode = 'split' | 'unified'

interface DiffViewModeToggleProps {
  value: DiffViewMode
  onChange: (mode: DiffViewMode) => void
}

function DiffViewModeToggle({ value, onChange }: DiffViewModeToggleProps) {
  return (
    <SegmentedControl
      value={value}
      onChange={onChange}
      options={[
        { value: 'split', label: 'Split', icon: Columns2 },
        { value: 'unified', label: 'Unified', icon: Rows2 },
      ]}
    />
  )
}

export default DiffViewModeToggle
