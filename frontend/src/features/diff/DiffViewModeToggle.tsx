import { Columns2, Rows2 } from 'lucide-react'
import SegmentedControl from '../../components/controls/SegmentedControl'
import { NARROW_SPLIT_WIDTH, useWindowNarrowerThan } from '../../lib/useWindowNarrowerThan'

export type DiffViewMode = 'split' | 'unified'

interface DiffViewModeToggleProps {
  value: DiffViewMode
  onChange: (mode: DiffViewMode) => void
}

function DiffViewModeToggle({ value, onChange }: DiffViewModeToggleProps) {
  const unifiedOnly = useWindowNarrowerThan(NARROW_SPLIT_WIDTH)
  if (unifiedOnly) return null
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
