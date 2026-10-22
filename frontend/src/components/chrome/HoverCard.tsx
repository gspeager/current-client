import {
  cloneElement,
  isValidElement,
  useEffect,
  useRef,
  useState,
  type FocusEvent,
  type MouseEvent,
  type ReactElement,
  type ReactNode,
} from 'react'
import { createPortal } from 'react-dom'
import { useCurrentClientRoot } from '../../lib/currentClientRoot'
import './HoverCard.scss'

const SHOW_DELAY_MS = 400
const GAP = 6
const WIDTH = 320

interface AnchorProps {
  onMouseEnter?: (e: MouseEvent<HTMLElement>) => void
  onMouseLeave?: (e: MouseEvent<HTMLElement>) => void
  onFocus?: (e: FocusEvent<HTMLElement>) => void
  onBlur?: (e: FocusEvent<HTMLElement>) => void
}

interface HoverCardProps {
  // The card's content; null shows no card.
  card: ReactNode
  // The anchor, which gets the pointer and focus handlers; no wrapper
  // element is added, so its accessible name is untouched.
  children: ReactElement<AnchorProps>
}

// A card that appears beside its anchor after a short pause of the pointer
// or on keyboard focus, and goes on mouse-out, blur or Escape. It shows only
// what the anchor already has; opening it never fetches.
function HoverCard({ card, children }: HoverCardProps) {
  const timer = useRef<number | undefined>(undefined)
  const [position, setPosition] = useState<{ top: number; left: number } | null>(null)
  const { element: root } = useCurrentClientRoot()

  const show = (e: MouseEvent<HTMLElement> | FocusEvent<HTMLElement>) => {
    const rect = e.currentTarget.getBoundingClientRect()
    window.clearTimeout(timer.current)
    timer.current = window.setTimeout(() => {
      setPosition({ top: rect.bottom + GAP, left: Math.max(GAP, Math.min(rect.left, window.innerWidth - WIDTH - GAP)) })
    }, SHOW_DELAY_MS)
  }

  const hide = () => {
    window.clearTimeout(timer.current)
    setPosition(null)
  }

  useEffect(() => () => window.clearTimeout(timer.current), [])

  useEffect(() => {
    if (!position) return
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') hide()
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [position])

  if (!card || !isValidElement(children)) return children

  const anchor = cloneElement(children, {
    onMouseEnter: (e: MouseEvent<HTMLElement>) => {
      children.props.onMouseEnter?.(e)
      show(e)
    },
    onMouseLeave: (e: MouseEvent<HTMLElement>) => {
      children.props.onMouseLeave?.(e)
      hide()
    },
    onFocus: (e: FocusEvent<HTMLElement>) => {
      children.props.onFocus?.(e)
      show(e)
    },
    onBlur: (e: FocusEvent<HTMLElement>) => {
      children.props.onBlur?.(e)
      hide()
    },
  })

  return (
    <>
      {anchor}
      {position &&
        createPortal(
          <div className="hover-card" role="tooltip" style={{ top: position.top, left: position.left }}>
            {card}
          </div>,
          root ?? document.body,
        )}
    </>
  )
}

export default HoverCard
