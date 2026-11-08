import { ChevronDown } from 'lucide-react'
import { useEffect, useRef, useState } from 'react'
import './ScrollCue.scss'

// The cue's own height and the gap above it, which are blank once the rows have ended.
const END_SPACE = 40

// Goes last inside a scroller and says when rows continue past its bottom edge.
// Arrow keys already reach those rows, so it's for the pointer alone.
function ScrollCue() {
  const cue = useRef<HTMLButtonElement>(null)
  const [more, setMore] = useState(false)

  useEffect(() => {
    const scroller = cue.current?.parentElement
    if (!scroller) return
    const update = () => setMore(scroller.scrollHeight - scroller.scrollTop - scroller.clientHeight > END_SPACE)
    update()
    scroller.addEventListener('scroll', update, { passive: true })
    const observer = typeof ResizeObserver === 'undefined' ? null : new ResizeObserver(update)
    observer?.observe(scroller)
    for (const child of scroller.children) observer?.observe(child)
    return () => {
      scroller.removeEventListener('scroll', update)
      observer?.disconnect()
    }
  }, [])

  return (
    <button
      ref={cue}
      type="button"
      className="scroll-cue"
      data-more={more}
      tabIndex={-1}
      aria-hidden="true"
      onClick={(e) => {
        const scroller = e.currentTarget.parentElement
        scroller?.scrollBy({ top: scroller.clientHeight * 0.8 })
      }}
    >
      <ChevronDown size={12} strokeWidth={1.75} />
      More below
    </button>
  )
}

export default ScrollCue
