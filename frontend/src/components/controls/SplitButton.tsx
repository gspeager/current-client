import { useState, type ReactNode } from 'react'
import { ChevronDown } from 'lucide-react'
import { useDismissibleDetails } from '../../lib/useDismissibleDetails'
import './SplitButton.scss'

interface SplitButtonProps {
  children: ReactNode
  onClick: () => void
  disabled?: boolean
  menuLabel: string
  menu: ReactNode
  // The menu opens above when the button sits at the bottom of its pane.
  placement?: 'above' | 'below'
  // Controlled when given, so a menu action can close the menu.
  open?: boolean
  onOpenChange?: (open: boolean) => void
}

function SplitButton({
  children,
  onClick,
  disabled = false,
  menuLabel,
  menu,
  placement = 'below',
  open,
  onOpenChange,
}: SplitButtonProps) {
  const [ownOpen, setOwnOpen] = useState(false)
  const isOpen = open ?? ownOpen
  const setOpen = onOpenChange ?? setOwnOpen
  const ref = useDismissibleDetails()

  return (
    <div className="split-button">
      <button type="button" className="split-button-main" onClick={onClick} disabled={disabled}>
        {children}
      </button>
      <details ref={ref} className="split-button-more" open={isOpen} onToggle={(e) => setOpen(e.currentTarget.open)}>
        <summary className="split-button-toggle" aria-label={menuLabel} title={menuLabel}>
          <ChevronDown size={14} strokeWidth={1.75} />
        </summary>
        <div className={`split-button-menu split-button-menu-${placement}`}>{menu}</div>
      </details>
    </div>
  )
}

export default SplitButton
