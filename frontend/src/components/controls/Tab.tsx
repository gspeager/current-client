import type { ReactNode } from 'react'
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
}

function Tab({ active, onClick, children, label, off }: TabProps) {
  return (
    <button
      type="button"
      className={active ? 'tab tab-active' : off ? 'tab tab-off' : 'tab'}
      onClick={onClick}
      aria-label={label}
      title={off}
    >
      {children}
      {off && <span className="tab-off-label">off</span>}
    </button>
  )
}

export default Tab
