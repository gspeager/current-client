import type { ReactNode } from 'react'
import { ariaShortcut, shortcutLabel } from '../../lib/shortcuts'
import './Tab.scss'

interface TabProps {
  active: boolean
  onClick: () => void
  children: ReactNode
  // The accessible name, when the children's text would not read as one.
  label?: string
  // A view that is off: dimmed and labelled off, with why in its tooltip; a click still
  // goes somewhere, such as the setting that turns it on.
  off?: string
  // Such as mod+1; shown in the tooltip and given to assistive tech.
  shortcut?: string
}

function Tab({ active, onClick, children, label, off, shortcut }: TabProps) {
  return (
    <button
      type="button"
      className={active ? 'tab tab-active' : off ? 'tab tab-off' : 'tab'}
      onClick={onClick}
      aria-label={label}
      aria-current={active ? 'page' : undefined}
      title={off ?? (shortcut && shortcutLabel(shortcut))}
      aria-keyshortcuts={shortcut && ariaShortcut(shortcut)}
    >
      {children}
      {off && <span className="tab-off-label">off</span>}
    </button>
  )
}

export default Tab
