import type { ReactNode } from 'react'
import { ChevronDown, ChevronRight } from 'lucide-react'
import './SidebarSection.scss'

interface SidebarSectionProps {
  label: string
  // Shown beside the label, open or closed; absent while unknown.
  count?: number
  open: boolean
  onToggle: () => void
  // Header controls that stay reachable while the section is closed.
  actions?: ReactNode
  // Shown under the header whether or not the section is open, such as an add form.
  pinned?: ReactNode
  children: ReactNode
}

function SidebarSection({ label, count, open, onToggle, actions, pinned, children }: SidebarSectionProps) {
  return (
    <>
      <div className="sidebar-section-header">
        <button
          type="button"
          className="sidebar-section-toggle"
          onClick={onToggle}
          aria-expanded={open}
          aria-label={count === undefined ? label : `${label}, ${count}`}
        >
          {open ? (
            <ChevronDown size={14} strokeWidth={1.5} className="sidebar-section-chevron" />
          ) : (
            <ChevronRight size={14} strokeWidth={1.5} className="sidebar-section-chevron" />
          )}
          <span className="sidebar-section-label">{label}</span>
          {count !== undefined && <span className="sidebar-section-count">{count}</span>}
        </button>
        {actions && <span className="sidebar-section-actions">{actions}</span>}
      </div>
      {pinned}
      {open && children}
    </>
  )
}

export default SidebarSection
