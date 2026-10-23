import type { ReactNode } from 'react'
import './Tab.scss'

interface TabProps {
  active: boolean
  onClick: () => void
  children: ReactNode
  // The accessible name, when the children's text would not read as one.
  label?: string
}

function Tab({ active, onClick, children, label }: TabProps) {
  return (
    <button type="button" className={active ? 'tab tab-active' : 'tab'} onClick={onClick} aria-label={label}>
      {children}
    </button>
  )
}

export default Tab
