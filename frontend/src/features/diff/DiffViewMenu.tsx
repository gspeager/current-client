import { SlidersHorizontal } from 'lucide-react'
import Checkbox from '../../components/forms/Checkbox'
import { useDismissibleDetails } from '../../lib/useDismissibleDetails'
import { DiffWrapToggle } from './diffWrap'
import './DiffViewMenu.scss'

interface DiffViewMenuProps {
  // Present when the diff can ignore whitespace.
  ignoreWhitespace?: boolean
  onIgnoreWhitespaceChange?: (ignore: boolean) => void
}

// The diff header's secondary choices in one menu, so the file path keeps its room.
function DiffViewMenu({ ignoreWhitespace, onIgnoreWhitespaceChange }: DiffViewMenuProps) {
  const ref = useDismissibleDetails()
  return (
    <details ref={ref} className="diff-view-menu">
      <summary className="diff-view-menu-toggle" aria-label="View options" title="View options">
        <SlidersHorizontal size={14} strokeWidth={1.5} />
        View
      </summary>
      <div className="diff-view-menu-items">
        {onIgnoreWhitespaceChange && (
          <Checkbox checked={ignoreWhitespace ?? false} onChange={onIgnoreWhitespaceChange} label="Ignore whitespace" />
        )}
        <DiffWrapToggle />
      </div>
    </details>
  )
}

export default DiffViewMenu
