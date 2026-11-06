import { useEffect, useState } from 'react'
import { createPortal } from 'react-dom'
import './TitleTooltips.scss'

const SHOW_DELAY_MS = 350
const GAP = 6

interface Tip {
  text: string
  x: number
  y: number
  above: boolean
}

// Draws every `title` as the app's own tooltip, on hover and on keyboard focus.
// The attribute comes off while the tooltip shows, so the OS doesn't draw its
// own as well, and goes back afterwards for React to keep managing.
function TitleTooltips({ root }: { root: HTMLElement }) {
  const [tip, setTip] = useState<Tip | null>(null)

  useEffect(() => {
    let timer = 0
    let owner: HTMLElement | null = null
    let text = ''

    const hide = () => {
      window.clearTimeout(timer)
      if (owner && text && !owner.hasAttribute('title')) owner.setAttribute('title', text)
      owner = null
      text = ''
      setTip(null)
    }

    const show = (el: HTMLElement) => {
      if (el === owner) return
      hide()
      const value = el.getAttribute('title')
      if (!value) return
      owner = el
      text = value
      el.removeAttribute('title')
      timer = window.setTimeout(() => {
        const box = el.getBoundingClientRect()
        const above = box.bottom + 48 > window.innerHeight
        setTip({
          text: value,
          x: Math.min(Math.max(box.left + box.width / 2, 8), window.innerWidth - 8),
          y: above ? box.top - GAP : box.bottom + GAP,
          above,
        })
      }, SHOW_DELAY_MS)
    }

    const onOver = (e: MouseEvent) => {
      const el = (e.target as Element).closest<HTMLElement>('[title]')
      if (el) show(el)
      else if (owner && !owner.contains(e.target as Node)) hide()
    }
    const onFocus = (e: FocusEvent) => {
      const el = e.target as HTMLElement
      if (el.matches?.(':focus-visible') && el.hasAttribute('title')) show(el)
    }
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape' && owner) hide()
    }

    root.addEventListener('mouseover', onOver)
    root.addEventListener('focusin', onFocus)
    root.addEventListener('focusout', hide)
    root.addEventListener('mousedown', hide)
    root.addEventListener('keydown', onKey)
    root.addEventListener('wheel', hide, { passive: true })
    return () => {
      hide()
      root.removeEventListener('mouseover', onOver)
      root.removeEventListener('focusin', onFocus)
      root.removeEventListener('focusout', hide)
      root.removeEventListener('mousedown', hide)
      root.removeEventListener('keydown', onKey)
      root.removeEventListener('wheel', hide)
    }
  }, [root])

  if (!tip) return null
  return createPortal(
    <div
      role="tooltip"
      className={tip.above ? 'title-tooltip title-tooltip-above' : 'title-tooltip'}
      style={{ left: tip.x, top: tip.y }}
    >
      {tip.text}
    </div>,
    root,
  )
}

export default TitleTooltips
