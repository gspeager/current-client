import { useEffect, useRef, useState, type KeyboardEvent as ReactKeyboardEvent } from 'react'

const FOCUSABLE = 'a[href], button:not([disabled]), input:not([disabled]), select, summary, [tabindex]'

interface RovingOptions {
  // Selects the row as the arrow keys reach it, as in a commit or file list.
  selectOnMove?: boolean
  // Left and right move between rows, as in a row of tabs.
  horizontal?: boolean
  // A virtualized list scrolls here, so Home and End reach rows not yet drawn.
  scrollToEdge?: (edge: 'first' | 'last') => void
  // Scrolls to the row after or before this one where it isn't drawn yet, and
  // answers whether it did. Next is the nearest one that is drawn.
  scrollPast?: (row: HTMLElement, forward: boolean, next?: HTMLElement) => boolean
  // For rows whose order on screen isn't their order in the page.
  compareRows?: (a: HTMLElement, b: HTMLElement) => number
}

function focusTarget(row: HTMLElement): HTMLElement | null {
  if (row.matches(FOCUSABLE) || row.hasAttribute('tabindex')) return row
  return row.querySelector<HTMLElement>('.file-row-path') ?? row.querySelector<HTMLElement>(FOCUSABLE)
}

// A list is one Tab stop: arrow keys, Home and End move between its rows, and
// left and right reach the controls inside a row.
export function useRovingRows(
  rowSelector: string,
  { selectOnMove = false, horizontal = false, scrollToEdge, scrollPast, compareRows }: RovingOptions = {},
) {
  const [container, setContainer] = useState<HTMLElement | null>(null)
  const scrollEdge = useRef(scrollToEdge)
  scrollEdge.current = scrollToEdge
  const past = useRef(scrollPast)
  past.current = scrollPast
  const compare = useRef(compareRows)
  compare.current = compareRows

  useEffect(() => {
    if (!container) return
    let current: HTMLElement | null = null
    // A row that took focus with it when it went, such as a file moving from
    // Unstaged to Staged: its place in the list gets the focus back.
    let lostAt: number | null = null
    let lostRow: HTMLElement | null = null
    container.dataset.roving = ''

    const rowsOf = () => {
      const rows = [...container.querySelectorAll<HTMLElement>(rowSelector)]
      return compare.current ? rows.sort(compare.current) : rows
    }

    const sync = () => {
      const rows = rowsOf()
      const lostHere = lostRow && rows[lostAt ?? -1] === lostRow && lostRow.isConnected
      if (lostAt !== null && !lostHere && document.activeElement === document.body && rows.length > 0) {
        current = rows[Math.min(lostAt, rows.length - 1)]
        lostAt = null
        focusTarget(current)?.focus()
      }
      const focused = rows.find((row) => row.contains(document.activeElement))
      const pinned = rows.find((row) => row.getAttribute('aria-current'))
      const active =
        (focused ?? pinned ?? (current && rows.includes(current)))
          ? (focused ?? pinned ?? current)
          : (container.querySelector<HTMLElement>(
              `${rowSelector}[aria-selected='true'], ${rowSelector}[class*='-selected']`,
            ) ?? rows[0])
      const stop = active ? focusTarget(active) : null
      for (const row of rows) {
        const targets = [...(row.matches(FOCUSABLE) ? [row] : []), ...row.querySelectorAll<HTMLElement>(FOCUSABLE)]
        for (const el of targets) {
          const tabIndex = el === stop ? '0' : '-1'
          if (el.getAttribute('tabindex') !== tabIndex) el.setAttribute('tabindex', tabIndex)
        }
      }
    }

    const moveTo = (row: HTMLElement) => {
      current = row
      sync()
      focusTarget(row)?.focus()
      row.scrollIntoView?.({ block: 'nearest' })
      if (!selectOnMove) return
      const target = focusTarget(row)
      if (target && target !== row && target.classList.contains('file-row-path')) target.click()
      else row.click()
    }

    const onceDrawn = (move: () => void) => requestAnimationFrame(() => requestAnimationFrame(move))

    const onKeyDown = (e: KeyboardEvent) => {
      const el = e.target as HTMLElement
      if (e.altKey || e.metaKey || e.ctrlKey) return
      if (el.matches('input:not([type=checkbox]), textarea, select')) return
      const row = el.closest<HTMLElement>(rowSelector)
      if (!row || !container.contains(row)) return
      const rows = rowsOf()
      const index = rows.indexOf(row)
      const [forward, back] = horizontal ? ['ArrowRight', 'ArrowLeft'] : ['ArrowDown', 'ArrowUp']
      if ((e.key === 'Home' || e.key === 'End') && scrollEdge.current) {
        e.preventDefault()
        const last = e.key === 'End'
        scrollEdge.current(last ? 'last' : 'first')
        onceDrawn(() => {
          const drawn = rowsOf()
          const row = last ? drawn[drawn.length - 1] : drawn[0]
          if (row) moveTo(row)
        })
        return
      }
      if (e.key === forward || e.key === back || e.key === 'Home' || e.key === 'End') {
        const next =
          e.key === forward
            ? rows[index + 1]
            : e.key === back
              ? rows[index - 1]
              : e.key === 'Home'
                ? rows[0]
                : rows[rows.length - 1]
        e.preventDefault()
        if ((e.key === forward || e.key === back) && past.current?.(row, e.key === forward, next)) {
          onceDrawn(() => {
            const drawn = rowsOf()
            const at = drawn.indexOf(row)
            const beyond = drawn[at + (e.key === forward ? 1 : -1)]
            if (at >= 0 && beyond) moveTo(beyond)
          })
        } else if (next) {
          moveTo(next)
        }
        return
      }
      if (!horizontal && (e.key === 'ArrowRight' || e.key === 'ArrowLeft')) {
        const inside = [...(row.matches(FOCUSABLE) ? [row] : []), ...row.querySelectorAll<HTMLElement>(FOCUSABLE)]
        const at = inside.indexOf(el)
        const next = inside[at + (e.key === 'ArrowRight' ? 1 : -1)]
        if (at >= 0 && next) {
          e.preventDefault()
          next.focus()
        }
      }
    }

    // A click elsewhere leaves the row on purpose; a virtualized list may drop it later.
    let clicking = false
    const onPointerDown = () => {
      clicking = true
      setTimeout(() => (clicking = false))
    }

    const onFocusOut = (e: FocusEvent) => {
      const row = (e.target as HTMLElement).closest<HTMLElement>(rowSelector)
      lostAt = row && !e.relatedTarget && !clicking ? rowsOf().indexOf(row) : null
      lostRow = row
    }

    const onFocusIn = (e: FocusEvent) => {
      lostAt = null
      const row = (e.target as HTMLElement).closest<HTMLElement>(rowSelector)
      if (row && container.contains(row) && row !== current) {
        current = row
        sync()
      }
    }

    sync()
    const observer = new MutationObserver(sync)
    observer.observe(container, { childList: true, subtree: true, attributes: true, attributeFilter: ['aria-current'] })
    container.addEventListener('keydown', onKeyDown)
    container.addEventListener('focusin', onFocusIn)
    container.addEventListener('focusout', onFocusOut)
    document.addEventListener('pointerdown', onPointerDown, true)
    document.addEventListener('mousedown', onPointerDown, true)
    return () => {
      observer.disconnect()
      container.removeEventListener('keydown', onKeyDown)
      container.removeEventListener('focusin', onFocusIn)
      container.removeEventListener('focusout', onFocusOut)
      document.removeEventListener('pointerdown', onPointerDown, true)
      document.removeEventListener('mousedown', onPointerDown, true)
    }
  }, [container, rowSelector, selectOnMove, horizontal])

  return setContainer
}

// From a list's filter field, the down arrow goes on into the list.
export function arrowIntoList(e: ReactKeyboardEvent<HTMLElement>, rowSelector = '[data-roving] [tabindex="0"]') {
  if (e.key !== 'ArrowDown') return
  const target = e.currentTarget.closest('[data-pane]')?.querySelector<HTMLElement>(rowSelector)
  if (target) {
    e.preventDefault()
    target.focus()
  }
}
