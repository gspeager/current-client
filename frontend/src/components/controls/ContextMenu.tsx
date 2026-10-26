import { useEffect, useRef, useState, type KeyboardEvent, type MouseEvent } from 'react'
import './ContextMenu.scss'

export interface ContextMenuItem {
  label: string
  onClick: () => void
  destructive?: boolean
  disabled?: boolean
}

export interface ContextMenuState {
  x: number
  y: number
  items: ContextMenuItem[]
  // With 'end', x is the menu's right edge, for a menu under a control's
  // right-hand chevron.
  align?: 'start' | 'end'
}

// Shift+F10 and the context-menu key open a row's menu from the keyboard.
export function isMenuKey(e: KeyboardEvent): boolean {
  return e.key === 'ContextMenu' || (e.shiftKey && e.key === 'F10')
}

// Where a menu opens: at the pointer, or under the element when a key or a
// keyboard-pressed button opened it, which leaves no pointer position.
export function menuAnchor(e: MouseEvent | KeyboardEvent): { x: number; y: number } {
  if ('clientX' in e && (e.clientX !== 0 || e.clientY !== 0)) return { x: e.clientX, y: e.clientY }
  const rect = (e.currentTarget as HTMLElement).getBoundingClientRect()
  return { x: rect.left, y: rect.bottom }
}

interface ContextMenuProps {
  state: ContextMenuState
  onClose: () => void
}

// Clamped to the viewport once its rendered size is known. It takes focus,
// moves with the arrow keys, and gives focus back to what opened it.
function ContextMenu({ state, onClose }: ContextMenuProps) {
  const { x, y, items, align = 'start' } = state
  const ref = useRef<HTMLDivElement>(null)
  const opener = useRef(document.activeElement)
  const [position, setPosition] = useState({ left: x, top: y, visible: false })

  useEffect(() => {
    const el = ref.current
    if (!el) return
    const { width, height } = el.getBoundingClientRect()
    const left = align === 'end' ? x - width : x
    setPosition({
      left: Math.max(4, Math.min(left, window.innerWidth - width - 4)),
      top: Math.min(y, window.innerHeight - height - 4),
      visible: true,
    })
  }, [x, y, align])

  const enabledItems = () => [...(ref.current?.querySelectorAll<HTMLButtonElement>('button:not(:disabled)') ?? [])]

  useEffect(() => {
    if (position.visible) enabledItems()[0]?.focus()
  }, [position.visible])

  useEffect(
    () => () => {
      if (opener.current instanceof HTMLElement && opener.current.isConnected) opener.current.focus()
    },
    [],
  )

  useEffect(() => {
    const onPointerDown = (e: globalThis.MouseEvent) => {
      if (ref.current && !ref.current.contains(e.target as Node)) onClose()
    }
    const onKeyDown = (e: globalThis.KeyboardEvent) => {
      if (e.key === 'Escape') onClose()
    }
    window.addEventListener('mousedown', onPointerDown)
    window.addEventListener('keydown', onKeyDown)
    return () => {
      window.removeEventListener('mousedown', onPointerDown)
      window.removeEventListener('keydown', onKeyDown)
    }
  }, [onClose])

  const moveFocus = (e: KeyboardEvent) => {
    const buttons = enabledItems()
    const at = buttons.indexOf(document.activeElement as HTMLButtonElement)
    const next = { ArrowDown: at + 1, ArrowUp: at - 1, Home: 0, End: buttons.length - 1 }[e.key]
    if (next === undefined || buttons.length === 0) return
    e.preventDefault()
    buttons[(next + buttons.length) % buttons.length].focus()
  }

  return (
    <div
      ref={ref}
      className="context-menu"
      onKeyDown={moveFocus}
      style={{ left: position.left, top: position.top, visibility: position.visible ? 'visible' : 'hidden' }}
    >
      {items.map((item) => (
        <button
          key={item.label}
          type="button"
          className={item.destructive ? 'context-menu-item context-menu-item-destructive' : 'context-menu-item'}
          disabled={item.disabled}
          onClick={() => {
            onClose()
            item.onClick()
          }}
        >
          {item.label}
        </button>
      ))}
    </div>
  )
}

export default ContextMenu
