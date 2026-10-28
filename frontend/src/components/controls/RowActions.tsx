import { useState, type KeyboardEvent, type MouseEvent, type ReactNode } from 'react'
import type { LucideIcon } from 'lucide-react'
import ContextMenu, { menuAnchor, type ContextMenuState } from './ContextMenu'

export interface RowAction {
  // The menu item's text and the button's tooltip.
  label: string
  // The button's accessible name, which names the row.
  ariaLabel: string
  icon: LucideIcon
  run: () => void
  destructive?: boolean
  disabled?: boolean
  // In the menu only: an action with no room on the row.
  menuOnly?: boolean
}

// A sidebar row's actions as hover buttons, from the same list its context
// menu uses, so nothing is reachable only by hovering.
export function RowActionButtons({ actions, className }: { actions: RowAction[]; className: string }) {
  return (
    <span className={className}>
      {actions
        .filter((a) => !a.menuOnly)
        .map(({ label, ariaLabel, icon: Icon, run, disabled }) => (
          <button
            key={label}
            type="button"
            onClick={(e) => {
              e.stopPropagation()
              run()
            }}
            disabled={disabled}
            aria-label={ariaLabel}
            title={label}
          >
            <Icon size={16} strokeWidth={1.75} />
          </button>
        ))}
    </span>
  )
}

export function useRowMenu(): {
  open: (actions: RowAction[]) => (e: MouseEvent | KeyboardEvent) => void
  menu: ReactNode
} {
  const [state, setState] = useState<ContextMenuState | null>(null)
  const open = (actions: RowAction[]) => (e: MouseEvent | KeyboardEvent) => {
    e.preventDefault()
    e.stopPropagation()
    setState({
      ...menuAnchor(e),
      items: actions.map((a) => ({ label: a.label, onClick: a.run, destructive: a.destructive, disabled: a.disabled })),
    })
  }
  return { open, menu: state && <ContextMenu state={state} onClose={() => setState(null)} /> }
}
