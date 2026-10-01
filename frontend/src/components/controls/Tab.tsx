import type { ReactNode } from 'react'
import './Tab.scss'

interface TabProps {
  active: boolean
  onClick: () => void
  children: ReactNode
}

function Tab({ active, onClick, children }: TabProps) {
  return (
    <button type="button" className={active ? 'tab tab-active' : 'tab'} onClick={onClick}>
      {children}
    </button>
  )
}

export default Tab
