import { useEffect, useRef, useState } from 'react'
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
}

interface ContextMenuProps {
  state: ContextMenuState
  onClose: () => void
}

// Clamped to the viewport once its rendered size is known.
function ContextMenu({ state, onClose }: ContextMenuProps) {
  const { x, y, items } = state
  const ref = useRef<HTMLDivElement>(null)
  const [position, setPosition] = useState({ left: x, top: y, visible: false })

  useEffect(() => {
    const el = ref.current
    if (!el) return
    const { width, height } = el.getBoundingClientRect()
    setPosition({
      left: Math.min(x, window.innerWidth - width - 4),
      top: Math.min(y, window.innerHeight - height - 4),
      visible: true,
    })
  }, [x, y])

  useEffect(() => {
    const onPointerDown = (e: MouseEvent) => {
      if (ref.current && !ref.current.contains(e.target as Node)) onClose()
    }
    const onKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape') onClose()
    }
    window.addEventListener('mousedown', onPointerDown)
    window.addEventListener('keydown', onKeyDown)
    return () => {
      window.removeEventListener('mousedown', onPointerDown)
      window.removeEventListener('keydown', onKeyDown)
    }
  }, [onClose])

  return (
    <div
      ref={ref}
      className="context-menu"
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
