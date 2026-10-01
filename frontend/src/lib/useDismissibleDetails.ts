import { useRef } from 'react'
import { useEventListener, useOnClickOutside } from 'usehooks-ts'

// Closes a <details> menu on Escape or a click outside it. Setting .open fires
// the element's toggle event, so a controlled menu's onToggle still sees it.
export function useDismissibleDetails() {
  const ref = useRef<HTMLDetailsElement>(null)
  const close = () => {
    if (ref.current?.open) ref.current.open = false
  }
  useOnClickOutside(ref, close)
  useEventListener('keydown', (e) => {
    if (e.key === 'Escape') close()
  })
  return ref
}
